---
title: SAST
description: Static Application Security Testing in runtz.
---

# SAST

SAST scans source files locally with an initial set of static rules and sends normalized findings to the runtz backend.

## What it scans

The first scanner walks source files under `--path` and skips common generated or dependency directories such as `.git`, `node_modules`, `dist`, `build`, `vendor` and `.next`.

Initial rules detect:

- Possible committed private keys and tokens.
- Dynamic code execution such as `eval`.
- Shell execution patterns.
- Disabled TLS verification.
- Weak hash functions such as MD5 and SHA-1.

## Run a scan

After a one-time `runtz login`:

```bash
runtz sast ./src
```

Use `--project` to choose the name shown in the platform:

```bash
runtz sast ./services/payments --project payments-api
```

## Stored result

The backend stores the project, source, files scanned, findings, severity summary and timestamp. The token generated in the platform identifies the workspace automatically.
