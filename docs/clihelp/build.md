---
title: 'podctl build'
---

# podctl build

Compile, encode, and package raw audio into MP3 podcast episodes. Supports configurable bitrate, loudness normalization, and embedded ID3 tags for distribution across Apple Podcasts, Spotify, and Google Podcasts.

## Usage

```
podctl build [options] <source-file>
```

## Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose output logs |
| `-s, --silent` | Suppress non-error output |
| `-o, --output PATH` | Write compiled MP3 output to specified PATH |
| `-b, --bitrate KBPS` | Set target audio encoding bitrate in kbps (default: 192) |
| `--[no-]normalize` | Apply LUFS loudness normalization (default: true) |
| `--tags TAGS` | Embed ID3 metadata tags (e.g. title, artist) |

## Examples

- `podctl build episode01.wav`
- `podctl build -o ep01.mp3 --bitrate 320 --normalize`

## Encoding Guidelines

Use `--bitrate 320` for *highest quality* or `--bitrate 128` for **voice-only** episodes (see [Audio Encoding Guide](https://podctl.example.com/docs/audio)).

---

[↑ podctl](index.md) — [nav](nav.md)
