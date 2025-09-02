package main

import (
	"embed"
	"os"

	"github.com/dboxed/dboxed-common/db/schematemplates"
)

//go:embed *
var E embed.FS

func main() {
	err := schematemplates.RenderSchemas(E, os.Args[1], os.Args[2])
	if err != nil {
		panic(err)
	}
}
