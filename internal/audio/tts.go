package audio

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// TTSConfig holds TTS configuration
type TTSConfig struct {
	Language string
	Voice    string
	Timeout  time.Duration
}

// DefaultConfig returns default TTS configuration
func DefaultConfig() *TTSConfig {
	return &TTSConfig{
		Language: "en",
		Voice:    "Daniel",
		Timeout:  30 * time.Second,
	}
}

// VoiceMap maps language codes to macOS voices
var VoiceMap = map[string]string{
	"en":      "Daniel",
	"es":      "Monica",
	"fr":      "Thomas",
	"de":      "Steffen",
	"it":      "Alice",
	"pt":      "Luciana",
	"ru":      "Milena",
	"ja":      "Kyoko",
	"zh-CN":   "Ting-Ting",
	"zh-TW":   "Mei-Jia",
	"ko":      "Yuna",
	"ar":      "Maged",
	"hi":      "Lekha",
}

// ConvertChunk converts a text chunk to AAC audio
func ConvertChunk(textFile, audioFile string, config *TTSConfig) error {
	// Read text to verify it's not empty
	data, err := os.ReadFile(textFile)
	if err != nil {
		return fmt.Errorf("read text file: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("empty text file")
	}

	// Get voice for language
	voice := config.Voice
	if voice == "" {
		voice = VoiceMap[config.Language]
		if voice == "" {
			voice = "Daniel"
		}
	}

	// Generate speech to temp CAF file
	cafFile := audioFile + ".caf"
	defer os.Remove(cafFile)

	done := make(chan error, 1)
	go func() {
		cmd := exec.Command("say", "-v", voice, "-o", cafFile, "-f", textFile)
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			// Try without voice specification
			cmd := exec.Command("say", "-o", cafFile, "-f", textFile)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("say command: %w", err)
			}
		}
	case <-time.After(config.Timeout):
		return fmt.Errorf("say command timed out after %v", config.Timeout)
	}

	// Convert CAF to raw AAC (ADTS format)
	cmd := exec.Command("afconvert", "-f", "adts", "-d", "aac", cafFile, audioFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("afconvert: %w", err)
	}

	return nil
}

// CombineAAC combines multiple AAC files into one
func CombineAAC(aacFiles []string, outputFile string) error {
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	for _, aacFile := range aacFiles {
		data, err := os.ReadFile(aacFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", aacFile, err)
		}
		if _, err := outFile.Write(data); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		os.Remove(aacFile)
	}

	return nil
}

// CreateM4A creates an M4A container from AAC using MP4Box
func CreateM4A(aacFile, outputFile, title string) error {
	args := []string{"-add", aacFile, "-name", "1=Audio"}
	if title != "" {
		args = append(args, "-itags", "tool="+title)
	}
	args = append(args, outputFile)

	cmd := exec.Command("MP4Box", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("MP4Box: %w", err)
	}

	os.Remove(aacFile)
	return nil
}
