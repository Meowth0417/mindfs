package usecase

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	rootfs "mindfs/server/internal/fs"
)

const environmentCommandsMetaPath = "environments/environment.toml"

var environmentCommandsMu sync.Mutex

type EnvironmentCommandIcon string

const (
	EnvironmentCommandIconRun   EnvironmentCommandIcon = "run"
	EnvironmentCommandIconTool  EnvironmentCommandIcon = "tool"
	EnvironmentCommandIconTest  EnvironmentCommandIcon = "test"
	EnvironmentCommandIconOther EnvironmentCommandIcon = "other"
)

type EnvironmentCommand struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Icon      EnvironmentCommandIcon `json:"icon"`
	Command   string                 `json:"command"`
	IsDefault bool                   `json:"is_default"`
}

type environmentCommandsFile struct {
	Version     int
	Name        string
	SetupScript string
	Commands    []EnvironmentCommand
}

type ListEnvironmentCommandsInput struct {
	RootID string
}

type ListEnvironmentCommandsOutput struct {
	Items []EnvironmentCommand `json:"items"`
}

type SaveEnvironmentCommandInput struct {
	RootID     string
	Name       string
	Icon       string
	Command    string
	SetDefault bool
}

type SaveEnvironmentCommandOutput struct {
	Items []EnvironmentCommand `json:"items"`
}

type RunEnvironmentCommandInput struct {
	RootID      string
	CommandID   string
	MakeDefault bool
}

type RunEnvironmentCommandOutput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Target  string `json:"target"`
	Action  string `json:"action"`
}

func (s *Service) ListEnvironmentCommands(_ context.Context, in ListEnvironmentCommandsInput) (ListEnvironmentCommandsOutput, error) {
	root, err := s.getEnvironmentRoot(in.RootID)
	if err != nil {
		return ListEnvironmentCommandsOutput{}, err
	}
	commands, err := readEnvironmentCommands(root)
	if err != nil {
		return ListEnvironmentCommandsOutput{}, err
	}
	return ListEnvironmentCommandsOutput{Items: commands}, nil
}

func (s *Service) SaveEnvironmentCommand(_ context.Context, in SaveEnvironmentCommandInput) (SaveEnvironmentCommandOutput, error) {
	root, err := s.getEnvironmentRoot(in.RootID)
	if err != nil {
		return SaveEnvironmentCommandOutput{}, err
	}
	name := strings.TrimSpace(in.Name)
	commandText := strings.TrimSpace(in.Command)
	if name == "" {
		return SaveEnvironmentCommandOutput{}, errors.New("command name required")
	}
	if commandText == "" {
		return SaveEnvironmentCommandOutput{}, errors.New("command text required")
	}

	environmentCommandsMu.Lock()
	defer environmentCommandsMu.Unlock()

	commands, err := readEnvironmentCommandsUnlocked(root)
	if err != nil {
		return SaveEnvironmentCommandOutput{}, err
	}
	command := EnvironmentCommand{
		ID:        generateEnvironmentCommandID(),
		Name:      name,
		Icon:      normalizeEnvironmentCommandIcon(in.Icon),
		Command:   commandText,
		IsDefault: in.SetDefault || len(commands) == 0,
	}
	commands = append(commands, command)
	if command.IsDefault {
		commands = markEnvironmentCommandDefault(commands, command.ID)
	} else {
		commands = normalizeEnvironmentCommands(commands)
	}
	if err := writeEnvironmentCommandsUnlocked(root, commands); err != nil {
		return SaveEnvironmentCommandOutput{}, err
	}
	return SaveEnvironmentCommandOutput{Items: commands}, nil
}

func (s *Service) RunEnvironmentCommand(ctx context.Context, in RunEnvironmentCommandInput) (RunEnvironmentCommandOutput, error) {
	root, err := s.getEnvironmentRoot(in.RootID)
	if err != nil {
		return RunEnvironmentCommandOutput{}, err
	}
	rootAbs, err := root.RootDir()
	if err != nil {
		return RunEnvironmentCommandOutput{}, err
	}

	environmentCommandsMu.Lock()
	defer environmentCommandsMu.Unlock()

	commands, err := readEnvironmentCommandsUnlocked(root)
	if err != nil {
		return RunEnvironmentCommandOutput{}, err
	}
	command, found := findEnvironmentCommand(commands, strings.TrimSpace(in.CommandID))
	if !found {
		return RunEnvironmentCommandOutput{}, errors.New("command not found")
	}
	if runtimeCommandDefaultNeedsUpdate(commands, command.ID, in.MakeDefault) {
		commands = markEnvironmentCommandDefault(commands, command.ID)
		if err := writeEnvironmentCommandsUnlocked(root, commands); err != nil {
			return RunEnvironmentCommandOutput{}, err
		}
		command, _ = findEnvironmentCommand(commands, command.ID)
	}

	return RunEnvironmentCommandOutput{
		ID:      command.ID,
		Name:    command.Name,
		Command: command.Command,
		Target:  rootAbs,
		Action:  "command",
	}, nil
}

