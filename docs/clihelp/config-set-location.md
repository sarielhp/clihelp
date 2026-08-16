# location

Set geographic storage region or default output zone ID

## Usage

```
podctl config set location <id> [options]
```

## Flags

- `--zone NAME` — Specify datacenter or cloud availability zone
- `--persist` — Save setting to configuration file

## Examples

- `podctl config set location 5`
- `podctl config set location 12 --zone us-east-1 --persist`
