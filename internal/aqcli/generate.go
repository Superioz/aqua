package aqcli

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"math/rand"
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
