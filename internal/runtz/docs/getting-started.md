---
title: Getting Started
description: Run the runtz platform with Docker Compose or Helm.
---

# Getting Started

runtz can run as a self-hosted stack with Docker Compose or as a Kubernetes
deployment with Helm.

## Requirements

- Docker and Docker Compose for local runs.
- Kubernetes and Helm for cluster installs.

## Docker Compose

Download the compose file and start the platform in detached mode:

```bash
curl -fsSL https://runtz.dev/home/docker-compose.yml -o docker-compose.yml
docker compose up -d
```

No secrets are required: browser sessions, API keys and email login codes are
issued and stored by the engine itself.

Open the platform:

```txt
http://localhost:3000
```

The backend health endpoint is available at:

```txt
http://localhost:8080/health
```

MongoDB is reachable only from the backend over the compose network; it is
not published to the host. If `8080` or `3000` are already in use, change
`BACKEND_PORT` and `FRONTEND_PORT` in `.env`.

## Helm

Add the runtz chart repository, update it, and install the platform:

```bash
helm repo add runtz https://helm.runtz.dev
helm repo update
helm upgrade --install runtz runtz/runtz \
  --namespace runtz \
  --create-namespace
```

## First access

On the first access, runtz asks for:

- `Admin Username`
- `Password`
- `Workspace Name`

After this setup, the admin user can create workspaces and manage users in `Settings`.

## Self-hosted Pro and Enterprise activation

Self-hosted Free runs fully inside your infrastructure. Pro and Enterprise are
activated from the in-app `Settings -> Billing` screen.

The self-hosted engine does not need Stripe keys. It starts Checkout through the
central engine, redirects the admin to Stripe, then returns to the same
installation and activates the license automatically. The only network
requirement is outbound HTTPS access to:

```bash
https://engine.runtz.dev
```

Each license can activate one installation. The engine keeps scan data local and
sends only the Stripe checkout session plus installation id to the central
validation endpoint for activation and heartbeat.

Self-hosted Google and GitHub authentication are Pro/Enterprise features. After
activation, configure your own OAuth apps and set `GOOGLE_CLIENT_ID`,
`NEXT_PUBLIC_GOOGLE_CLIENT_ID`, `GITHUB_CLIENT_ID`, and `GITHUB_CLIENT_SECRET`.

## Environment variables

```bash
PORT=8080
MONGODB_URI=mongodb://mongodb:27017
MONGODB_DATABASE=runtz
RUNTZ_DEPLOYMENT_MODE=self-hosted
RUNTZ_PUBLIC_URL=http://localhost:3000
CORS_ALLOWED_ORIGINS=http://localhost:3000
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Central/cloud deployments also need Stripe and the private license signing key:

```bash
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_PRO_CLOUD=price_...
STRIPE_PRICE_ENTERPRISE_CLOUD=price_...
STRIPE_PRICE_PRO_SELF_HOSTED=price_...
STRIPE_PRICE_ENTERPRISE_SELF_HOSTED=price_...
RUNTZ_LICENSE_PRIVATE_KEY=base64-ed25519-private-key
```
