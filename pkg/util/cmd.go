package util

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
)

type RunCommandOptions struct {
	CatchStdout bool
	Dir         string
}

type CommandHelper struct {
	Command string
	Args    []string

	CatchStdout bool
	Dir         string

	Stdout []byte
}

func (c *CommandHelper) Run() error {
	var cmdStdout bytes.Buffer

	cmd := exec.Command(c.Command, c.Args...)
	cmd.Dir = c.Dir
	if c.CatchStdout {
		cmd.Stdout = &cmdStdout
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	c.Stdout = cmdStdout.Bytes()
	return err
}

func (c *CommandHelper) RunStdout() ([]byte, error) {
	c.CatchStdout = true
	err := c.Run()
	return c.Stdout, err
}

func (c *CommandHelper) RunStdoutJson(ret any) error {
	stdout, err := c.RunStdout()
	if err != nil {
		return err
	}
	err = json.Unmarshal(stdout, ret)
	if err != nil {
		return err
	}
	return nil
}

func RunCommand(command string, args ...string) error {
	c := CommandHelper{
		Command: command,
		Args:    args,
	}
	return c.Run()
}

func RunCommandStdout(command string, args ...string) ([]byte, error) {
	c := CommandHelper{
		Command:     command,
		Args:        args,
		CatchStdout: true,
	}
	return c.RunStdout()
}

func RunCommandJson[T any](command string, args ...string) (*T, error) {
	var ret T
	c := CommandHelper{
		Command:     command,
		Args:        args,
		CatchStdout: true,
	}
	err := c.RunStdoutJson(&ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
