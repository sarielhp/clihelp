---
title: podctl serve
---

# podctl serve

Start local development RSS feed server

## Usage

```
podctl serve [options]
```

## Flags

| Flag | Description |
|------|-------------|
| `-p, --port N` | Listen HTTP port number (default: 8080) |
| `-H, --host HOST` | Bind IP host address (default: 127.0.0.1) |
| `--tls-cert PATH` | Path to TLS public certificate file for HTTPS |
| `--tls-key PATH` | Path to TLS private key file for HTTPS |
| `--live-reload` | Automatically reload RSS feed on XML or audio updates |

## Examples

- `podctl serve`
- `podctl serve --port 9090 --live-reload`
- `podctl serve -H 0.0.0.0 -p 8443 --tls-cert cert.pem --tls-key key.pem`

---

[↑ podctl](index.md) — [nav](nav.md)
