package commands

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/util"
	"github.com/dboxed/dboxed-volume/pkg/volume"
)

type ResticBackupCmd struct {
	Image string `help:"Specify the location of the volume image" type:"existingfile"`

	Repo         string `help:"Restic Repo"`
	PasswordFile string `help:"Restic password file" type:"existingfile"`

	Tag []string `help:"Restic tags"`
}

func (cmd *ResticBackupCmd) Run(g *flags.GlobalFlags) error {
	snapName := "restic-backup-snapshot"

	v, err := volume.Open(cmd.Image)
	if err != nil {
		return err
	}
	defer v.Close(true)

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.Remove(tmpDir)

	tmpMnt := filepath.Join(tmpDir, "mnt")
	err = os.Mkdir(tmpMnt, 0700)
	if err != nil {
		return err
	}
	defer os.Remove(tmpMnt)

	err = v.CreateSnapshot(snapName, true)
	if err != nil {
		return err
	}
	defer func() {
		err := v.DeleteSnapshot(snapName)
		if err != nil {
			slog.Error("deferred deletion of snapshot failed", slog.Any("error", err))
		}
	}()
	err = v.MountSnapshot(snapName, tmpMnt)
	if err != nil {
		return err
	}
	defer func() {
		_, err := util.RunCommand(false, "umount", tmpMnt)
		if err != nil {
			slog.Error("deferred unmounting failed", slog.Any("error", err))
		}
	}()

	err = cmd.doBackup(tmpMnt)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *ResticBackupCmd) doBackup(tmpMnt string) error {
	args := []string{
		"backup",
		"--one-file-system",
		"--no-scan",
		"-g", "tags",
	}

	if cmd.Repo != "" {
		args = append(args, "-r", cmd.Repo)
	}
	if cmd.PasswordFile != "" {
		args = append(args, "--password-file", cmd.PasswordFile)
	}

	for _, t := range cmd.Tag {
		args = append(args, "--tag", t)
	}

	args = append(args, ".")

	env := os.Environ()

	c := exec.Command("restic", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = env
	c.Dir = tmpMnt

	err := c.Run()
	if err != nil {
		return err
	}

	return nil
}
