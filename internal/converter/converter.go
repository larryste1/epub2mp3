package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"epub2mp3/internal/audio"
	"epub2mp3/internal/epub"
)

// Config holds conversion configuration
type Config struct {
	InputFile    string
	OutputFile   string
	Language     string
	ChapterMin   int
	Verbose      bool
	Workers      int
	SplitMins    int
	LogFile      string
	LogWriter    *os.File
	KeepTemp     bool // Keep temp chunk files
	KeepLogs     bool // Keep log files
}

// ProgressCallback is called with progress updates
type ProgressCallback func(stage string, current, total int)

// Converter handles EPUB to M4A conversion
type Converter struct {
	config   *Config
	progress ProgressCallback
}

// New creates a new Converter
func New(config *Config) *Converter {
	// Auto-detect workers if not specified
	if config.Workers <= 0 {
		config.Workers = runtime.NumCPU() * 3 / 4
		if config.Workers < 4 {
			config.Workers = 4
		}
		if config.Workers > 12 {
			config.Workers = 12
		}
	}

	// Default output file
	if config.OutputFile == "" {
		config.OutputFile = strings.TrimSuffix(config.InputFile, filepath.Ext(config.InputFile)) + ".m4a"
	}

	return &Converter{
		config: config,
	}
}

// SetProgress sets the progress callback
func (c *Converter) SetProgress(cb ProgressCallback) {
	c.progress = cb
}

// Convert runs the conversion
func (c *Converter) Convert() error {
	cfg := c.config

	c.log("Opening EPUB: %s\n", cfg.InputFile)

	// Parse EPUB
	epubBook, err := epub.Parse(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("parse EPUB: %w", err)
	}

	c.log("  Title: %s\n", epubBook.Title)
	c.log("  Author: %s\n", epubBook.Author)
	c.log("  Content files: %d\n", len(epubBook.ContentFiles))

	// Extract text
	c.log("Extracting text content...\n")
	text, err := epubBook.ExtractText(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("extract text: %w", err)
	}

	text = epub.PrepareText(text)
	if len(text) < cfg.ChapterMin {
		return fmt.Errorf("insufficient text: %d chars (min %d)", len(text), cfg.ChapterMin)
	}

	c.log("  Total characters: %d\n", len(text))

	// Convert to audio
	c.log("Converting to speech (language: %s, workers: %d)...\n", cfg.Language, cfg.Workers)
	if err := c.convertToAudio(text, cfg.OutputFile, cfg.Language); err != nil {
		return fmt.Errorf("convert to audio: %w", err)
	}

	c.log("✓ Conversion complete: %s\n", cfg.OutputFile)
	return nil
}

func (c *Converter) convertToAudio(text, outputFile, language string) error {
	cfg := c.config

	// Create output directory for all files
	outputDir := outputFile + ".audiobook"
	chunkDir := filepath.Join(outputDir, "chunks")
	os.RemoveAll(outputDir)
	os.MkdirAll(chunkDir, 0755)

	c.log("  Output directory: %s\n", outputDir)

	// Split text into chunks
	chunks := epub.SplitText(text, 500)
	total := len(chunks)
	c.log("  Splitting into %d chunks...\n", total)

	// Convert chunks in parallel
	ttsConfig := audio.DefaultConfig()
	ttsConfig.Language = language

	workChan := make(chan int, total)
	semaphore := make(chan struct{}, cfg.Workers)

	var wg sync.WaitGroup
	for w := 0; w < cfg.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range workChan {
				chunk := chunks[idx]
				txtFile := filepath.Join(chunkDir, fmt.Sprintf("%05d.txt", idx))
				aacFile := filepath.Join(chunkDir, fmt.Sprintf("%05d.aac", idx))

				// Write text
				os.WriteFile(txtFile, []byte(chunk), 0644)

				// Convert to audio
				if err := audio.ConvertChunk(txtFile, aacFile, ttsConfig); err != nil {
					c.log("  WARNING: chunk %d failed: %v\n", idx, err)
				}

				// Cleanup
				os.Remove(txtFile)

				// Release semaphore
				<-semaphore
			}
		}()
	}

	// Send work
	for i := 0; i < total; i++ {
		semaphore <- struct{}{}
		workChan <- i
		if (i+1)%50 == 0 && c.progress != nil {
			c.progress("Converting", i+1, total)
			c.log("  Chunk %d/%d (%d%%)\n", i+1, total, (i+1)*100/total)
		}
	}
	close(workChan)
	wg.Wait()

	c.log("  Completed %d chunks\n", total)

	// Combine chunks
	c.log("Combining audio streams...\n")

	// Calculate split points
	chunksPerFile := cfg.SplitMins * 60 / 25
	if cfg.SplitMins <= 0 {
		chunksPerFile = total
	}

	numFiles := (total + chunksPerFile - 1) / chunksPerFile
	c.log("  Splitting into %d file(s)...\n", numFiles)

	for fileNum := 0; fileNum < numFiles; fileNum++ {
		startChunk := fileNum * chunksPerFile
		endChunk := startChunk + chunksPerFile
		if endChunk > total {
			endChunk = total
		}

		// Determine output filename - put in output directory
		var outputName string
		if numFiles == 1 {
			outputName = filepath.Join(outputDir, filepath.Base(outputFile))
		} else {
			ext := filepath.Ext(outputFile)
			base := strings.TrimSuffix(filepath.Base(outputFile), ext)
			outputName = filepath.Join(outputDir, fmt.Sprintf("%s_part%02d%s", base, fileNum+1, ext))
		}

		// Combine this batch
		combinedAAC := filepath.Join(chunkDir, fmt.Sprintf("combined_part%d.aac", fileNum))
		var aacFiles []string
		for i := startChunk; i < endChunk; i++ {
			aacFiles = append(aacFiles, filepath.Join(chunkDir, fmt.Sprintf("%05d.aac", i)))
		}

		if err := audio.CombineAAC(aacFiles, combinedAAC); err != nil {
			return fmt.Errorf("combine AAC: %w", err)
		}

		// Create M4A
		c.log("  Creating %s (%d chunks)...\n", filepath.Base(outputName), endChunk-startChunk)
		if err := audio.CreateM4A(combinedAAC, outputName, "epub2mp3"); err != nil {
			return fmt.Errorf("create M4A: %w", err)
		}
	}

	// Cleanup temp files (or keep for debugging)
	if !cfg.KeepTemp {
		os.RemoveAll(chunkDir)
		c.log("  Cleaned up temporary files\n")
	} else {
		c.log("  Temporary files preserved in: %s\n", chunkDir)
	}

	return nil
}

func (c *Converter) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(msg)
	if c.config.LogWriter != nil {
		c.config.LogWriter.WriteString(msg)
		c.config.LogWriter.Sync()
	}
}

// FindEPUBs finds EPUB files in a directory
func FindEPUBs(dir string) ([]string, error) {
	var epubs []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".epub") {
			epubs = append(epubs, filepath.Join(dir, entry.Name()))
		}
	}

	return epubs, nil
}
