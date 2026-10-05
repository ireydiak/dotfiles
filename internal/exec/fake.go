package exec

import (
	"context"
	"io"
	"sync"
)

// Fake is a Runner for tests. Queue entries for a command are consumed first,
// in order; then Scripts; then Default. Every command is recorded in Calls.
type Fake struct {
	mu      sync.Mutex
	Scripts map[string]Result
	Queue   map[string][]Result
	Default Result
	Calls   []string
}

func NewFake() *Fake {
	return &Fake{Scripts: map[string]Result{}, Queue: map[string][]Result{}}
}

func (f *Fake) Run(_ context.Context, command string, out io.Writer) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, command)
	var res Result
	switch {
	case len(f.Queue[command]) > 0:
		res = f.Queue[command][0]
		f.Queue[command] = f.Queue[command][1:]
	default:
		var ok bool
		if res, ok = f.Scripts[command]; !ok {
			res = f.Default
		}
	}
	if out != nil {
		io.WriteString(out, res.Output())
	}
	return res, nil
}
