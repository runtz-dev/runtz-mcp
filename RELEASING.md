# Releasing runtz-mcp

Releases are driven by CI. Publishing a GitHub Release runs
`.github/workflows/runtz-mcp-pipeline-prod-k8s.yml`, which builds the public
image, pushes the Helm chart to ChartMuseum and deploys prod (`mcp.runtz.dev`).
Versions follow `1.0.0-rc1 → 1.0.0-rc2 → ... → 1.0.0`, then regular semver.

## Prerequisites (one-time, on the runtz-dev org)

- Variable `DOCKER_LOGIN`; secrets `DOCKER_PASS`, `CHARTMUSEUM_USER`,
  `CHARTMUSEUM_PASSWORD`.
- Self-hosted runners labelled `runtz-runners` and the private
  `runtzdev/deploy-k8s:v1` image.

## Cut a release

1. On `dev`, bump `VERSION` and `helm/runtz-mcp/Chart.yaml`
   (`version` + `appVersion`) to match.
2. Promote `dev → main` (PR + merge).
3. Publish the GitHub Release for tag `v$(cat VERSION)` (pre-release for `-rc`).

Publishing triggers the prod pipeline:

- builds/pushes multi-arch `runtzdev/runtz-mcp` (`:latest` for stable versions);
- pushes the chart to https://helm.runtz.dev;
- `helm upgrade --install` in the prod namespace.

The dev environment (`mcp-dev.runtz.dev`) deploys automatically on
every push to `dev`.

## Verify

```bash
curl -s https://mcp.runtz.dev/healthz
helm repo update && helm search repo runtz/runtz-mcp
```
