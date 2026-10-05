package steps

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

var list = []manifest.Step{
	{ID: "a", Check: "check-a", Run: "run-a", Update: "update-a"},
	{ID: "b", Check: "check-b", Run: "run-b"},
}

func TestInstallSkipsSatisfiedSteps(t *testing.T) {
	f := exec.NewFake() // Default exit 0: every check passes
	s := Install(context.Background(), f, list, false, &bytes.Buffer{})
	if len(s) != 2 || s[0].Status != result.Skipped || s[1].Status != result.Skipped {
		t.Fatalf("summary = %+v", s)
	}
	if strings.Join(f.Calls, ",") != "check-a,check-b" {
		t.Fatalf("run must not be called: %v", f.Calls)
	}
}

func TestInstallRunsThenRechecks(t *testing.T) {
	f := exec.NewFake()
	f.Queue["check-a"] = []exec.Result{{ExitCode: 1}, {ExitCode: 0}}
	f.Scripts["run-a"] = exec.Result{Stdout: "installing...\n"}
	var out bytes.Buffer
	s := Install(context.Background(), f, list[:1], false, &out)
	if s[0].Status != result.Ok || s[0].Detail != "installed" {
		t.Fatalf("entry = %+v", s[0])
	}
	if strings.Join(f.Calls, ",") != "check-a,run-a,check-a" {
		t.Fatalf("calls = %v", f.Calls)
	}
	if !strings.Contains(out.String(), "installing...") {
		t.Fatal("run output should stream to out")
	}
}

func TestInstallFailsWhenRecheckFails(t *testing.T) {
	f := exec.NewFake()
	f.Scripts["check-a"] = exec.Result{ExitCode: 1}
	f.Scripts["run-a"] = exec.Result{ExitCode: 2, Stderr: "curl: (6) could not resolve\n"}
	s := Install(context.Background(), f, list[:1], false, &bytes.Buffer{})
	if s[0].Status != result.Failed || !strings.Contains(s[0].Detail, "could not resolve") {
		t.Fatalf("entry = %+v", s[0])
	}
}

func TestInstallDryRunDoesNotRun(t *testing.T) {
	f := exec.NewFake()
	f.Default = exec.Result{ExitCode: 1}
	s := Install(context.Background(), f, list, true, &bytes.Buffer{})
	for i, e := range s {
		if e.Status != result.Skipped || e.Detail != "would run" {
			t.Fatalf("entry %d = %+v", i, e)
		}
	}
	if strings.Join(f.Calls, ",") != "check-a,check-b" {
		t.Fatalf("dry run must only check: %v", f.Calls)
	}
}

func TestUpdateOnlyForStepsThatDefineIt(t *testing.T) {
	f := exec.NewFake()
	f.Scripts["check-a"] = exec.Result{ExitCode: 1} // must be ignored by Update
	s := Update(context.Background(), f, list, false, &bytes.Buffer{})
	if len(s) != 1 || s[0].ID != "a" || s[0].Status != result.Ok || s[0].Detail != "updated" {
		t.Fatalf("summary = %+v", s)
	}
	if strings.Join(f.Calls, ",") != "update-a" {
		t.Fatalf("calls = %v", f.Calls)
	}
}

func TestUpdateDryRunAndFailure(t *testing.T) {
	f := exec.NewFake()
	s := Update(context.Background(), f, list, true, &bytes.Buffer{})
	if len(s) != 1 || s[0].Status != result.Skipped || s[0].Detail != "would update" || len(f.Calls) != 0 {
		t.Fatalf("dry run = %+v calls=%v", s, f.Calls)
	}
	f.Scripts["update-a"] = exec.Result{ExitCode: 1, Stderr: "nope\n"}
	s = Update(context.Background(), f, list, false, &bytes.Buffer{})
	if s[0].Status != result.Failed || s[0].Detail != "nope" {
		t.Fatalf("failure = %+v", s[0])
	}
}
