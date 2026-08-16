# rule del \<email|title>

Remove an auto-labeling rule for a sender email address or subject prefix.

## Usage

```
mail_cli rule del <email|title>
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<email|title>` | The sender email address or subject prefix of the rule to remove. |

## Examples

- `mail_cli rule del newsletter@example.com`
- `mail_cli rule del "[Alert]"`
