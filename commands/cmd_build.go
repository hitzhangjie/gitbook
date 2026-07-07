package commands

import (
	"fmt"
	"path/filepath"

	"github.com/hitzhangjie/gitbook/book"
	"github.com/hitzhangjie/gitbook/builder"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewBuildCommand creates the build command
func NewBuildCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "build [book] [output]",
		Short: "Build a gitbook from a directory",
		Long:  "Build a static website using gitbook",
		Run: func(cmd *cobra.Command, args []string) {
			runCommand("build", cmd.Flags(), args)
		},
	}
}

func handleBuild(bookRoot string, fset *pflag.FlagSet, args []string) error {
	if err := book.ValidateBookRoot(bookRoot); err != nil {
		return err
	}

	outputDir := ""
	if len(args) >= 2 {
		outputDir = args[1]
	}

	resolvedOutput := outputDir
	if resolvedOutput == "" {
		resolvedOutput = filepath.Join(bookRoot, "_book")
	}
	if absOutput, err := filepath.Abs(resolvedOutput); err == nil {
		resolvedOutput = absOutput
	}

	if len(args) == 0 {
		fmt.Printf("Using default book directory: %s\n", bookRoot)
	}
	if len(args) < 2 {
		fmt.Printf("Using default output directory: %s\n", resolvedOutput)
	}

	b, err := builder.NewBuilder(bookRoot, outputDir)
	if err != nil {
		return err
	}

	return b.Build()
}
