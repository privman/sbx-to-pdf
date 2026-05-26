package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/privman/sbx-to-pdf/internal/parser"
	"github.com/privman/sbx-to-pdf/internal/renderer"
)

type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

var version = "dev"

func main() {
	outPath := flag.String("out", "", "Output PDF path (default: input with .pdf extension)")
	omitSceneNums := flag.Bool("omit-scene-nums", false, "Do not render scene numbers")
	omitPageNums := flag.Bool("omit-page-nums", false, "Do not render page numbers")
	overwrite := flag.Bool("overwrite", false, "Allow overwriting an existing output file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	title := flag.String("title", "", "Title for the title page (renders ALL CAPS, underlined)")

	var byLines stringSlice
	flag.Var(&byLines, "by", "Author name (repeatable, rendered under \"Written by\")")

	var contacts stringSlice
	flag.Var(&contacts, "contact", "Contact info as Key:Value (repeatable, rendered bottom-left)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sbx2pdf <input.sbx> [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputPath := flag.Arg(0)
	if _, err := os.Stat(inputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot read input file: %s\n", err)
		os.Exit(1)
	}

	output := *outPath
	if output == "" {
		ext := filepath.Ext(inputPath)
		output = strings.TrimSuffix(inputPath, ext) + ".pdf"
	}

	if !*overwrite {
		if _, err := os.Stat(output); err == nil {
			fmt.Fprintf(os.Stderr, "Error: output file already exists: %s\nUse --overwrite to replace it.\n", output)
			os.Exit(1)
		}
	}

	f, err := os.Open(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	elements, err := parser.Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing SBX: %s\n", err)
		os.Exit(1)
	}

	opts := renderer.Options{
		OutputPath:    output,
		OmitSceneNums: *omitSceneNums,
		OmitPageNums:  *omitPageNums,
		Title:         *title,
		ByLines:       byLines,
		Contacts:      contacts,
	}

	if err := renderer.Render(elements, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering PDF: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s (%d elements)\n", output, len(elements))
}
