package usecase

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type LaunchVSCodeInput struct {
	RootID string
	Path   string
	Line   int
	Column int
	Action string
}

type LaunchVSCodeOutput struct {
	Mode   string `json:"mode"`
	Target string `json:"target"`
	Action string `json:"action"`
}

func (s *Service) LaunchVSCode(ctx context.Context, in LaunchVSCodeInput) (LaunchVSCodeOutput, error) {
	if err := s.ensureRegistry(); err != nil {
		return LaunchVSCodeOutput{}, err
	}
	root, err := s.Registry.GetRoot(in.RootID)
	if err != nil {
		return LaunchVSCodeOutput{}, err
	}
	rootAbs, err := root.RootDir()
	if err != nil {
		return LaunchVSCodeOutput{}, err
	}
	action := normalizeLaunchAction(in.Action)

	targetAbs := rootAbs
	targetFileAbs := ""
	if strings.TrimSpace(in.Path) != "" {
		relPath, err := root.NormalizePath(in.Path)
		if err != nil {
			return LaunchVSCodeOutput{}, err
		}
		if _, _, err := root.StatFile(relPath); err != nil {
			return LaunchVSCodeOutput{}, err
		}
		targetFileAbs = filepath.Join(rootAbs, filepath.FromSlash(relPath))
		targetAbs = targetFileAbs
	}

	switch action {
	case "explorer":
		return launchSystemFileManager(ctx, rootAbs, targetFileAbs)
	case "powershell":
		return launchPowerShellTerminal(ctx, rootAbs, targetFileAbs)
	}

	if cmdPath, args, ok := resolveVSCodeCLI(rootAbs, targetFileAbs, in.Line, in.Column); ok {
		cmd := exec.Command(cmdPath, args...)
		if err := cmd.Start(); err == nil {
			_ = cmd.Process.Release()
			return LaunchVSCodeOutput{Mode: "cli", Target: targetAbs, Action: action}, nil
		}
	}

	targetURI := buildVSCodeURI(targetAbs)
	if targetURI == "" {
		return LaunchVSCodeOutput{}, errors.New("unable to build vscode target")
	}
	if err := openExternalURL(ctx, targetURI); err != nil {
		return LaunchVSCodeOutput{}, err
	}
	return LaunchVSCodeOutput{Mode: "uri", Target: targetAbs, Action: action}, nil
}

func normalizeLaunchAction(input string) string {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "explorer":
		return "explorer"
	case "powershell":
		return "powershell"
	default:
		return "vscode"
	}
}

func resolveVSCodeCLI(rootAbs, targetFileAbs string, line, column int) (string, []string, bool) {
	commandPath := findVSCodeCLI()
	if commandPath == "" {
		return "", nil, false
	}
	args := []string{"--reuse-window", rootAbs}
	if targetFileAbs != "" {
		gotoTarget := targetFileAbs
		if line > 0 {
			gotoTarget += ":" + strconv.Itoa(line)
			if column > 0 {
				gotoTarget += ":" + strconv.Itoa(column)
			}
		}
		args = append(args, "--goto", gotoTarget)
	}
	return commandPath, args, true
}

func findVSCodeCLI() string {
	candidates := []string{"code"}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			"code.cmd",
			"code.exe",
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Microsoft VS Code", "bin", "code.cmd"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Microsoft VS Code", "Code.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft VS Code", "bin", "code.cmd"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft VS Code", "Code.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft VS Code", "bin", "code.cmd"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft VS Code", "Code.exe"),
		)
	}
	return findExistingExecutable(candidates)
}

func findExistingExecutable(candidates []string) string {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if strings.Contains(candidate, string(filepath.Separator)) || filepath.IsAbs(candidate) {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate
			}
			continue
		}
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved
		}
	}
	return ""
}

func findPowerShellExecutable() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	candidates := []string{
		"powershell.exe",
		filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
		"pwsh.exe",
		filepath.Join(os.Getenv("ProgramFiles"), "PowerShell", "7", "pwsh.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "PowerShell", "6", "pwsh.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WindowsApps", "pwsh.exe"),
	}
	return findExistingExecutable(candidates)
}

func findWindowsPowerShellHost() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	candidates := []string{
		"powershell.exe",
		filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
	}
	return findExistingExecutable(candidates)
}

func buildVSCodeURI(targetAbs string) string {
	clean := filepath.Clean(strings.TrimSpace(targetAbs))
	if clean == "" {
		return ""
	}
	slashed := filepath.ToSlash(clean)
	if runtime.GOOS == "windows" && !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return (&url.URL{
		Scheme: "vscode",
		Host:   "file",
		Path:   slashed,
	}).String()
}

func launchSystemFileManager(_ context.Context, rootAbs, targetFileAbs string) (LaunchVSCodeOutput, error) {
	targetAbs := rootAbs
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		if targetFileAbs != "" {
			targetAbs = targetFileAbs
			cmd = exec.Command("explorer.exe", "/select,", targetFileAbs)
		} else {
			cmd = exec.Command("explorer.exe", rootAbs)
		}
	case "darwin":
		if targetFileAbs != "" {
			targetAbs = targetFileAbs
			cmd = exec.Command("open", "-R", targetFileAbs)
		} else {
			cmd = exec.Command("open", rootAbs)
		}
	default:
		if targetFileAbs != "" {
			targetAbs = filepath.Dir(targetFileAbs)
		}
		cmd = exec.Command("xdg-open", targetAbs)
	}
	if err := cmd.Start(); err != nil {
		return LaunchVSCodeOutput{}, err
	}
	_ = cmd.Process.Release()
	return LaunchVSCodeOutput{
		Mode:   "explorer",
		Target: targetAbs,
		Action: "explorer",
	}, nil
}

func launchPowerShellTerminal(_ context.Context, rootAbs, targetFileAbs string) (LaunchVSCodeOutput, error) {
	if runtime.GOOS != "windows" {
		return LaunchVSCodeOutput{}, errors.New("powershell launch is only available on Windows")
	}
	workingDir := rootAbs
	if targetFileAbs != "" {
		workingDir = filepath.Dir(targetFileAbs)
	}
	executable := findPowerShellExecutable()
	if executable == "" {
		return LaunchVSCodeOutput{}, errors.New("unable to find PowerShell executable")
	}
	script := "Set-Location -LiteralPath '" + escapePowerShellLiteral(workingDir) + "'"
	args := []string{
		"-NoLogo",
		"-NoExit",
		"-Command",
		script,
	}
	if err := startDetachedWindowsProcess(executable, workingDir, args); err != nil {
		return LaunchVSCodeOutput{}, err
	}
	return LaunchVSCodeOutput{
		Mode:   "powershell",
		Target: workingDir,
		Action: "powershell",
	}, nil
}

func escapePowerShellLiteral(input string) string {
	return strings.ReplaceAll(input, "'", "''")
}

func quotePowerShellLiteral(input string) string {
	return "'" + escapePowerShellLiteral(input) + "'"
}

func buildWindowsStartProcessScript(executable, workingDir string, args []string) string {
	quotedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		quotedArgs = append(quotedArgs, quotePowerShellLiteral(arg))
	}
	return "$ErrorActionPreference = 'Stop'; Start-Process -FilePath " +
		quotePowerShellLiteral(executable) +
		" -WorkingDirectory " +
		quotePowerShellLiteral(workingDir) +
		" -ArgumentList @(" +
		strings.Join(quotedArgs, ", ") +
		") | Out-Null"
}

func openExternalURL(_ context.Context, rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
