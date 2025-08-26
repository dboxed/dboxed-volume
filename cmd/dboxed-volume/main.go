package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/commands"
	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	versionpkg "github.com/dboxed/dboxed-volume/pkg/version"
)

type Cli struct {
	flags.GlobalFlags

	Serve commands.ServeCmd `cmd:"" help:"Serve dboxed-volume"`

	Repo   commands.RepoCmd   `cmd:"" help:"Repo commands"`
	Volume commands.VolumeCmd `cmd:"" help:"Volume commands"`

	Init  commands.InitCmd  `cmd:"" help:"Initialize a volume"`
	Mount commands.MountCmd `cmd:"" help:"Mount a volume"`

	ResticBackup commands.ResticBackupCmd `cmd:"" help:"Backup via restic"`
}

func Execute() {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	cli := &Cli{}

	ctx := kong.Parse(cli,
		kong.Name("dboxed-volume"),
		kong.Description("A simple container volume syncer."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)

	err := ctx.Run(&cli.GlobalFlags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// set via ldflags
var version = ""

func main() {
	// was it set via -ldflags -X
	if //goland:noinspection ALL
	version != "" {
		versionpkg.Version = version
	}

	Execute()
}
