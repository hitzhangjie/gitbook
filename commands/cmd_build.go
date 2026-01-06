package commands

import (
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
	outputDir := ""
	if len(args) >= 2 {
		outputDir = args[1]
	}
	if outputDir == "" {
		panic("output directory is required")
	}

	builder, err := builder.NewBuilder(bookRoot, outputDir)
	if err != nil {
		return err
	}

	return builder.Build()
}
