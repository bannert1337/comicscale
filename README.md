# Comic Upscaler

A Go CLI tool to upscale images in CBZ (comic/manga archive) files using waifu2x-ncnn-vulkan for AI-based upscaling.

## Features

- Extracts images from CBZ, upscales with waifu2x (Vulkan accelerated), repackages into new CBZ.
- Supports PNG/JPG/GIF/BMP images.
- Defaults: scale=2, noise=2.

## Prerequisites

- Go 1.16+ installed.
- waifu2x-ncnn-vulkan binary in PATH (download from https://github.com/nihui/waifu2x-ncnn-vulkan).

## Installation

1. Clone repo: git clone <repo> .
2. Build: go build -o comic-upscaler .

## Usage

./comic-upscaler --input example.cbz [--output output.cbz] [--scale 2] [--noise 2]
Examples:
- Default: ./comic-upscaler --input manga.cbz (outputs manga_upscaled.cbz)
- Custom: ./comic-upscaler --input manga.cbz --output high-res.cbz --scale 3 --noise 1

## Flags

- --input: Path to input CBZ (required).
- --output: Path to output CBZ (optional, default {input}_upscaled.cbz).
- --scale: Upscale factor (int >0, default 2).
- --noise: Noise reduction level (int >=0, default 2).

## Notes

- Assumes waifu2x installed; tool checks and errors if missing.
- Temp files auto-cleaned.
- Vulkan enabled for GPU if available.
- License: MIT (or add basic MIT license text if desired).

## Development

Contributed by Oak AI.