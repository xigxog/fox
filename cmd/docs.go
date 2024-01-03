package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/xigxog/fox/internal/log"
	"github.com/xigxog/fox/internal/utils"
)

const (
	docsDir = "docs"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Run:   generateDocs,
	Short: "Generate docs for 🦊 Fox",
	Long: strings.TrimSpace(`
Run this command to automatically generate 🦊 Fox documentation. Output is 
placed in the subdirectory docs of the working directory.
`),
}

func init() {
	rootCmd.AddCommand(docsCmd)
}

func generateDocs(cmd *cobra.Command, args []string) {
	log.Verbose("Generating docs")

	// ensure dir exists
	utils.EnsureDir(docsDir)

	// remove any existing markdown files
	mdFiles, err := filepath.Glob(docsDir + "/*.md")
	if err != nil {
		log.Fatal("Error removing existing docs: %v", err)
	}
	for _, f := range mdFiles {
		if err := os.Remove(f); err != nil {
			log.Fatal("Error removing existing doc file '%s': %v", f, err)
		}
	}

	// generate docs
	if err := doc.GenMarkdownTree(rootCmd, docsDir); err != nil {
		log.Fatal("Error generating docs: %v", err)
	}
}
