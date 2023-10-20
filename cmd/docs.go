/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/xigxog/kubefox-cli/internal/log"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Run:   generateDocs,
	Short: "Generate docs for 🦊 Fox",
	Long: `
Run this command to automatically generate 🦊 Fox documentation. Output is 
placed in the subdirectory docs of the working directory.`,
}

func init() {
	rootCmd.AddCommand(docsCmd)
}

func generateDocs(cmd *cobra.Command, args []string) {
	log.Verbose("Generating docs")
	docsDir := "docs"

	// ensure dir exists
	if err := os.MkdirAll(docsDir, os.ModePerm); err != nil {
		log.Fatal("Error creating docs dir '%s': %v", docsDir, err)
	}

	// remove any existing markdown files
	mdFiles, err := filepath.Glob("docs/*.md")
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
