---
title: Overview
description: Platform overview for runtz.
---

# runtz

runtz is a DevSecOps scans platform composed of three local services:

- `frontend`: web platform built with Next.js and shadcn/ui.
- `runtz`: Go backend engine used by the frontend and CLI.
- `cli`: Go command line scanner that runs scans and sends results to the backend.

The implemented scan types are SCA for npm `package.json` files, SAST source checks, host package scanning, container package scanning and Kubernetes cluster scanning. DAST is still listed as coming soon.

## Current capabilities

- First-run admin setup.
- Workspace creation and selection.
- User administration.
- Password change after first access.
- SCA result ingestion from the CLI.
- SCA dashboard with CVEs/GHSAs found in dependencies.
- SAST finding ingestion and dashboard.
- Host package scan ingestion and dashboard.
- Container package scan ingestion and dashboard.
- Kubernetes cluster finding ingestion and dashboard.
- Local installation with Docker Compose and MongoDB.

## Repository layout

```txt
runtz/
  frontend/  # Next.js web platform
  cli/       # Go scanner CLI
  runtz/     # Go backend engine
```
