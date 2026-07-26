---
title: Settings
description: Manage workspaces, users and profile.
---

# Settings

The `Settings` area has profile, workspace, billing and user administration tabs.

## Workspaces

Admins can view existing workspaces and create new ones. The first workspace is created during the initial setup.

## Usuários

Admins can create users with:

- Username
- Password
- Role
- Workspace
- Password change after first access

Admins can also generate an invite link preview for a user. Email delivery is not implemented yet.

## Billing

Cloud workspaces use Stripe Checkout for Pro and Enterprise subscriptions. The
Billing tab shows the current plan and opens the Stripe Customer Portal for card
changes, cancellation and invoice management.

Self-hosted deployments also use the Billing tab. Click `Comprar Pro` or
`Comprar Enterprise`, finish Stripe Checkout, and runtz returns to the same
installation to activate the license automatically. One license can activate one
installation, identified by the local installation id. The installation needs
outbound HTTPS access to `https://engine.runtz.dev` for activation and heartbeat
validation.

Pro and Enterprise include:

- Google and GitHub authentication for self-hosted deployments
- Smart email reports
- Smart alerts
- AI Alert Agent responses in Slack alert threads

Enterprise also enables multiple workspaces.

## Profile

The current user can change their own password.
