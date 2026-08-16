# rule add \<email> \<lbl>

Add an auto-labeling rule by sender. Emails from the specified sender address will automatically be labeled with the target label and archived (the "received" label will be removed).

## Usage

```
mail_cli rule add <email> <lbl>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<email>` | The sender email address (e.g. newsletter@example.com). |
| `<lbl>` | The target label hierarchy (e.g. "Sort/Newsletters"). |

## Examples

- `mail_cli rule add newsletter@example.com "Sort/Newsletters"`
