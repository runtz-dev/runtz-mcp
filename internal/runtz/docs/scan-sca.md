---
title: SCA
description: Software Composition Analysis in runtz.
---

# SCA

SCA is the first implemented scan type in runtz.

## What it scans

The initial implementation reads npm dependencies from `package.json`:

- `dependencies`
- `devDependencies`
- `optionalDependencies`
- `peerDependencies`

The CLI extracts a concrete semantic version from each dependency range when possible.

## Advisory lookup

For dependencies with a resolvable version, the CLI queries GitHub Global Security Advisories with:

```txt
ecosystem=npm
affects=package@version
```

The returned advisories are normalized into the scan result and sent to the backend.

## Stored result

The backend stores:

- Project name
- Workspace
- Source path
- Target file
- Dependencies
- Vulnerabilities
- Severity summary
- Scan timestamp

The frontend uses this data to render the SCA dashboard and CVE/GHSA table.

