package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	var (
		inputFile  string
		outputFlag string
		scale      int
		noise      int
	)

	flag.StringVar(&inputFile, "input", "", "Input CBZ file (required)")
	flag.StringVar(&outputFlag, "output", "", "Output CBZ file (default: {input}_upscaled.cbz)")
	flag.IntVar(&scale, "scale", 2, "Scale factor (default: 2)")
	flag.IntVar(&noise, "noise", 2, "Noise reduction level (default: 2)")

	flag.Parse()

	// Defer cleanup of temp directory
	var extractDir string
	defer func() {
		if extractDir != "" {
			os.RemoveAll(extractDir)
		}
	}()

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
	extractDir = tempDir // For defer cleanup

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

	// Print extraction summary
	fmt.Printf("Extracted %d images to temp directory: %s\n", len(imageFiles), tempDir)

	// Create upscaled directory
	upscaleDir := filepath.Join(tempDir, "upscaled")
	if err := os.Mkdir(upscaleDir, 0755); err != nil {
		fmt.Printf("Failed to create upscale dir: %v\n", err)
		os.Exit(1)
	}

	// Upscale each image file
	for _, file := range imageFiles {
		inputPath := filepath.Join(tempDir, file.Name)
		outputPath := filepath.Join(upscaleDir, file.Name)
		cmd := exec.Command("waifu2x-ncnn-vulkan", "-i", inputPath, "-o", outputPath, "-s", strconv.Itoa(scale), "-n", strconv.Itoa(noise), "-x")
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to upscale %s: %v\n", file.Name, err)
			os.Exit(1)
		}
	}

	// Print upscale summary
	fmt.Printf("Upscaled %d images to %s\n", len(imageFiles), upscaleDir)

	// Determine output filename
	var outputFile string
	if outputFlag == "" {
		base := filepath.Base(inputFile)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext) + "_upscaled.cbz"
		outputFile = filepath.Join(filepath.Dir(inputFile), name)
	} else {
		outputFile = outputFlag
	}

	// Create output ZIP
	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Failed to create output file %s: %v\n", outputFile, err)
		os.Exit(1)
	}
	defer outFile.Close()

	writer := zip.NewWriter(outFile)
	defer writer.Close()

	// Convert zip.File slice to string slice for consistent processing
	var imageNames []string
	for _, file := range imageFiles {
		imageNames = append(imageNames, file.Name)
	}

	for _, filename := range imageNames {
		filePath := filepath.Join(upscaleDir, filename)
		f, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("Failed to open upscaled %s: %v\n", filename, err)
			os.Exit(1)
		}
		defer f.Close()

		w, err := writer.Create(filename)
		if err != nil {
			fmt.Printf("Failed to create zip entry for %s: %v\n", filename, err)
			os.Exit(1)
		}

		if _, err := io.Copy(w, f); err != nil {
			fmt.Printf("Failed to copy %s to zip: %v\n", filename, err)
			os.Exit(1)
		}
	}

	// Print success message
	fmt.Printf("Created upscaled CBZ: %s\n", outputFile)
}