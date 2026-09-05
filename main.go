// Command go-bw turns a folder of photos of paper documents into flattened,
// black-and-white "scanned" images, similar to iOS Notes.app's document
// scanner.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sonar-Support/go-bw/internal/scan"
)

var supportedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

func main() {
	inputDir := flag.String("input", "", "folder of input photos (required)")
	outputDir := flag.String("output", "", "folder to write scanned output images to (required)")
	format := flag.String("format", "jpg", "output image format: jpg or png")
	verbose := flag.Bool("v", false, "verbose per-file progress/warnings")
	flag.Parse()

	if *inputDir == "" || *outputDir == "" {
		fmt.Fprintln(os.Stderr, "usage: go-bw -input <dir> -output <dir> [-format jpg|png] [-v]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	if *format != "jpg" && *format != "jpeg" && *format != "png" {
		log.Fatalf("unsupported -format %q: must be jpg or png", *format)
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		log.Fatalf("creating output dir: %v", err)
	}

	entries, err := os.ReadDir(*inputDir)
	if err != nil {
		log.Fatalf("reading input dir: %v", err)
	}

	opts := scan.Options{Verbose: *verbose}

	processed, skipped := 0, 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !supportedExts[ext] {
			if *verbose {
				log.Printf("skipping %s: unsupported file type", entry.Name())
			}
			continue
		}

		inPath := filepath.Join(*inputDir, entry.Name())
		outName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())) + "." + *format
		outPath := filepath.Join(*outputDir, outName)

		if *verbose {
			log.Printf("processing %s", entry.Name())
		}

		if err := scan.ProcessImage(inPath, outPath, opts); err != nil {
			log.Printf("skipping %s: %v", entry.Name(), err)
			skipped++
			continue
		}
		processed++
	}

	log.Printf("done: processed %d, skipped %d", processed, skipped)
}
