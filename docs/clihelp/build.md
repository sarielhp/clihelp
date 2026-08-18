---
title: podctl build
---

# podctl build

Compile & package audio episodes with **metadata** and *ID3 tags*

## Usage

```
podctl build [options] <source-file>
```

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output PATH` | Write compiled MP3 output to specified PATH |
| `-b, --bitrate KBPS` | Set target audio encoding bitrate in kbps |
| `--[no-]normalize` | Apply LUFS loudness normalization |
| `--tags TAGS` | Embed ID3 metadata tags (e.g. title, artist) |

## Examples

- `podctl build episode01.wav`
- `podctl build -o ep01.mp3 --bitrate 320 --normalize`

## Encoding Guidelines

Use `--bitrate 320` for *highest quality* or `--bitrate 128` for **voice-only** episodes (see [Audio Encoding Guide](https://podctl.example.com/docs/audio)).

---

[↑ podctl](index.md) — [nav](nav.md)
