package usecase

import (
	"bufio"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type EnvironmentCommandRunStatus string

const (
	EnvironmentCommandRunStatusRunning   EnvironmentCommandRunStatus = "running"
	EnvironmentCommandRunStatusCompleted EnvironmentCommandRunStatus = "completed"
	EnvironmentCommandRunStatusFailed    EnvironmentCommandRunStatus = "failed"
)

type EnvironmentCommandRun struct {
	RunID     string                      `json:"run_id"`
	RootID    string                      `json:"root_id"`
	CommandID string                      `json:"command_id"`
	Name      string                      `json:"name"`
	Command   string                      `json:"command"`
	Target    string                      `json:"target"`
	Status    EnvironmentCommandRunStatus `json:"status"`
	StartedAt time.Time                   `json:"started_at"`
	FinishedAt *time.Time                 `json:"finished_at,omitempty"`
	ExitCode  *int                        `json:"exit_code,omitempty"`
}

type EnvironmentCommandRunEvent struct {
	Type   string
	Run    EnvironmentCommandRun
	Stream string
	Chunk  string
	Error  string
}

type StartEnvironmentCommandRunInput struct {
	ClientID  string
	RootID    string
	CommandID string
	Name      string
	Command   string
	Target    string
	OnEvent   func(EnvironmentCommandRunEvent)
}

type EnvironmentCommandRunner struct {
	mu     sync.Mutex
	active map[string]string
	runs   map[string]*environmentCommandProcess
}

type environmentCommandProcess struct {
	clientID string
	run      EnvironmentCommandRun
	cmd      *exec.Cmd
	onEvent  func(EnvironmentCommandRunEvent)
}

func NewEnvironmentCommandRunner() *EnvironmentCommandRunner {
	return &EnvironmentCommandRunner{
		active: make(map[string]string),
		runs:   make(map[string]*environmentCommandProcess),
	}
}

func (r *EnvironmentCommandRunner) Start(in StartEnvironmentCommandRunInput) (EnvironmentCommandRun, error) {
	if strings.TrimSpace(in.Command) == "" {
		return EnvironmentCommandRun{}, errors.New("command text required")
	}
	if strings.TrimSpace(in.Target) == "" {
		return EnvironmentCommandRun{}, errors.New("command target required")
	}
	executable := findPowerShellExecutable()
	if executable == "" {
		return EnvironmentCommandRun{}, errors.New("unable to find PowerShell executable")
	}

	clientID := strings.TrimSpace(in.ClientID)
	rootID := strings.TrimSpace(in.RootID)
	runKey := clientID + "::" + rootID

	r.mu.Lock()
	if existingRunID := r.active[runKey]; existingRunID != "" {
		if existing := r.runs[existingRunID]; existing != nil && existing.run.Status == EnvironmentCommandRunStatusRunning {
			r.mu.Unlock()
			return EnvironmentCommandRun{}, errors.New("another command is already running")
		}
		delete(r.active, runKey)
	}
	run := EnvironmentCommandRun{
		RunID:     generateEnvironmentCommandID(),
		RootID:    rootID,
		CommandID: strings.TrimSpace(in.CommandID),
		Name:      strings.TrimSpace(in.Name),
		Command:   strings.TrimSpace(in.Command),
		Target:    strings.TrimSpace(in.Target),
		Status:    EnvironmentCommandRunStatusRunning,
		StartedAt: time.Now().UTC(),
	}
	proc := &environmentCommandProcess{
		clientID: clientID,
		run:      run,
		onEvent:  in.OnEvent,
	}
	r.active[runKey] = run.RunID
	r.runs[run.RunID] = proc
	r.mu.Unlock()

	cmd := exec.Command(
		executable,
		"-NoLogo",
		"-NoProfile",
		"-ExecutionPolicy",
		"Bypass",
		"-Command",
		buildEnvironmentCommandScript(run.Target, run.Command),
	)
	cmd.Dir = run.Target
	configureEnvironmentCommandProcess(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.failStart(runKey, run.RunID)
		return EnvironmentCommandRun{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.failStart(runKey, run.RunID)
		return EnvironmentCommandRun{}, err
	}
	if err := cmd.Start(); err != nil {
		r.failStart(runKey, run.RunID)
		return EnvironmentCommandRun{}, err
	}
	proc.cmd = cmd

	go r.watchRun(runKey, proc, stdout, stderr)
	return run, nil
}

func (r *EnvironmentCommandRunner) failStart(runKey, runID string) {
	r.mu.Lock()
	delete(r.active, runKey)
	delete(r.runs, runID)
	r.mu.Unlock()
}

func (r *EnvironmentCommandRunner) watchRun(runKey string, proc *environmentCommandProcess, stdout io.ReadCloser, stderr io.ReadCloser) {
	proc.emit(EnvironmentCommandRunEvent{
		Type: "started",
		Run:  proc.run,
	})

	var streams sync.WaitGroup
	streams.Add(2)
	go func() {
		defer streams.Done()
		streamEnvironmentCommandOutput(stdout, "stdout", proc.emitOutput)
	}()
	go func() {
		defer streams.Done()
		streamEnvironmentCommandOutput(stderr, "stderr", proc.emitOutput)
	}()

	waitErr := proc.cmd.Wait()
	streams.Wait()

	finished := time.Now().UTC()
	run := proc.run
	run.FinishedAt = &finished
	if waitErr != nil {
		run.Status = EnvironmentCommandRunStatusFailed
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			code := exitErr.ExitCode()
			run.ExitCode = &code
		}
	} else {
		run.Status = EnvironmentCommandRunStatusCompleted
		code := 0
		run.ExitCode = &code
	}

	r.mu.Lock()
	delete(r.active, runKey)
	if current := r.runs[run.RunID]; current != nil {
		current.run = run
	}
	r.mu.Unlock()

	proc.emit(EnvironmentCommandRunEvent{
		Type: "finished",
		Run:  run,
		Error: func() string {
			if waitErr == nil {
				return ""
			}
			return waitErr.Error()
		}(),
	})
}

func streamEnvironmentCommandOutput(reader io.Reader, stream string, emit func(stream, chunk string)) {
	if reader == nil || emit == nil {
		return
	}
	buf := bufio.NewReader(reader)
	for {
		chunk, err := buf.ReadString('\n')
		if chunk != "" {
			emit(stream, chunk)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				emit("stderr", err.Error()+"\n")
			}
			return
		}
	}
}

func (p *environmentCommandProcess) emit(event EnvironmentCommandRunEvent) {
	if p == nil || p.onEvent == nil {
		return
	}
	p.onEvent(event)
}

func (p *environmentCommandProcess) emitOutput(stream, chunk string) {
	if strings.TrimSpace(chunk) == "" && !strings.Contains(chunk, "\n") && !strings.Contains(chunk, "\r") {
		return
	}
	p.emit(EnvironmentCommandRunEvent{
		Type:   "output",
		Run:    p.run,
		Stream: stream,
		Chunk:  chunk,
	})
}
