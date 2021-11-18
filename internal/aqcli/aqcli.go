package aqcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/superioz/aqua/internal/handler"
	"github.com/superioz/aqua/pkg/shttp"
	"github.com/urfave/cli/v2"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

var GenerateCommand = &cli.Command{
	Name:    "generate",
	Aliases: []string{"gen"},
	Usage:   "Generates a possible auth token",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:    "length",
			Aliases: []string{"l"},
			Value:   32,
			Usage:   "Length of the token",
		},
	},
	Action: func(c *cli.Context) error {
		size := c.Int("length")
		if size <= 1 {
			return cli.Exit("You cannot generate a token with this length. Must be >=2.", 1)
		}

		fmt.Println(generateToken(size))
		return nil
	},
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-+?!#$&%"

// generateToken generates a random `size` long string from
// a predefined hexadecimal charset.
func generateToken(size int) string {
	rand.Seed(time.Now().UnixNano())

	b := make([]byte, size)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

var UploadCommand = &cli.Command{
	Name:      "upload",
	Usage:     "Uploads a file to the aqua server",
	ArgsUsage: "<file [file2 file3 ...]>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "host",
			Usage: "Specifies to which host to upload to",
			Value: "http://localhost:8765",
		},
		&cli.StringFlag{
			Name:    "token",
			Aliases: []string{"t"},
			Usage:   "Token used for authorization",
		},
		&cli.IntFlag{
			Name:    "expires",
			Aliases: []string{"e"},
			Value:   -1,
			Usage:   "Time in seconds when the file should expire. -1 = never.",
		},
	},
	Action: func(c *cli.Context) error {
		paths := c.Args().Slice()

		if len(paths) == 0 {
			return cli.Exit("You have to provide at least one file to upload", 1)
		}

		host := c.String("host")
		if !strings.HasPrefix(host, "http") {
			// defaults to https
			host = "https://" + host
		}

		token := c.String("token")
		expires := c.Int("expires")

		for _, path := range paths {
			file, err := os.Open(path)
			if err != nil {
				// one of the file does not exist
				return fmt.Errorf("could not open file: %v", err)
			}

			id, err := doPostRequest(host, token, file, &handler.RequestMetadata{
				Expiration: int64(expires),
			})
			if err != nil {
				// one of the file does not exist
				return fmt.Errorf("could not upload file %s: %v", path, err)
			}
			file.Close()

			fmt.Printf("Uploaded file %s to %s/%s", path, host, id)
		}

		return nil
	},
}

type postResponse struct {
	Id string
}

func doPostRequest(host string, token string, file *os.File, metadata *handler.RequestMetadata) (string, error) {
	md, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	values := map[string]io.Reader{
		"file":     file,
		"metadata": bytes.NewReader(md),
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := shttp.Upload(client, host+"/upload", values, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code: %d", res.StatusCode)
	}

	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var resp postResponse
	err = json.Unmarshal(resData, &resp)
	if err != nil {
		return "", err
	}
	return resp.Id, nil
}
