package commands

import (
	"path/filepath"

	"github.com/hitzhangjie/gitbook/ebook"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewEPUBCommand creates the epub command
func NewEPUBCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "epub [book] [output]",
		Short: "Build an epub from a book",
		Long:  "Generate an EPUB file from your book",
		Run: func(cmd *cobra.Command, args []string) {
			runCommand("epub", cmd.Flags(), args)
		},
	}
}

func handleEPUB(bookRoot string, fset *pflag.FlagSet, args []string) error {
	outputPath := "book.epub"
	if len(args) > 0 {
		outputPath = args[0]
	}

	outputDir := filepath.Join(bookRoot, "_book")
	gen, err := ebook.NewGenerator(bookRoot, outputDir, "epub")
	if err != nil {
		return err
	}

	return gen.Generate(outputPath)
}
