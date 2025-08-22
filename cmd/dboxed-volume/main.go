package dboxed_volume

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

	Run     commands.RunCmd     `cmd:"" help:"Download, unpack and run a box"`
	Systemd commands.SystemdCmd `cmd:"" help:"Sub commands to control dboxed systemd integration"`
	Runc    commands.RuncCmd    `cmd:"" help:"Run runc for a box"`
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
