# Plan: GNU-Standard Column Alignment & Line Wrapping

## Overview
The goal is to implement standard GNU-style two-column formatting for command and option listings in `clihelp`. When commands or options have long signatures/names (e.g., `split <source_label> <pattern> <target_label>`), they should not inflate the indentation column for all other items.

## Key Changes

### 1. Column Threshold (`DefaultMaxColIndent = 24`)
- Capped column width at 24 columns for description text alignment (`colIndent` in `render.go`).
- Items whose label + 4 spaces fit within 24 columns determine the shared column indent.
- Items exceeding 24 columns break to their own line, with description text beginning on the following line at the standard column indent.

### 2. Next-Line Wrapping Logic (`reflowSegment`)
- Updated `reflowSegment` in `render.go` to check if `visualLen("  " + prefix) + 2 > indent`.
- If the prefix exceeds the indent column, it prints `"  " + prefix` on its own line and indents the description on the next line at `indent`.
- Short prefixes align side-by-side with their descriptions as before.

### 3. Test Oracle Synchronization
- Update `oracleColIndent` and `oracleDetailedUsage` / `oracleGlobalUsage` in `example/mail_cli_fake/mail_cli_fake_test.go` to mirror `clihelp`'s column capping and wrapping.
