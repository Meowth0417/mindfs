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
}

type LaunchVSCodeOutput struct {
	Mode   string `json:"mode"`
	Target string `json:"target"`
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

	if cmdPath, args, ok := resolveVSCodeCLI(rootAbs, targetFileAbs, in.Line, in.Column); ok {
		cmd := exec.CommandContext(ctx, cmdPath, args...)
		if err := cmd.Start(); err == nil {
			_ = cmd.Process.Release()
			return LaunchVSCodeOutput{Mode: "cli", Target: targetAbs}, nil
		}
	}

	targetURI := buildVSCodeURI(targetAbs)
	if targetURI == "" {
		return LaunchVSCodeOutput{}, errors.New("unable to build vscode target")
	}
	if err := openExternalURL(ctx, targetURI); err != nil {
		return LaunchVSCodeOutput{}, err
	}
	return LaunchVSCodeOutput{Mode: "uri", Target: targetAbs}, nil
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

func openExternalURL(ctx context.Context, rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", rawURL)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", rawURL)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
