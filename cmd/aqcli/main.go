package main

import (
	"github.com/superioz/aqua/internal/aqcli"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	app := &cli.App{
		Name:  "aq",
		Usage: "Tool to upload local files to an aqua server.",
		Commands: []*cli.Command{
			aqcli.UploadCommand,
			aqcli.GenerateCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
