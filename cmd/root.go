package cmd

import (
	"context"
	"github.com/urfave/cli/v3"
	"log"
	"os"
)

func Execute() {
	cmd := &cli.Command{
		Name:                  "peridot",
		EnableShellCompletion: true,
		Version:               "v0.1.0",
		// Author: ,
		Copyright:   "(c) 2025 Daniel Sanso",
		Usage:       "feature-rich dotfiles manager",
		HideHelp:    false,
		HideVersion: false,
		Commands: []*cli.Command{
			{
				Name: "deploy",
			},
			{
				Name:      "add",
				Aliases:   []string{"a"},
				Usage:     "add a module to the peridot dotfiles directory",
				ArgsUsage: "<module>",
				Description: "If not already existing, creates a directory and a module config file\n" +
					"for the specified <module>. By default, the module config will be\n" +
					"identical to the provided default-module.toml.",
				Action: func(ctx context.Context, c *cli.Command) error {
					return ExecuteAdd()
				},
			},
			{
				Name: "remove",
			},
			{
				Name:    "init",
				Aliases: []string{"i"},
				Usage:   "initialize dotfiles dir",
				Description: "If not already existing, creates the directory specified in the\n" +
					"'dotifles_dir' field of the peridot config, along with directories and\n" +
					"config files for all modules specified in the 'managed_modules' field.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "dir",
						Aliases: []string{"d"},
						Value:   "",
						Usage:   "path of the dir to be initialized",
					},
					&cli.BoolFlag{
						Name:    "here",
						Aliases: []string{"H"},
						Value:   false,
						Usage:   "set the dir to be initialized to the current dir",
					},
					&cli.BoolFlag{
						Name:    "persist",
						Aliases: []string{"p"},
						Value:   false,
						Usage: "if the dir to be initialized was set via a flag (like --here or --dir), \n" +
							"overwrite the current user dir's configuration file's dotfiles_dir field.",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return ExecuteInit()
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