func (s *Service) getEnvironmentRoot(rootID string) (rootfs.RootInfo, error) {
	if err := s.ensureRegistry(); err != nil {
		return rootfs.RootInfo{}, err
	}
	rootID = strings.TrimSpace(rootID)
	if rootID == "" {
		return rootfs.RootInfo{}, errors.New("root required")
	}
	return s.Registry.GetRoot(rootID)
}

func readEnvironmentCommands(root rootfs.RootInfo) ([]EnvironmentCommand, error) {
	environmentCommandsMu.Lock()
	defer environmentCommandsMu.Unlock()
	return readEnvironmentCommandsUnlocked(root)
}

func readEnvironmentCommandsUnlocked(root rootfs.RootInfo) ([]EnvironmentCommand, error) {
	parsed, err := readEnvironmentCommandsFileUnlocked(root)
	if err != nil {
		return nil, err
	}
	return normalizeEnvironmentCommands(parsed.Commands), nil
}

func readEnvironmentCommandsFileUnlocked(root rootfs.RootInfo) (environmentCommandsFile, error) {
	payload, err := root.ReadMetaFile(environmentCommandsMetaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return environmentCommandsFile{}, nil
		}
		return environmentCommandsFile{}, err
	}
	parsed, err := parseEnvironmentCommandsTOML(payload)
	if err != nil {
		return environmentCommandsFile{}, err
	}
	parsed.Commands = normalizeEnvironmentCommands(parsed.Commands)
	return parsed, nil
}

func writeEnvironmentCommandsUnlocked(root rootfs.RootInfo, commands []EnvironmentCommand) error {
	existing, err := readEnvironmentCommandsFileUnlocked(root)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(existing.Name)
	if name == "" {
		name = strings.TrimSpace(root.Name)
	}
	if name == "" {
		name = strings.TrimSpace(root.ID)
	}
	payload := marshalEnvironmentCommandsTOML(environmentCommandsFile{
		Version:     1,
		Name:        name,
		SetupScript: existing.SetupScript,
		Commands:    reorderEnvironmentCommandsForPersistence(normalizeEnvironmentCommands(commands)),
	})
	return root.WriteMetaFile(environmentCommandsMetaPath, payload)
}

