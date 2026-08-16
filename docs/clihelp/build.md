---
title: podctl build
---

# podctl build

Compile & package audio episodes with metadata

## Usage

```
podctl build [options] <source-file>
```

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output PATH` | Write compiled MP3 output to specified PATH |
| `-b, --bitrate KBPS` | Set target audio encoding bitrate in kbps (default: 192) |
| `--normalize` | Apply LUFS loudness normalization filter across tracks |
| `--tags TAGS` | Embed ID3 metadata tags (e.g. title, artist, album, year) |
| `-v, --verbose` | Enable verbose ffmpeg build output logs |

## Examples

- `podctl build episode01.wav`
- `podctl build -o ep01.mp3 --bitrate 320 --normalize`
- `podctl build -o dist/ep01.mp3 --tags 'title=Ep1,artist=Podcast' episode01.wav`

---

[↑ podctl](index.md) — [nav](nav.md)
