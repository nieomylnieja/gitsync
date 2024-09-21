package gitsync

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
)

type command struct {
	skipErroneousStatus []int
	stdin               io.Reader
	env                 []string
}

func newCmd() *command {
	return &command{}
}

func execCmd(name string, arg ...string) (*bytes.Buffer, error) {
	return newCmd().Exec(name, arg...)
}

func (c *command) SkipErroneousStatus(status ...int) *command {
	c.skipErroneousStatus = append(c.skipErroneousStatus, status...)
	return c
}

func (c *command) SetStdin(stdin io.Reader) *command {
	c.stdin = stdin
	return c
}

func (c *command) WithEnv(key, value string) *command {
	c.env = append(c.env, fmt.Sprintf("%s=%s", key, value))
	return c
}

func (c *command) Exec(name string, arg ...string) (*bytes.Buffer, error) {
	cmd := exec.Command(name, arg...)
	if cmd.Stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if cmd.Stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}
	if c.stdin != nil {
		cmd.Stdin = c.stdin
	}
	if len(c.env) > 0 {
		cmd.Env = c.env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		var execErr *exec.ExitError
		if errors.As(err, &execErr) && slices.Contains(c.skipErroneousStatus, execErr.ExitCode()) {
			return &stdout, nil
		}
		return nil, fmt.Errorf("failed to execute '%s' command: %s", cmd, stderr.String())
	}
	return &stdout, nil
}
