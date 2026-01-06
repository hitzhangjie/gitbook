package commands

import (
	"path/filepath"

	"github.com/hitzhangjie/gitbook/ebook"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewPDFCommand creates the pdf command
func NewPDFCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pdf [book] [output]",
		Short: "Build a pdf from a book",
		Long:  "Generate a PDF file from your book",
		Run: func(cmd *cobra.Command, args []string) {
			runCommand("pdf", cmd.Flags(), args)
		},
	}
}

func handlePDF(bookRoot string, fset *pflag.FlagSet, args []string) error {
	outputPath := "book.pdf"
	if len(args) > 0 {
		outputPath = args[0]
	}

	outputDir := filepath.Join(bookRoot, "_book")
	gen, err := ebook.NewGenerator(bookRoot, outputDir, "pdf")
	if err != nil {
		return err
	}

	return gen.Generate(outputPath)
}
