package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mermonia/peridot/internal/paths"
	"github.com/urfave/cli/v3"
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
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "manage",
						Aliases: []string{"m"},
						Value:   false,
						Usage:   "add the new module to the config's managed_modules field",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return ExecuteAdd(c.Bool("manage"))
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
					&cli.BoolFlag{
						Name:    "persist",
						Aliases: []string{"p"},
						Value:   false,
						Usage: "if the dir to be initialized was set via a flag (like --here or --dir), \n" +
							"overwrite the current user dir's configuration file's dotfiles_dir field.",
					},
				},
				MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
					{
						Required: false,
						Flags: [][]cli.Flag{
							{
								&cli.StringFlag{
									Name:    "dir",
									Aliases: []string{"d"},
									Value:   "",
									Usage:   "path of the dir to be initialized",
								},
							},
							{
								&cli.BoolFlag{
									Name:    "here",
									Aliases: []string{"H"},
									Value:   false,
									Usage:   "set the dir to be initialized to the current dir",
								},
							},
						},
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					cwd, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("Could not initialize dotfiles_dir: %w", err)
					}

					var initDir string
					// Infer initDir from the flags. Even though some flags technically override
					// others, they are mutually exclusive. The code's priority order should be
					// ignored in practice.
					if c.Bool("here") {
						initDir = cwd
					}
					if c.String("dir") != "" {
						initDir, err = paths.ResolvePath(c.String("dir"), cwd)

						if err != nil {
							return fmt.Errorf("Could not initialize dir scpecified by the --dir flag: %w", err)
						}
					}

					return ExecuteInit(initDir, c.Bool("persist"))
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
