package util

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
)

func RunCommand(catchStdout bool, command string, args ...string) ([]byte, error) {
	var cmdStdout bytes.Buffer

	cmd := exec.Command(command, args...)
	if catchStdout {
		cmd.Stdout = &cmdStdout
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	return cmdStdout.Bytes(), err
}

func RunCommandJson(ret any, command string, args ...string) error {
	stdout, err := RunCommand(true, command, args...)
	if err != nil {
		return err
	}
	err = json.Unmarshal(stdout, ret)
	if err != nil {
		return err
	}
	return nil
}
