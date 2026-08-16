# set

Set configuration attribute values

## Usage

```
podctl config set <attribute> <value> [options]
```

## Subcommands

- [time](config-set-time.md) — Set max execution timeout or timestamp window
- [space](config-set-space.md) — Set maximum disk space allocation or cache budget in MB
- [location](config-set-location.md) — Set geographic storage region or default output zone ID

## Flags

- `--global` — Apply setting across all system profiles
- `--persist` — Save attribute setting permanently to config.json

## Examples

- `podctl config set location 5`
- `podctl config set time 120`
- `podctl config set space 500`
