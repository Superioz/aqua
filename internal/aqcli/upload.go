package aqcli

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
)

var UploadCommand = &cli.Command{
	Name:      "upload",
	Usage:     "Uploads a file to the aqua server",
	ArgsUsage: "<file [file2 file3 ...]>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "host",
			Aliases: []string{"h"},
			Usage:   "Specifies to which host to upload to",
			Value:   "localhost:8765",
		},
		&cli.StringFlag{
			Name:    "token",
			Aliases: []string{"t"},
			Usage:   "Token used for authorization",
		},
	},
	Action: func(c *cli.Context) error {
		paths := c.Args().Slice()

		if len(paths) == 0 {
			return cli.Exit("You have to provide at least one file to upload", 1)
		}

		host := c.String("host")
		token := c.String("token")

		fmt.Println("Host: " + host)
		fmt.Println("Token: " + token)
		fmt.Println(paths)

		var files []*os.File
		for _, path := range paths {
			file, err := os.Open(path)
			if err != nil {
				// one of the file does not exist
				return fmt.Errorf("could not open file: %v", err)
			}

			files = append(files, file)
		}

		return doPostRequest(host, token, files)
	},
}

func doPostRequest(host string, token string, files []*os.File) error {

	return nil
}
