package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func main() {
	guiApp := app.NewWithID("com.larryste1.epub2mp3")
	window := guiApp.NewWindow("EPUB to M4A Converter")
	window.Resize(fyne.NewSize(600, 450))

	// Input file picker
	inputLabel := widget.NewLabel("EPUB File:")
	inputPath := widget.NewEntry()
	inputPath.SetPlaceHolder("Select an EPUB file...")
	inputBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			inputPath.SetText(reader.URI().Path())
		}, window)
	})
	inputRow := container.NewBorder(nil, nil, nil, inputBtn, inputPath)

	// Output file picker
	outputLabel := widget.NewLabel("Output M4A:")
	outputPath := widget.NewEntry()
	outputPath.SetPlaceHolder("Auto-generated if empty")
	outputBtn := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			outputPath.SetText(writer.URI().Path())
		}, window)
	})
	outputRow := container.NewBorder(nil, nil, nil, outputBtn, outputPath)

	// Language selector
	langLabel := widget.NewLabel("Language:")
	langSelect := widget.NewSelect(
		[]string{"English (en)", "Spanish (es)", "French (fr)", "German (de)",
			"Italian (it)", "Portuguese (pt)", "Russian (ru)", "Japanese (ja)",
			"Chinese (zh-CN)", "Korean (ko)"},
		nil,
	)
	langSelect.SetSelected("English (en)")
	langRow := container.NewBorder(nil, nil, langLabel, nil, langSelect)

	// Workers selector
	optimalWorkers := runtime.NumCPU() * 3 / 4
	if optimalWorkers < 4 {
		optimalWorkers = 4
	}
	if optimalWorkers > 12 {
		optimalWorkers = 12
	}
	workerLabel := widget.NewLabel(fmt.Sprintf("Workers: %d (auto)", optimalWorkers))
	workerSelect := widget.NewSelect(
		[]string{"4", "6", "8", "10", "12", "Max (" + fmt.Sprintf("%d", runtime.NumCPU()) + ")"},
		nil,
	)
	workerSelect.SetSelected(fmt.Sprintf("%d", optimalWorkers))
	workerRow := container.NewBorder(nil, nil, workerLabel, nil, workerSelect)

	// Checkboxes
	verboseCheck := widget.NewCheck("Verbose output", nil)
	splitCheck := widget.NewCheck("Split into hourly files (~60 min each)", nil)
	splitCheck.SetChecked(true)
	keepTempCheck := widget.NewCheck("Keep temp files (for debugging)", nil)
	keepLogsCheck := widget.NewCheck("Keep log file", nil)
	keepLogsCheck.SetChecked(true)

	// Convert button
	var convertBtn *widget.Button
	convertBtn = widget.NewButtonWithIcon("Convert", theme.MediaPlayIcon(), func() {
		convert(inputPath.Text, outputPath.Text, langSelect.Selected, workerSelect.Selected,
			verboseCheck.Checked, splitCheck.Checked, keepTempCheck.Checked, keepLogsCheck.Checked, window, convertBtn)
	})
	convertBtn.Importance = widget.HighImportance

	// Status and progress
	statusLabel := widget.NewLabel("Ready")
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	// Layout
	content := container.NewVBox(
		inputLabel,
		inputRow,
		outputLabel,
		outputRow,
		langRow,
		workerRow,
		verboseCheck,
		splitCheck,
		keepTempCheck,
		keepLogsCheck,
		layout.NewSpacer(),
		convertBtn,
		statusLabel,
		progressBar,
	)

	window.SetContent(container.NewPadded(content))
	window.ShowAndRun()
}

func convert(inputFile, outputFile, lang, workers string, verbose, split, keepTemp, keepLogs bool, window fyne.Window, convertBtn *widget.Button) {
	if inputFile == "" {
		dialog.ShowError(fmt.Errorf("please select an EPUB file"), window)
		return
	}

	if outputFile == "" {
		outputFile = inputFile[:len(inputFile)-5] + ".m4a"
	}

	// Map language
	langMap := map[string]string{
		"English (en)": "en", "Spanish (es)": "es", "French (fr)": "fr",
		"German (de)": "de", "Italian (it)": "it", "Portuguese (pt)": "pt",
		"Russian (ru)": "ru", "Japanese (ja)": "ja", "Chinese (zh-CN)": "zh-CN",
		"Korean (ko)": "ko",
	}
	langCode := langMap[lang]

	// Parse workers
	workerCount := 8
	fmt.Sscanf(workers, "%d", &workerCount)

	// Build args
	args := []string{"-input", inputFile, "-output", outputFile, "-lang", langCode, "-workers", fmt.Sprintf("%d", workerCount)}
	if verbose {
		args = append(args, "-verbose")
	}
	if split {
		args = append(args, "-split", "60")
	} else {
		args = append(args, "-split", "0")
	}
	if keepTemp {
		args = append(args, "-keep-temp")
	}
	if keepLogs {
		args = append(args, "-keep-logs")
	}

	// Create log file in output directory
	outputDir := outputFile + ".audiobook"
	logFile := filepath.Join(outputDir, "conversion.log")
	logWriter, _ := os.Create(logFile)

	// Run conversion
	convertBtn.Disable()
	done := make(chan bool)
	go func() {
		cmd := exec.Command(os.Args[0], args...)
		output, err := cmd.CombinedOutput()

		if logWriter != nil {
			logWriter.Write(output)
			logWriter.Close()
		}

		// Use channel to signal completion
		if err != nil {
			done <- false
		} else if _, statErr := os.Stat(outputFile); os.IsNotExist(statErr) {
			done <- false
		} else {
			done <- true
		}
		_ = output // suppress unused warning
	}()

	// Handle completion
	go func() {
		success := <-done
		window.Canvas().Focus(nil)
		if !success {
			dialog.ShowError(fmt.Errorf("conversion failed\nLog: %s", logFile), window)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Created:\n%s\nLog: %s", outputFile, logFile), window)
			exec.Command("open", "-R", outputFile).Run()
		}
		convertBtn.Enable()
	}()
}
