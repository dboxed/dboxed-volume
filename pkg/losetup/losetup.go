package losetup

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/dboxed/dboxed-volume/pkg/util"
)

type Entry struct {
	Name      string `json:"name"`
	Sizelimit int    `json:"sizelimit"`
	Offset    int    `json:"offset"`
	Autoclear bool   `json:"autoclear"`
	Ro        bool   `json:"ro"`
	BackFile  string `json:"back-file"`
	Dio       bool   `json:"dio"`
	LogSec    int    `json:"log-sec"`
}

type holder struct {
	Loopdevices []Entry `json:"loopdevices"`
}

func List() ([]Entry, error) {
	stdout, err := util.RunCommandStdout("losetup", "-J")
	if err != nil {
		return nil, err
	}

	var h holder
	err = json.Unmarshal(stdout, &h)
	if err != nil {
		return nil, err
	}
	return h.Loopdevices, nil
}

func Attach(file string) (string, error) {
	stdout, err := util.RunCommandStdout("losetup", "-f", "--show", file)
	if err != nil {
		return "", err
	}
	loDev := strings.TrimSpace(string(stdout))
	return loDev, nil
}

func GetOrAttach(file string, allowAttach bool) (string, bool, error) {
	l, err := List()
	if err != nil {
		return "", false, err
	}
	for _, e := range l {
		if e.BackFile == file {
			return e.Name, false, nil
		}
	}

	if !allowAttach {
		return "", false, os.ErrNotExist
	}

	dev, err := Attach(file)
	if err != nil {
		return "", false, err
	}
	return dev, true, nil
}

func Detach(loDev string) error {
	err := util.RunCommand("losetup", "-d", loDev)
	if err != nil {
		return err
	}
	return nil
}
