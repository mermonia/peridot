package cmd

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func Execute() {
	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "deploy",
			},
			{
				Name: "add",
			},
			{
				Name: "remove",
			},
			{
				Name:    "init",
				Aliases: []string{"i"},
				Usage:   "initialize dotfiles dir specified in the config",
				Action: func(ctx context.Context, c *cli.Command) error {
					if err := ExecuteInit(); err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name: "status",
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
