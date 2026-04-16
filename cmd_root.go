package main

import (
	"fmt"
	"os"

	"github.com/dotcommander/statusline/internal/configtui"
	"github.com/dotcommander/statusline/internal/setupcmd"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "statusline",
		Short: "Two-line ANSI status bar for Claude Code",
		Long: "statusline renders a Tokyo Night themed status bar from JSON piped on stdin.\n" +
			"Run with no arguments and piped input to render. Use subcommands to configure.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run:           func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
	}
	root.AddCommand(newConfigCmd())
	root.AddCommand(newSetupCmd())
	return root
}

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Edit the statusline config in an interactive TUI",
		RunE:  func(cmd *cobra.Command, args []string) error { return configtui.Run() },
	}
}

func newSetupCmd() *cobra.Command {
	var local bool
	c := &cobra.Command{
		Use:   "setup",
		Short: "Configure Claude Code's settings.json to use statusline",
		RunE: func(cmd *cobra.Command, args []string) error {
			scope := "global"
			if local {
				scope = "local"
			}
			return setupcmd.Run(scope)
		},
	}
	c.Flags().BoolVar(&local, "local", false, "configure .claude/settings.json in cwd instead of ~/.claude/settings.json")
	return c
}

func executeRoot() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
