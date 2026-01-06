package commands

import (
	"path/filepath"

	"github.com/hitzhangjie/gitbook/ebook"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewMOBICommand creates the mobi command
func NewMOBICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mobi [book] [output]",
		Short: "Build a mobi from a book",
		Long:  "Generate a MOBI file from your book",
		Run: func(cmd *cobra.Command, args []string) {
			runCommand("mobi", cmd.Flags(), args)
		},
	}
}

func handleMOBI(bookRoot string, fset *pflag.FlagSet, args []string) error {
	outputPath := "book.mobi"
	if len(args) > 0 {
		outputPath = args[0]
	}

	outputDir := filepath.Join(bookRoot, "_book")
	gen, err := ebook.NewGenerator(bookRoot, outputDir, "mobi")
	if err != nil {
		return err
	}

	return gen.Generate(outputPath)
}
