package main

import (
	"os"
	"os/exec"
	"regexp"

	"github.com/jncornett/patmatch"
)

type Filter interface {
	Apply(string) map[string]string
}

type Action interface {
	Act(map[string]string) error
}

type Trigger struct {
	Filter
	Action
}

type ShellAction []string

func NewShellAction(args ...string) ShellAction {
	return ShellAction(args)
}

func (a ShellAction) Act(line map[string]string) error {
	if len(a) == 0 {
		return nil
	}
	var processed []string
	for _, arg := range a {
		processed = append(processed, patmatch.Interpolate(arg, line))
	}
	cmd := exec.Command(processed[0], processed[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
