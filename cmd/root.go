package cmd

import (
	"context"
	"log"
	"os"

	"github.com/mermonia/peridot/internal/logger"
	"github.com/urfave/cli/v3"
)

func Execute() {
	cmd := &cli.Command{
		Name:                  "peridot",
		EnableShellCompletion: true,
		Version:               "v0.1.0",
		Authors: []any{
			"Daniel Sanso <cs.daniel.sanso@gmail.com>",
		},
		Copyright:   "(c) 2025 Daniel Sanso",
		Usage:       "feature-rich dotfiles manager",
		HideHelp:    false,
		HideVersion: false,
		Commands: []*cli.Command{
			&AddCommand,
			&DeployCommand,
			&InitCommand,
			{
				Name: "remove",
			},
			{
				Name: "status",
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Error(err.Error())
		log.Fatal(err)
	}
}