func parseEnvironmentCommandsTOML(payload []byte) (environmentCommandsFile, error) {
	lines := strings.Split(strings.ReplaceAll(string(payload), "\r\n", "\n"), "\n")
	parsed := environmentCommandsFile{
		Version: 1,
	}
	var current *EnvironmentCommand
	section := ""
	for index, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimPrefix(rawLine, "\uFEFF"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch line {
		case "[setup]":
			section = "setup"
			current = nil
			continue
		case "[[actions]]", "[[command]]":
			section = "actions"
			parsed.Commands = append(parsed.Commands, EnvironmentCommand{})
			current = &parsed.Commands[len(parsed.Commands)-1]
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return environmentCommandsFile{}, fmt.Errorf("invalid environment.toml line %d", index+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch section {
		case "":
			switch key {
			case "version":
				version, err := strconv.Atoi(value)
				if err != nil {
					return environmentCommandsFile{}, err
				}
				parsed.Version = version
			case "name":
				name, err := parseEnvironmentCommandString(value)
				if err != nil {
					return environmentCommandsFile{}, err
				}
				parsed.Name = name
			}
		case "setup":
			if key != "script" {
				continue
			}
			script, err := parseEnvironmentCommandString(value)
			if err != nil {
				return environmentCommandsFile{}, err
			}
			parsed.SetupScript = script
		case "actions":
			if current == nil {
				continue
			}
			switch key {
			case "id":
				parsedValue, err := parseEnvironmentCommandString(value)
				if err != nil {
					return environmentCommandsFile{}, err
				}
				current.ID = parsedValue
			case "name":
				parsedValue, err := parseEnvironmentCommandString(value)
				if err != nil {
					return environmentCommandsFile{}, err
				}
				current.Name = parsedValue
			case "icon":
				parsedValue, err := parseEnvironmentCommandString(value)
				if err != nil {
					return environmentCommandsFile{}, err
				}
				current.Icon = normalizeEnvironmentCommandIcon(parsedValue)
			case "command":
				parsedValue, err := parseEnvironmentCommandString(value)
				if err != nil {
					return environmentCommandsFile{}, err
				}
				current.Command = parsedValue
			case "default":
				parsedValue, err := strconv.ParseBool(value)
				if err != nil {
					return environmentCommandsFile{}, err
				}
				current.IsDefault = parsedValue
			}
		}
	}
	return parsed, nil
}

func parseEnvironmentCommandString(input string) (string, error) {
	value, err := strconv.Unquote(input)
	if err != nil {
		return "", err
	}
	return value, nil
}

func marshalEnvironmentCommandsTOML(file environmentCommandsFile) []byte {
	var builder strings.Builder
	builder.WriteString("# THIS IS AUTOGENERATED. DO NOT EDIT MANUALLY\n")
	builder.WriteString("version = ")
	builder.WriteString(strconv.Itoa(max(1, file.Version)))
	builder.WriteString("\n")
	builder.WriteString("name = ")
	builder.WriteString(strconv.Quote(strings.TrimSpace(file.Name)))
	builder.WriteString("\n\n[setup]\nscript = ")
	builder.WriteString(strconv.Quote(file.SetupScript))
	builder.WriteString("\n")
	for _, command := range normalizeEnvironmentCommands(file.Commands) {
		builder.WriteString("\n[[actions]]\n")
		builder.WriteString("name = ")
		builder.WriteString(strconv.Quote(command.Name))
		builder.WriteString("\nicon = ")
		builder.WriteString(strconv.Quote(string(normalizeEnvironmentCommandIcon(string(command.Icon)))))
		builder.WriteString("\ncommand = ")
		builder.WriteString(strconv.Quote(command.Command))
		builder.WriteString("\n")
	}
	return []byte(builder.String())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func normalizeEnvironmentCommands(commands []EnvironmentCommand) []EnvironmentCommand {
	normalized := make([]EnvironmentCommand, 0, len(commands))
	defaultIndex := -1
	for index, command := range commands {
		id := strings.TrimSpace(command.ID)
		name := strings.TrimSpace(command.Name)
		commandText := strings.TrimSpace(command.Command)
		if name == "" || commandText == "" {
			continue
		}
		if id == "" {
			id = deriveEnvironmentCommandID(name, commandText, index)
		}
		next := EnvironmentCommand{
			ID:        id,
			Name:      name,
			Icon:      normalizeEnvironmentCommandIcon(string(command.Icon)),
			Command:   commandText,
			IsDefault: command.IsDefault,
		}
		if next.IsDefault && defaultIndex == -1 {
			defaultIndex = len(normalized)
		} else {
			next.IsDefault = false
		}
		normalized = append(normalized, next)
	}
	if len(normalized) > 0 && defaultIndex == -1 {
		normalized[0].IsDefault = true
	}
	return normalized
}

func reorderEnvironmentCommandsForPersistence(commands []EnvironmentCommand) []EnvironmentCommand {
	if len(commands) <= 1 {
		return commands
	}
	defaultIndex := -1
	for index, command := range commands {
		if command.IsDefault {
			defaultIndex = index
			break
		}
	}
	if defaultIndex <= 0 {
		return commands
	}
	reordered := make([]EnvironmentCommand, 0, len(commands))
	reordered = append(reordered, commands[defaultIndex])
	reordered = append(reordered, commands[:defaultIndex]...)
	reordered = append(reordered, commands[defaultIndex+1:]...)
	for index := range reordered {
		reordered[index].IsDefault = index == 0
	}
	return reordered
}

func deriveEnvironmentCommandID(name, command string, index int) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(strings.TrimSpace(name)))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(strings.TrimSpace(command)))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(strconv.Itoa(index)))
	return strconv.FormatUint(hasher.Sum64(), 36)
}

func generateEnvironmentCommandID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func normalizeEnvironmentCommandIcon(input string) EnvironmentCommandIcon {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case string(EnvironmentCommandIconRun):
		return EnvironmentCommandIconRun
	case string(EnvironmentCommandIconTool):
		return EnvironmentCommandIconTool
	case string(EnvironmentCommandIconTest):
		return EnvironmentCommandIconTest
	default:
		return EnvironmentCommandIconOther
	}
}

func findEnvironmentCommand(commands []EnvironmentCommand, id string) (EnvironmentCommand, bool) {
	for _, command := range commands {
		if command.ID == id {
			return command, true
		}
	}
	return EnvironmentCommand{}, false
}

func markEnvironmentCommandDefault(commands []EnvironmentCommand, id string) []EnvironmentCommand {
	next := make([]EnvironmentCommand, len(commands))
	copy(next, commands)
	matched := false
	for index := range next {
		next[index].IsDefault = next[index].ID == id
		if next[index].IsDefault {
			matched = true
		}
	}
	if !matched && len(next) > 0 {
		next[0].IsDefault = true
	}
	return normalizeEnvironmentCommands(next)
}

func runtimeCommandDefaultNeedsUpdate(commands []EnvironmentCommand, id string, makeDefault bool) bool {
	if !makeDefault {
		return false
	}
	for _, command := range commands {
		if command.ID == id {
			return !command.IsDefault
		}
	}
	return false
}

func buildEnvironmentCommandScript(workingDir, command string) string {
	return "[Console]::InputEncoding = [System.Text.Encoding]::UTF8; " +
		"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; " +
		"$OutputEncoding = [System.Text.Encoding]::UTF8; " +
		"Set-Location -LiteralPath '" + escapePowerShellLiteral(workingDir) + "'; " +
		command
}
