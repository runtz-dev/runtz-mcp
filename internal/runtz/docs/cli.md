---
title: CLI
description: Use the runtz CLI to run scans.
---

# CLI

The runtz CLI is a single static Go binary. The implemented scan commands are
`sca`, `sast`, `host`, `container` and `k8s`, plus `login`/`logout`/`whoami`
for authentication and `update`/`version`.

Install it with:

```bash
curl -fsSL https://runtz.dev/install.sh | bash
```

On Windows:

```powershell
irm https://runtz.dev/install.ps1 | iex
```

## Authentication

Every scan authenticates with a workspace token generated in the platform
(`rtz_live_...`). Log in once and scan commands stop needing `--token` —
`runtz login` verifies the token and stores it in
`~/.config/runtz/config.json` (permissions `0600`):

```bash
runtz login                                  # paste the token at a hidden prompt
runtz login --token rtz_live_...             # non-interactive
runtz login --endpoint http://localhost:8080 # self-hosted: endpoint is stored too
runtz whoami                                 # workspace + where the token comes from
runtz logout                                 # remove the stored token
```

The token is resolved in this order — an explicit flag or environment
variable (the CI/CD path) always beats the stored login:

1. `--token` flag
2. `RUNTZ_TOKEN` environment variable
3. `runtz login` stored token

`--endpoint` defaults to the Runtz SaaS engine (`https://engine.runtz.dev`)
and is only needed for self-hosted deployments.

## Run an SCA scan

Point it at a repository root (every supported manifest is discovered) or a
single manifest file:

```bash
runtz sca ./
runtz sca ./package.json --token rtz_live_...   # explicit token (CI/CD)
```

Expected output:

```txt
SCA scan completed and sent to Runtz Platform.
Project: frontend
Manifests: 1
Dependencies: 42
Vulnerabilities: 0
```

## Run a SAST scan

The SAST scanner uses local static rules for high-signal checks such as
committed secrets, dynamic code execution, disabled TLS verification and weak
hash usage.

```bash
runtz sast ./
runtz sast ./services/payments --project payments-api
```

## Run a host package scan

The host scanner inventories the packages installed on the current machine
(dpkg, rpm, apk, pacman or Homebrew) and matches them against OSV.

```bash
runtz host
```

Use `--hostname` when you want the scan to appear under a specific hostname in
the platform. Use `--rootfs` only when scanning packages from another mounted
root filesystem.

## Run a container package scan

The container scanner pulls and reads the image layers directly — no Docker
needed. It does not call Trivy or Grype.

```bash
runtz container ubuntu:22.04
```

For an image that exists only in the local Docker daemon, add `--local`:

```bash
runtz container myapp:latest --local
```

## Run a Kubernetes scan

The Kubernetes scanner uses your current `kubectl` connection by default. The
user running the command must already be authenticated to a cluster.

```bash
runtz k8s
```

Scope a live cluster scan to one namespace:

```bash
runtz k8s --context production --namespace payments
```

Pass a path when you want to scan YAML/JSON manifests from a repo or rendered
chart instead of the live cluster:

```bash
runtz k8s ./helm/runtz/templates
```

## Flags

| Flag | Description | Default |
| --- | --- | --- |
| `--target` | Kubernetes target name shown in the platform | current context or directory name |
| `--kubectl` | kubectl binary path for Kubernetes scans | `kubectl` |
| `--kubeconfig` | kubeconfig path for Kubernetes scans | `KUBECONFIG` |
| `--context` | Kubernetes context override | current context |
| `--namespace` | Namespace to scan instead of all namespaces | unset |
| `--all-namespaces` | Scan all namespaces when `--namespace` is not set | `true` |
| `--rootfs` | Host root filesystem path whose package database is scanned | `/` |
| `--hostname` | Hostname override for host scans | local hostname |
| `--local` | Read the container image from the local Docker daemon instead of a registry | `RUNTZ_CONTAINER_LOCAL` |
| `--endpoint` | Backend endpoint (self-hosted only) | `https://engine.runtz.dev` |
| `--token` | Token generated in the platform | stored `runtz login` token |
| `--project` | Project name override | `RUNTZ_PROJECT` |
| `--source` | Source path or repository URL | `RUNTZ_SOURCE` |
| `--github-token` | Optional GitHub token for higher API limits | `GITHUB_TOKEN` |
| `--osv-url` | Optional OSV API base URL for host/container CVE matching | `https://api.osv.dev` |

Severity gates for CI/CD are available on every scan command:
`--critical-threshold N`, `--high-threshold N`, `--medium-threshold N`,
`--low-threshold N` fail the run with exit code 3 when reached.

## Advisory sources

The SCA command uses GitHub Global Security Advisories and queries npm advisories with the `affects=package@version` filter.

Reference: [GitHub REST API endpoints for global security advisories](https://docs.github.com/en/rest/security-advisories/global-advisories).

The host and container commands read `etc/os-release` and the OS package database, normalize installed packages and query OSV for package CVEs.

The SAST and Kubernetes commands send normalized findings to the backend and use the same workspace token model as the other scan types.
