package manual

import (
	"context"
	"testing"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
)

func TestStatusAndPending(t *testing.T) {
	f := exec.NewFake()
	f.Scripts["csrutil-check"] = exec.Result{ExitCode: 1}
	list := []manifest.Manual{
		{ID: "xcode", Check: "xcode-select -p", How: "xcode-select --install"},
		{ID: "sip", Check: "csrutil-check", How: "recovery mode"},
		{ID: "accessibility", How: "System Settings"},
	}
	items := Status(context.Background(), f, list)
	if items[0].State != Done || items[1].State != Pending || items[2].State != Unverifiable {
		t.Fatalf("items = %+v", items)
	}
	if items[2].State.String() != "verify" || items[1].How != "recovery mode" {
		t.Fatalf("string/how = %q %q", items[2].State, items[1].How)
	}
	p := Remaining(items)
	if len(p) != 2 || p[0].ID != "sip" || p[1].ID != "accessibility" {
		t.Fatalf("Remaining = %+v", p)
	}
	if len(f.Calls) != 2 {
		t.Fatalf("items without check must not run anything: %v", f.Calls)
	}
}
