package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"epub2mp3/internal/converter"
)

func main() {
	config := converter.Config{}

	flag.StringVar(&config.InputFile, "input", "", "Input EPUB file path")
	flag.StringVar(&config.OutputFile, "output", "", "Output M4A file path")
	flag.StringVar(&config.Language, "lang", "en", "Language code for TTS")
	flag.IntVar(&config.ChapterMin, "min-chars", 50, "Minimum characters per chapter")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.IntVar(&config.Workers, "workers", 0, "Number of parallel workers (0 = auto)")
	flag.StringVar(&config.LogFile, "log", "", "Log file path")
	flag.IntVar(&config.SplitMins, "split", 60, "Split output every N minutes (0 = single file)")
	flag.BoolVar(&config.KeepTemp, "keep-temp", false, "Keep temporary chunk files")
	flag.BoolVar(&config.KeepLogs, "keep-logs", false, "Keep log files after conversion")
	flag.Parse()

	// Setup log file
	if config.LogFile == "" && config.InputFile != "" {
		baseName := strings.TrimSuffix(config.InputFile, filepath.Ext(config.InputFile))
		config.LogFile = baseName + ".log"
	}

	if !config.KeepLogs && config.LogFile != "" {
		defer os.Remove(config.LogFile)
	}

	if config.LogFile != "" {
		logWriter, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not create log file: %v\n", err)
		} else {
			defer logWriter.Close()
			config.LogWriter = logWriter
		}
	}

	// Find EPUB if not specified
	if config.InputFile == "" {
		epubs, err := converter.FindEPUBs(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(epubs) == 0 {
			fmt.Println("EPUB to M4A Converter")
			fmt.Println("\nNo EPUB files found.")
			fmt.Println("\nUsage: epub2mp3 -input <file.epub> [options]")
			flag.PrintDefaults()
			os.Exit(1)
		}
		config.InputFile = epubs[0]
		fmt.Printf("Found EPUB: %s\n", config.InputFile)
	}

	// Run conversion
	conv := converter.New(&config)
	if err := conv.Convert(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
