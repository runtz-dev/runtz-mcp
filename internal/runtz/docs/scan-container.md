---
title: Container scanning
description: Scan packages that compose a container image.
---

# Container scanning

Container scanning inventories packages from a container image and sends the normalized result to the runtz backend.

## What it scans

The scanner pulls the image, reads its OCI layers and reconstructs only the package inventory files needed for the first release:

- `/etc/os-release`
- `/usr/lib/os-release`
- `/var/lib/dpkg/status`

It does not call Trivy or Grype. The initial implementation supports dpkg-based images such as Ubuntu and Debian.

## Run a scan

```bash
cd cli
go run ./cmd/runtz container \
  --image ubuntu:22.04 \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

The image name appears in the Container scanning dashboard. Click the image to see the CVEs from the latest scan.

For an image that exists only in the local Docker daemon, add `--local`:

```bash
go run ./cmd/runtz container \
  --image myapp:latest \
  --local \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

## CVE matching

The scanner maps the image OS release to an OSV ecosystem such as `Ubuntu:22.04:LTS`, reads installed source package versions from dpkg metadata and queries OSV for affected CVEs.
