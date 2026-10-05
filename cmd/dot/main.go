// cmd/dot/main.go
package main

import (
	"fmt"
	"os"
)

const usage = `usage: dot <command> [flags]

commands:
  status   report packages, links, services, manual steps and repo state
  install  trust taps, brew bundle install, link, run steps, start services
  link     create or repair symlinks from home/ into $HOME
  export   dump installed packages and update the Brewfile
  update   brew update, brew upgrade, then step updates
  commit   git add -A, commit and push
  (none)   open the dashboard
`

func main() {
	fmt.Fprint(os.Stderr, usage)
	os.Exit(2)
}
