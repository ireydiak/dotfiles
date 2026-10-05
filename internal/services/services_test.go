package services

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

var list = []manifest.Service{
	{ID: "yabai", Label: "com.koekeishiya.yabai", Start: "yabai --start-service"},
	{ID: "skhd", Label: "com.jackielii.skhd", Start: "skhd --start-service"},
}

func TestStatus(t *testing.T) {
	f := exec.NewFake()
	f.Scripts[Command("501", "com.jackielii.skhd")] = exec.Result{ExitCode: 113, Stderr: "Could not find service\n"}
	items := Status(context.Background(), f, list, "501")
	if !items[0].Loaded || items[1].Loaded {
		t.Fatalf("items = %+v", items)
	}
	if f.Calls[0] != "launchctl print gui/501/com.koekeishiya.yabai" {
		t.Fatalf("command = %q", f.Calls[0])
	}
}

func TestStartOnlyUnloaded(t *testing.T) {
	f := exec.NewFake()
	f.Queue[Command("501", "com.jackielii.skhd")] = []exec.Result{{ExitCode: 113}, {ExitCode: 0}}
	s := Start(context.Background(), f, list, "501", false, &bytes.Buffer{})
	if s[0].Status != result.Skipped || s[0].Detail != "already loaded" {
		t.Fatalf("yabai = %+v", s[0])
	}
	if s[1].Status != result.Ok || s[1].Detail != "started" {
		t.Fatalf("skhd = %+v", s[1])
	}
	if !strings.Contains(strings.Join(f.Calls, ","), "skhd --start-service") || strings.Contains(strings.Join(f.Calls, ","), "yabai --start-service") {
		t.Fatalf("calls = %v", f.Calls)
	}
}

func TestStartDryRunAndFailure(t *testing.T) {
	f := exec.NewFake()
	f.Default = exec.Result{ExitCode: 113}
	s := Start(context.Background(), f, list[:1], "501", true, &bytes.Buffer{})
	if s[0].Status != result.Skipped || s[0].Detail != "would start" {
		t.Fatalf("dry run = %+v", s[0])
	}
	f.Scripts["yabai --start-service"] = exec.Result{ExitCode: 1, Stderr: "yabai: permission denied\n"}
	s = Start(context.Background(), f, list[:1], "501", false, &bytes.Buffer{})
	if s[0].Status != result.Failed || !strings.Contains(s[0].Detail, "permission denied") {
		t.Fatalf("failure = %+v", s[0])
	}
}

func TestStatusReportsRunningFromLaunchctlOutput(t *testing.T) {
	f := exec.NewFake()
	f.Scripts[Command("501", "com.koekeishiya.yabai")] = exec.Result{Stdout: "\tstate = running\n\tpid = 5138\n"}
	f.Scripts[Command("501", "com.jackielii.skhd")] = exec.Result{Stdout: "\tstate = spawn scheduled\n\truns = 42\n"}
	items := Status(context.Background(), f, list, "501")
	if !items[0].Loaded || !items[0].Running {
		t.Fatalf("yabai = %+v, want loaded and running", items[0])
	}
	if !items[1].Loaded || items[1].Running {
		t.Fatalf("skhd = %+v, want loaded but not running", items[1])
	}
}
