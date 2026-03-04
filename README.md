# epub2mp3

A fast, pure Go EPUB to M4A audiobook converter for macOS.

## Features

- ✅ **Pure Go** - No Python, no external dependencies (except MP4Box)
- ✅ **Single binary** - Fast startup, minimal overhead
- ✅ **Parallel processing** - Auto-detects optimal worker count for your CPU
- ✅ **Multi-language** - Works with any macOS TTS voice
- ✅ **Smart text extraction** - Parses EPUB structure, extracts clean text
- ✅ **Split output** - Optionally split long books into hourly files
- ✅ **Real-time progress** - Shows chunk progress with memory monitoring
- ✅ **Automatic logging** - Full conversion logs saved automatically
- ✅ **Organized output** - All files in dedicated audiobook directory
- ✅ **30s timeout** - Won't hang forever if `say` gets stuck

## Requirements

- **macOS** (uses built-in `say` command for TTS)
- **MP4Box** (for M4A container): `brew install gpac`
- Go 1.21+ (for building)
- **Offline capable** - No internet required!

## Quick Start

### GUI (Easiest!)
```bash
./epub2mp3-gui
```

Features:
- 📁 File picker for EPUB selection
- 💾 Choose output location
- 🌐 Language selection
- ⚙️ Auto-detected optimal worker count
- ✅ Split into hourly files
- 🗑️ Keep temp files (for debugging)
- 📝 Keep log file
- 📊 Progress bar

### CLI
```bash
./epub2mp3-cli -input book.epub
```

Creates `book.epub.audiobook/` directory with:
- `book.m4a` (or `book_part01.m4a`, `book_part02.m4a`, etc. if split)
- `conversion.log`
- `chunks/` (temp files, deleted on success unless `-keep-temp`)

## Usage Examples

### Basic conversion
```bash
./epub2mp3-cli -input book.epub
# Output: book.epub.audiobook/book.m4a
```

### Split long books into hourly files
```bash
# Default: ~60 minute files (~50MB each)
./epub2mp3-cli -input book.epub -split 60

# 30-minute chunks (~25MB each)
./epub2mp3-cli -input book.epub -split 30

# Single file (no splitting)
./epub2mp3-cli -input book.epub -split 0
```

### Keep files for debugging
```bash
# Keep temp chunk files
./epub2mp3-cli -input book.epub -keep-temp

# Keep log file
./epub2mp3-cli -input book.epub -keep-logs

# Keep everything
./epub2mp3-cli -input book.epub -keep-temp -keep-logs
```

### Different languages
```bash
# Spanish
./epub2mp3-cli -input libro.epub -lang es

# French
./epub2mp3-cli -input livre.epub -lang fr

# German
./epub2mp3-cli -input buch.epub -lang de

# Chinese
./epub2mp3-cli -input shu.epub -lang zh-CN
```

### Adjust worker count
```bash
# Faster (more CPU usage)
./epub2mp3-cli -input book.epub -workers 12

# Slower but less CPU
./epub2mp3-cli -input book.epub -workers 4

# Let it auto-detect (recommended)
./epub2mp3-cli -input book.epub
```

## Command-line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-input` | (required) | Input EPUB file path |
| `-output` | auto | Output M4A file path |
| `-lang` | `en` | Language code for TTS |
| `-min-chars` | `50` | Minimum characters per chapter |
| `-verbose` | `false` | Enable verbose output |
| `-workers` | auto | Parallel workers (auto: 75% of CPU cores, max 12) |
| `-log` | auto | Log file path |
| `-split` | `60` | Split output every N minutes (`0` = single file) |
| `-keep-temp` | `false` | **Keep temporary chunk files** |
| `-keep-logs` | `false` | **Keep log file after conversion** |

## GUI Checkboxes

- ☑️ **Split into hourly files** - Create ~60 min chunks (default: ON)
- ☐ **Verbose output** - Show detailed logs
- ☐ **Keep temp files** - Preserve chunk files for debugging
- ☑️ **Keep log file** - Save conversion log (default: ON)

