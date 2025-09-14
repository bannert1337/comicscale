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
- Multi-GPU with auto-threads: ./comic-upscaler --input manga.cbz --gpu-id auto (adjusts threads automatically)
- Multi-GPU auto: ./comic-upscaler --input manga.cbz --gpu-id auto --threads 2:4:4 (outputs manga_upscaled.cbz using all detected GPUs).

## Flags

- --input: Path to input CBZ (required).
- --output: Path to output CBZ (optional, default {input}_upscaled.cbz).
- --scale: Upscale factor (int >0, default 2).
- --noise: Noise reduction level (int >=0, default 2).
- --gpu-id: GPU device(s) to use (-1 for CPU, 0 for first GPU, comma-separated like 0,1 for multi-GPU; default "auto" - detects NVIDIA GPUs via nvidia-smi).
- --threads: Thread counts for load:proc:save (e.g., "1:2:2"; default "1:2:2" - auto-adjusted for multi-GPU, e.g., "1:2,2:2" for 2 GPUs when using --gpu-id auto or multi).

## Notes

- Assumes waifu2x installed; tool checks and errors if missing.
- Temp files auto-cleaned.
- Vulkan enabled for GPU if available.
- GPU auto-detection requires nvidia-smi (part of NVIDIA drivers); falls back to single GPU if unavailable, CPU if none. For AMD/Intel GPUs, set --gpu-id manually (tool uses Vulkan, supports multiple backends via waifu2x).
- For multi-GPU, default threads are auto-adjusted to match GPU count (e.g., proc/save parts per GPU); custom --threads overrides this.
- License: MIT (or add basic MIT license text if desired).

## Development

Contributed by Oak AI.