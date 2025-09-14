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
	var gpuId string
	flag.StringVar(&gpuId, "gpu-id", "auto", "GPU ID (-1=cpu, 0,1,... or comma-separated for multi-GPU; default auto-detect)")
	var threads string
	flag.StringVar(&threads, "threads", "1:2:2", "Threads for load:proc:save (default 1:2:2)")

	flag.Parse()

	// Validate scale and noise parameters
	if scale <= 0 {
		fmt.Println("Scale must be >0")
		os.Exit(1)
	}
	if noise < 0 {
		fmt.Println("Noise must be >=0")
		os.Exit(1)
	}

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

	// Auto-detect GPUs if gpuId is set to "auto"
	if gpuId == "auto" {
		cmd := exec.Command("nvidia-smi", "-L")
		outputBytes, err := cmd.Output()
		if err != nil {
			fmt.Printf("nvidia-smi not found or failed: %v; assuming 1 GPU\n", err)
			gpuId = "0"
			fmt.Printf("Detected %d GPUs, using IDs: %s\n", 1, gpuId)
		} else {
			output := string(outputBytes)
			lines := strings.Split(output, "\n")
			numGpus := 0
			for _, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "GPU ") {
					numGpus++
				}
			}
			if numGpus == 0 {
				gpuId = "-1"
				fmt.Println("No GPUs detected, using CPU")
			} else if numGpus == 1 {
				gpuId = "0"
			} else {
				gpuIds := make([]string, numGpus)
				for i := 0; i < numGpus; i++ {
					gpuIds[i] = strconv.Itoa(i)
				}
				gpuId = strings.Join(gpuIds, ",")
			}
			fmt.Printf("Detected %d GPUs, using IDs: %s\n", numGpus, gpuId)
		}
	}

	// Dynamically adjust threads for multi-GPU
	if threads == "1:2:2" { // Only adjust if using default threads
		// Count GPUs by splitting gpuId
		gpuIds := strings.Split(gpuId, ",")
		numGpus := len(gpuIds)
		if numGpus > 1 {
			procParts := make([]string, numGpus)
			saveParts := make([]string, numGpus)
			for i := 0; i < numGpus; i++ {
				procParts[i] = "2"
				saveParts[i] = "2"
			}
			threads = "1:" + strings.Join(procParts, ",") + ":" + strings.Join(saveParts, ",")
		} else {
			threads = "1:2:2"
		}
		fmt.Printf("Adjusted threads for %d GPUs: %s\n", numGpus, threads)
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

	// Check if any images were found
	if len(imageFiles) == 0 {
		fmt.Println("No image files found in CBZ")
		os.Exit(1)
	}

	// Check if waifu2x binary exists
	_, err = exec.LookPath("waifu2x-ncnn-vulkan")
	if err != nil {
		fmt.Printf("waifu2x-ncnn-vulkan not found in PATH: %v\n", err)
		fmt.Println("Install from https://github.com/nihui/waifu2x-ncnn-vulkan")
		os.Exit(1)
	}

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
		args := []string{"-i", inputPath, "-o", outputPath, "-s", strconv.Itoa(scale), "-n", strconv.Itoa(noise), "-x", "-g", gpuId, "-j", threads, "-v"}
		cmd := exec.Command("waifu2x-ncnn-vulkan", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to upscale %s: %v\n", file.Name, err)
			os.Exit(1)
		}

		// Check if output file was created
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			fmt.Printf("Upscale failed for %s: output not created\n", file.Name)
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

	// Check if output file already exists
	if _, err := os.Stat(outputFile); err == nil {
		fmt.Println("Output file exists, remove it first")
		os.Exit(1)
	}

	// Check if parent directory exists, create if not
	outputDir := filepath.Dir(outputFile)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("Failed to create output directory %s: %v\n", outputDir, err)
			os.Exit(1)
		}
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