## Output Structure

```
book.epub.audiobook/
├── book.m4a              # Final audiobook (or book_part01.m4a, etc.)
├── conversion.log        # Full conversion log
└── chunks/               # Temp files (deleted unless -keep-temp)
    ├── 00000.aac
    ├── 00001.aac
    └── ...
```

## Supported Languages

Any language supported by macOS TTS:

| Code | Language | Voice |
|------|----------|-------|
| `en` | English | Daniel |
| `es` | Spanish | Monica |
| `fr` | French | Thomas |
| `de` | German | Steffen |
| `it` | Italian | Alice |
| `pt` | Portuguese | Luciana |
| `ru` | Russian | Milena |
| `ja` | Japanese | Kyoko |
| `zh-CN` | Chinese (Simplified) | Ting-Ting |
| `zh-TW` | Chinese (Traditional) | Mei-Jia |
| `ko` | Korean | Yuna |
| `ar` | Arabic | Maged |
| `hi` | Hindi | Lekha |

## Output Format

- **Format**: M4A (AAC 64kbps, 22.05 kHz, mono)
- **File size**: ~480 KB/minute (~28 MB/hour)
- **Split files**: `book_part01.m4a`, `book_part02.m4a`, etc.

**Example** (10-hour book):
- Single file: ~280 MB
- Split hourly: 10 files × ~28 MB each

## Performance

**Speed**: ~0.15-0.5 seconds per chunk (25 seconds audio)

| CPU | Cores | Workers | Speed (chunks/min) | 10-hour book |
|-----|-------|---------|-------------------|--------------|
| M2 Max | 12 | 9 (auto) | ~400 | ~3 min |
| M1/M2 Pro | 10 | 8 (auto) | ~350 | ~4 min |
| M1/M2 | 8 | 6 (auto) | ~300 | ~5 min |
| Intel i7 | 8 | 6 (auto) | ~250 | ~6 min |
| Intel i5 | 4 | 3 (auto) | ~150 | ~10 min |

## Building

```bash
cd epub2mp3

# Build CLI
go build -o epub2mp3-cli ./cmd/cli

# Build GUI
go build -tags gui -o epub2mp3-gui ./cmd/gui
```

### Project Structure

```
epub2mp3/
├── cmd/
│   ├── cli/          # CLI entry point
│   │   └── main.go
│   └── gui/          # GUI entry point
│       └── main.go
├── internal/
│   ├── audio/        # TTS/audio processing
│   │   └── tts.go
│   ├── converter/    # Main conversion logic
│   │   └── converter.go
│   └── epub/         # EPUB parsing
│       └── epub.go
├── go.mod
└── README.md
```

## Troubleshooting

### Output file too small
If output is only a few KB instead of MB:
1. Check log file: `cat book.epub.audiobook/conversion.log`
2. Verify MP4Box: `which MP4Box`
3. Keep temp files: `-keep-temp` flag
4. Check `chunks/` directory for partial files

### Conversion hangs
The `say` command has a 30-second timeout per chunk:
1. Check Activity Monitor for stuck `say` processes
2. Kill them: `pkill -f say`
3. Restart conversion with `-keep-temp` to see which chunk failed

### Memory issues
For very large books (20+ hours):
- Use split mode: `-split 30` (smaller batches)
- Reduce workers: `-workers 4`
- Monitor log file for memory usage

### GUI shows "conversion failed"
1. Check the log file shown in error dialog
2. Verify output directory exists: `book.epub.audiobook/`
3. Check if any `.m4a` files were created
4. Try CLI with `-verbose -keep-temp` for more details

## M4A vs MP3

**Why M4A?**
- ✅ 50% smaller for same quality
- ✅ Better sound at low bitrates
- ✅ Native on iPhone/Mac/Android
- ✅ Works on Windows 10/11, VLC, most modern players

**Compatibility**: M4A works on anything made after 2015. For older devices:
```bash
ffmpeg -i book.m4a -codec:a libmp3lame -qscale:a 2 book.mp3
```

## License

MIT
