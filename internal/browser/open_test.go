package browser

import (
	"errors"
	"strings"
	"testing"
)

const testURL = "https://example.test/app"

func TestNew_ReturnsOpener(t *testing.T) {
	o := New()
	if o == nil {
		t.Fatal("New returned nil")
	}
	if o.GOOS != nil || o.LookPath != nil || o.Start != nil {
		t.Error("New should leave hooks nil (production defaults)")
	}
}

func TestNoop_DoesNotStart(t *testing.T) {
	started := false
	// Noop sets Start; Open must still resolve a command then call it.
	// Replace Start after Noop to observe, or rely on Noop's own Start.
	o := Noop()
	// Hijack to detect calls while keeping no-op behavior.
	orig := o.Start
	o.Start = func(name string, arg ...string) error {
		started = true
		return orig(name, arg...)
	}
	o.GOOS = func() string { return "darwin" }
	if err := o.Open(testURL); err != nil {
		t.Fatal(err)
	}
	if !started {
		t.Fatal("Noop Start should still be invoked by Open")
	}
}

func TestResolveCommand_Darwin(t *testing.T) {
	name, args, err := resolveCommand("darwin", nil, testURL)
	if err != nil {
		t.Fatal(err)
	}
	if name != "open" || len(args) != 1 || args[0] != testURL {
		t.Fatalf("got %q %v", name, args)
	}
}

func TestResolveCommand_Windows(t *testing.T) {
	name, args, err := resolveCommand("windows", nil, testURL)
	if err != nil {
		t.Fatal(err)
	}
	if name != "rundll32" || len(args) != 2 || args[0] != "url.dll,FileProtocolHandler" || args[1] != testURL {
		t.Fatalf("got %q %v", name, args)
	}
}

func TestResolveCommand_Unsupported(t *testing.T) {
	_, _, err := resolveCommand("plan9", nil, testURL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported platform") {
		t.Errorf("err = %v", err)
	}
}

func TestResolveCommand_Linux_XDGOpen(t *testing.T) {
	look := func(file string) (string, error) {
		if file == "xdg-open" {
			return "/bin/xdg-open", nil
		}
		return "", errors.New("not found")
	}
	name, args, err := resolveCommand("linux", look, testURL)
	if err != nil {
		t.Fatal(err)
	}
	if name != "xdg-open" || len(args) != 1 || args[0] != testURL {
		t.Fatalf("got %q %v", name, args)
	}
}

func TestResolveCommand_Linux_GIO(t *testing.T) {
	look := func(file string) (string, error) {
		if file == "gio" {
			return "/bin/gio", nil
		}
		return "", errors.New("not found")
	}
	name, args, err := resolveCommand("linux", look, testURL)
	if err != nil {
		t.Fatal(err)
	}
	if name != "gio" || len(args) != 2 || args[0] != "open" || args[1] != testURL {
		t.Fatalf("got %q %v", name, args)
	}
}

func TestResolveCommand_Linux_SensibleBrowser(t *testing.T) {
	look := func(file string) (string, error) {
		if file == "sensible-browser" {
			return "/bin/sensible-browser", nil
		}
		return "", errors.New("not found")
	}
	name, args, err := resolveCommand("linux", look, testURL)
	if err != nil {
		t.Fatal(err)
	}
	if name != "sensible-browser" || len(args) != 1 || args[0] != testURL {
		t.Fatalf("got %q %v", name, args)
	}
}

func TestResolveCommand_Linux_NoOpener(t *testing.T) {
	look := func(file string) (string, error) {
		return "", errors.New("not found")
	}
	_, _, err := resolveCommand("linux", look, testURL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "xdg-open") {
		t.Errorf("err = %v", err)
	}
}

func TestOpen_DarwinUsesOpen(t *testing.T) {
	var gotName string
	var gotArgs []string
	o := &Opener{
		GOOS: func() string { return "darwin" },
		Start: func(name string, arg ...string) error {
			gotName, gotArgs = name, append([]string(nil), arg...)
			return nil
		},
	}
	if err := o.Open(testURL); err != nil {
		t.Fatal(err)
	}
	if gotName != "open" || len(gotArgs) != 1 || gotArgs[0] != testURL {
		t.Fatalf("got %q %v", gotName, gotArgs)
	}
}

func TestOpen_WindowsUsesRundll32(t *testing.T) {
	var gotName string
	var gotArgs []string
	o := &Opener{
		GOOS: func() string { return "windows" },
		Start: func(name string, arg ...string) error {
			gotName, gotArgs = name, append([]string(nil), arg...)
			return nil
		},
	}
	if err := o.Open(testURL); err != nil {
		t.Fatal(err)
	}
	if gotName != "rundll32" {
		t.Fatalf("name = %q", gotName)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "url.dll,FileProtocolHandler" || gotArgs[1] != testURL {
		t.Fatalf("args = %v", gotArgs)
	}
}

func TestOpen_LinuxUsesInjectedLookPath(t *testing.T) {
	var gotName string
	o := &Opener{
		GOOS: func() string { return "linux" },
		LookPath: func(file string) (string, error) {
			if file == "gio" {
				return "/usr/bin/gio", nil
			}
			return "", errors.New("missing")
		},
		Start: func(name string, arg ...string) error {
			gotName = name
			return nil
		},
	}
	if err := o.Open(testURL); err != nil {
		t.Fatal(err)
	}
	if gotName != "gio" {
		t.Fatalf("name = %q, want gio", gotName)
	}
}

func TestOpen_UnsupportedPlatformErrors(t *testing.T) {
	o := &Opener{
		GOOS: func() string { return "plan9" },
		Start: func(name string, arg ...string) error {
			t.Fatal("Start must not be called for unsupported platform")
			return nil
		},
	}
	err := o.Open(testURL)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_StartErrorPropagates(t *testing.T) {
	o := &Opener{
		GOOS: func() string { return "darwin" },
		Start: func(name string, arg ...string) error {
			return errors.New("spawn failed")
		},
	}
	err := o.Open(testURL)
	if err == nil {
		t.Fatal("expected start error")
	}
	if !strings.Contains(err.Error(), "spawn failed") {
		t.Errorf("err = %v", err)
	}
}

func TestOpen_LinuxNoOpenerDoesNotStart(t *testing.T) {
	o := &Opener{
		GOOS:     func() string { return "linux" },
		LookPath: func(file string) (string, error) { return "", errors.New("none") },
		Start: func(name string, arg ...string) error {
			t.Fatal("Start must not be called when no opener exists")
			return nil
		},
	}
	err := o.Open(testURL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no opener found") {
		t.Errorf("err = %v", err)
	}
}
