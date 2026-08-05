// Package browser opens URLs in the system default browser.
package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Opener opens URLs in the system default browser.
type Opener struct {
	// GOOS overrides runtime.GOOS (tests).
	GOOS func() string
	// LookPath finds an executable on PATH. If nil, uses exec.LookPath.
	LookPath func(file string) (string, error)
	// Start runs the opener binary. If nil, uses defaultStart (exec.Command
	// Start without Wait). Tests inject a no-op to avoid opening real tabs.
	Start func(name string, arg ...string) error
}

// New creates a production Opener (system browser).
func New() *Opener { return &Opener{} }

// Noop returns an Opener that never spawns a process.
// Use in CLI/unit tests so foreground run paths stay hermetic.
func Noop() *Opener {
	return &Opener{
		Start: func(name string, arg ...string) error { return nil },
	}
}

// Open opens url in the system default browser.
// Best-effort: errors are returned but not fatal — the user can always
// copy the URL from stdout.
func (o *Opener) Open(url string) error {
	goos := runtime.GOOS
	if o != nil && o.GOOS != nil {
		goos = o.GOOS()
	}
	lookPath := exec.LookPath
	if o != nil && o.LookPath != nil {
		lookPath = o.LookPath
	}
	start := defaultStart
	if o != nil && o.Start != nil {
		start = o.Start
	}

	name, args, err := resolveCommand(goos, lookPath, url)
	if err != nil {
		return err
	}
	if err := start(name, args...); err != nil {
		return fmt.Errorf("browser: start: %w", err)
	}
	return nil
}

// resolveCommand selects the platform opener binary and args.
// lookPath is only consulted on linux (xdg-open / gio / sensible-browser).
// Pure aside from the injected lookPath — unit tests pass a fake lookPath
// and never touch the host PATH.
func resolveCommand(goos string, lookPath func(string) (string, error), url string) (name string, args []string, err error) {
	switch goos {
	case "darwin":
		return "open", []string{url}, nil
	case "linux":
		if lookPath == nil {
			lookPath = exec.LookPath
		}
		if _, e := lookPath("xdg-open"); e == nil {
			return "xdg-open", []string{url}, nil
		}
		if _, e := lookPath("gio"); e == nil {
			return "gio", []string{"open", url}, nil
		}
		if _, e := lookPath("sensible-browser"); e == nil {
			return "sensible-browser", []string{url}, nil
		}
		return "", nil, fmt.Errorf("browser: no opener found (install xdg-open, gio, or sensible-browser)")
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}, nil
	default:
		return "", nil, fmt.Errorf("browser: unsupported platform %q", goos)
	}
}

func defaultStart(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	if err := cmd.Start(); err != nil {
		return err
	}
	// Don't wait — the child process is independent.
	go func() { _ = cmd.Wait() }()
	return nil
}
