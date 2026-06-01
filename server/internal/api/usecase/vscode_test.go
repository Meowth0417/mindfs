package usecase

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNormalizeLaunchAction(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: "vscode"},
		{input: "vscode", want: "vscode"},
		{input: " explorer ", want: "explorer"},
		{input: "POWERSHELL", want: "powershell"},
		{input: "unknown", want: "vscode"},
	}
	for _, tt := range tests {
		if got := normalizeLaunchAction(tt.input); got != tt.want {
			t.Fatalf("normalizeLaunchAction(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapePowerShellLiteral(t *testing.T) {
	if got := escapePowerShellLiteral(`C:\demo\that's\it`); got != `C:\demo\that''s\it` {
		t.Fatalf("unexpected escaped literal: %q", got)
	}
}

func TestBuildWindowsStartProcessScript(t *testing.T) {
	got := buildWindowsStartProcessScript(
		`C:\Program Files\PowerShell\7\pwsh.exe`,
		`D:\demo\that's`,
		[]string{
			"-NoLogo",
			"-NoExit",
			"-Command",
			`Set-Location -LiteralPath 'D:\demo\that''s'`,
		},
	)
	want := "$ErrorActionPreference = 'Stop'; Start-Process -FilePath 'C:\\Program Files\\PowerShell\\7\\pwsh.exe' -WorkingDirectory 'D:\\demo\\that''s' -ArgumentList @('-NoLogo', '-NoExit', '-Command', 'Set-Location -LiteralPath ''D:\\demo\\that''''s''') | Out-Null"
	if got != want {
		t.Fatalf("buildWindowsStartProcessScript() = %q, want %q", got, want)
	}
}

func TestBuildVSCodeURIWindowsDrivePrefix(t *testing.T) {
	input := filepath.Clean(`C:\demo\file.txt`)
	got := buildVSCodeURI(input)
	if runtime.GOOS == "windows" {
		if got != "vscode://file/C:/demo/file.txt" {
			t.Fatalf("unexpected windows vscode uri: %q", got)
		}
		return
	}
	if got == "" {
		t.Fatal("expected non-empty vscode uri")
	}
}

func TestFindExistingExecutableAbsoluteCandidate(t *testing.T) {
	tempDir := t.TempDir()
	executable := filepath.Join(tempDir, "demo.exe")
	if err := os.WriteFile(executable, []byte(""), 0o644); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	got := findExistingExecutable([]string{"missing-demo-executable", executable})
	if got != executable {
		t.Fatalf("findExistingExecutable() = %q, want %q", got, executable)
	}
}
