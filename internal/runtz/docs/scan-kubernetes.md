---
title: Kubernetes scanning
description: Kubernetes cluster scanning in runtz.
---

# Kubernetes scanning

Kubernetes scanning uses `kubectl` against the connected cluster by default and sends posture findings to the runtz backend.

## Prerequisites

The machine running the CLI must have:

- `kubectl` installed.
- A valid kubeconfig.
- Access to the target cluster.

Confirm the active context before scanning:

```bash
kubectl config current-context
```

## What it scans

The first cluster scanner runs `kubectl get` for common workload, network and RBAC resources. By default it scans all namespaces for namespaced resources and also reads cluster-scoped RBAC resources.

Initial checks include:

- Privileged containers and missing non-root controls.
- Privilege escalation not disabled.
- Mutable image tags such as `latest`.
- `hostNetwork`, `hostPID`, `hostIPC` and `hostPath` usage.
- Default service accounts and automounted service account tokens.
- Missing resource requests or limits.
- Services exposed as `LoadBalancer` or `NodePort`.
- Ingress resources without TLS.
- RBAC bindings to `cluster-admin` and wildcard RBAC rules.

## Run a cluster scan

After a one-time `runtz login`:

```bash
runtz k8s
```

Scan a specific context and namespace:

```bash
runtz k8s --context production --namespace payments
```

Use `--kubeconfig` when the kubeconfig is not in the default location:

```bash
runtz k8s --kubeconfig ~/.kube/prod.yaml
```

## Scan manifests instead

Pass a path when you want to scan YAML/JSON manifests from a repository or rendered chart instead of a live cluster:

```bash
runtz k8s ./helm/runtz/templates --target production-manifests
```

## Stored result

The backend stores the target, source, Kubernetes resources scanned, optional manifest files scanned, findings, severity summary and timestamp. The token generated in the platform identifies the workspace automatically.
