package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	var (
		inputFile string
		outputFile string
		scale     int
		noise     int
	)

	flag.StringVar(&inputFile, "input", "", "Input CBZ file (required)")
	flag.StringVar(&outputFile, "output", "", "Output CBZ file (default: {input}_upscaled.cbz)")
	flag.IntVar(&scale, "scale", 2, "Scale factor (default: 2)")
	flag.IntVar(&noise, "noise", 2, "Noise reduction level (default: 2)")

	flag.Parse()

	// Check if input file is provided
	if inputFile == "" {
		fmt.Println("Error: --input flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Printf("Error: input file %s does not exist\n", inputFile)
		os.Exit(1)
	}

	// Set default output file if not provided
	if outputFile == "" {
		outputFile = fmt.Sprintf("%s_upscaled.cbz", inputFile)
	}

	// Open the CBZ file as a zip archive
	reader, err := zip.OpenReader(inputFile)
	if err != nil {
		fmt.Printf("Error: failed to open CBZ file: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	// Collect image files
	var imageFiles []zip.File
	for _, file := range reader.File {
		// Check if file has an image extension
		ext := strings.ToLower(filepath.Ext(file.Name))
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".bmp" {
			imageFiles = append(imageFiles, *file)
		}
	}

	// Sort image files lexicographically
	sort.Slice(imageFiles, func(i, j int) bool {
		return imageFiles[i].Name < imageFiles[j].Name
	})

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "comic-upscaler-")
	if err != nil {
		fmt.Printf("Error: failed to create temporary directory: %v\n", err)
		os.Exit(1)
	}

	// Extract image files to temporary directory
	for _, file := range imageFiles {
		// Open the file from the zip archive
		src, err := file.Open()
		if err != nil {
			fmt.Printf("Error: failed to open file %s from archive: %v\n", file.Name, err)
			os.Exit(1)
		}

		// Create destination file
		dst, err := os.Create(filepath.Join(tempDir, file.Name))
		if err != nil {
			src.Close()
			fmt.Printf("Error: failed to create file %s: %v\n", file.Name, err)
			os.Exit(1)
		}

		// Copy file content
		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()
		if err != nil {
			fmt.Printf("Error: failed to copy file %s: %v\n", file.Name, err)
			os.Exit(1)
		}
	}

	// Print summary and exit as placeholder
	fmt.Printf("Extracted %d images to temp directory: %s\n", len(imageFiles), tempDir)
	os.Exit(0)
}