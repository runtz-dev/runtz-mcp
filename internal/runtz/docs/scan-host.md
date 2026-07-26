---
title: Host scanning
description: Scan installed packages on a host.
---

# Host scanning

Host scanning inventories installed packages from a dpkg-based machine and sends the normalized result to the runtz backend.

## What it scans

The first implementation reads:

- `/etc/os-release`
- `/var/lib/dpkg/status`

It supports Ubuntu/Debian style package inventories and does not scan arbitrary files.

## Run a scan

```bash
cd cli
go run ./cmd/runtz host \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

Use `--hostname` to choose the name shown in the Hosts dashboard:

```bash
go run ./cmd/runtz host \
  --hostname ubuntu-prod-01 \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

Use `--rootfs /mnt/ubuntu-root` only when scanning the package database from another mounted root filesystem.

## CVE matching

The scanner maps the OS release to an OSV ecosystem such as `Ubuntu:22.04:LTS`, reads installed source package versions from dpkg metadata and queries OSV for affected CVEs.
