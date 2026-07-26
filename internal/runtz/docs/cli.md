---
title: CLI
description: Use the runtz CLI to run scans.
---

# CLI

The runtz CLI is written in Go. The implemented commands are `sca`, `sast`, `host`, `container` and `k8s`.

Every scan command requires `--endpoint` and `--token`. For the Runtz SaaS engine, use:

```txt
https://engine.runtz.dev
```

## Run an SCA scan

From the `runtz/` project root:

```bash
cd cli
go run ./cmd/runtz sca \
  --file ../frontend/package.json \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

Expected output:

```txt
SCA scan completed and sent to Runtz Platform.
Project: frontend
Dependencies: 42
Vulnerabilities: 0
```

## Run a SAST scan

The first SAST scanner uses local static rules for high-signal checks such as committed secrets, dynamic code execution, disabled TLS verification and weak hash usage.

```bash
cd cli
go run ./cmd/runtz sast \
  --path ../frontend/src \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

## Run a host package scan

The first host scanner reads the package database from a dpkg-based root filesystem.

```bash
cd cli
go run ./cmd/runtz host \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

Use `--hostname` when you want the scan to appear under a specific hostname in the platform. Use `--rootfs` only when scanning packages from another mounted root filesystem.

## Run a container package scan

The first container scanner pulls and reads the image layers directly. It does not call Trivy or Grype.

```bash
cd cli
go run ./cmd/runtz container \
  --image ubuntu:22.04 \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

For an image that exists only in the local Docker daemon, add `--local`:

```bash
cd cli
go run ./cmd/runtz container \
  --image myapp:latest \
  --local \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

## Run a Kubernetes scan

The Kubernetes scanner uses your current `kubectl` connection by default. The user running the command must already be authenticated to a cluster.

```bash
cd cli
go run ./cmd/runtz k8s \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

Scope a live cluster scan to one namespace:

```bash
go run ./cmd/runtz k8s \
  --context production \
  --namespace payments \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

Use `--path` when you want to scan YAML/JSON manifests from a repo or rendered chart instead of the live cluster:

```bash
go run ./cmd/runtz k8s \
  --path ../helm/runtz/templates \
  --endpoint https://engine.runtz.dev \
  --token rtz_live_...
```

## Flags

| Flag | Description | Default |
| --- | --- | --- |
| `--file` | SCA path to `package.json` | `package.json` |
| `--path` | Source path for SAST, or optional manifest path for Kubernetes | SAST: `.`, Kubernetes: live cluster |
| `--target` | Kubernetes target name shown in the platform | current context or directory name |
| `--kubectl` | kubectl binary path for Kubernetes scans | `kubectl` |
| `--kubeconfig` | kubeconfig path for Kubernetes scans | `KUBECONFIG` |
| `--context` | Kubernetes context override | current context |
| `--namespace` | Namespace to scan instead of all namespaces | unset |
| `--all-namespaces` | Scan all namespaces when `--namespace` is not set | `true` |
| `--rootfs` | Host root filesystem path whose dpkg package database is scanned | `/` |
| `--hostname` | Hostname override for host scans | local hostname |
| `--image` | Container image reference | `RUNTZ_CONTAINER_IMAGE` |
| `--local` | Read the container image from the local Docker daemon instead of a registry | `RUNTZ_CONTAINER_LOCAL` |
| `--endpoint` | Backend endpoint | required |
| `--token` | Token generated in the platform | required |
| `--project` | Project name override | `RUNTZ_PROJECT` |
| `--source` | Source path or repository URL | `RUNTZ_SOURCE` |
| `--github-token` | Optional GitHub token for higher API limits | `GITHUB_TOKEN` |
| `--osv-url` | Optional OSV API base URL for host/container CVE matching | `https://api.osv.dev` |

## Advisory sources

The SCA command uses GitHub Global Security Advisories and queries npm advisories with the `affects=package@version` filter.

Reference: [GitHub REST API endpoints for global security advisories](https://docs.github.com/en/rest/security-advisories/global-advisories).

The host and container commands read `etc/os-release` and `var/lib/dpkg/status`, normalize installed packages and query OSV for Ubuntu/Debian package CVEs.

The SAST and Kubernetes commands send normalized findings to the backend and use the same workspace token model as the other scan types.
