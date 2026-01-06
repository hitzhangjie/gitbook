package main

import (
	"os"

	"github.com/hitzhangjie/gitbook/commands"
	"github.com/spf13/cobra"
)

var (
	gitbookVersion string
	debug          bool
)

var rootCmd = &cobra.Command{
	Use:   "gitbook",
	Short: "CLI to generate books and documentation using gitbook",
	Long:  "The GitBook command line interface.",
}

func init() {
	commands.RegisterCommands(rootCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		commands.PrintError(err)
		os.Exit(1)
	}
}
