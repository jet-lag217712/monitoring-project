# Equate appliance connected updates

**Status:** Engineering  
**Audience:** Release engineering, appliance operators  
**Authority:** [`.ai/decisions/appliance-4.md`](../../.ai/decisions/appliance-4.md), [`.ai/project-context/appliance.md`](../../.ai/project-context/appliance.md)

Connected updates let a configured appliance fetch a signed `.eqa` release from
**Azure Blob Storage** (or compatible HTTPS static hosting), verify it, and apply
it through the existing `configure-vm.sh --upgrade` path. Air-gapped sites keep
using offline staging.

## Operator flow (connected)

1. Ensure `/etc/equate/update-channel.conf` exists:

```ini
channel_url=https://<storage-account>.blob.core.windows.net/updates/v1/channel/stable/manifest.json
edition=standard
```

2. Check for an update:

```bash
sudo equate upgrade --check
```

3. Apply (downloads, verifies SHA-256 + Ed25519 signature, extracts, upgrades):

```bash
sudo equate upgrade
```

4. Rollback if needed:

```bash
sudo equate upgrade --rollback
```

## Operator flow (offline)

```bash
make appliance-stage HOST=<vm> ARCH=amd64 VERSION=<semver>
# on the appliance:
sudo equate upgrade --bundle /tmp/equate-staging/bundle --version <semver>
```

If no channel config is present, `equate upgrade` also looks for a staged bundle
at `/tmp/equate-staging/bundle`.

## `.eqa` package

An `.eqa` file is a gzip-compressed tar of the offline bundle directory produced
by `make appliance-bundle` (contents of `dist/appliance-<arch>-<version>/`).

```bash
make appliance-bundle ARCH=amd64 VERSION=1.0.3
EQUATE_UPDATE_SIGNING_KEY=/secure/equate-updates.priv \
  make appliance-package ARCH=amd64 VERSION=1.0.3
```

Outputs:

| File | Purpose |
|---|---|
| `dist/Equate-<ver>-<arch>.eqa` | Artifact |
| `dist/Equate-<ver>-<arch>.eqa.sha256` | SHA-256 |
| `dist/Equate-<ver>-<arch>.eqa.sig` | Base64 Ed25519 signature |

Signature message (canonical):

```text
EQUATE-EQA-v1\n
<lowercase sha256 hex>
```

The verifying public key is embedded in `equate` (`internal/update.EmbeddedPublicKeyHex`)
and also stored at `appliance/keys/equate-updates.pub`. Regenerate with
`appliance/scripts/generate-update-keys.sh` and update both places together.
**Never commit the private key.**

## Azure publish (public-read)

Anonymous HTTPS download is supported: appliances do **not** authenticate to
Blob Storage. Integrity is enforced by SHA-256 + Ed25519 signature verification
inside `equate`, not by Azure ACLs.

### One-time Azure setup

You already created the storage account. Finish container + public access:

```bash
az login
./appliance/scripts/setup-update-channel-azure.sh \
  --storage-account <your-storage-account>
```

That enables `allowBlobPublicAccess` and creates/updates container `updates`
with `--public-access blob` (anyone can `GET` a blob URL; listing is not public).

### One-time GitHub Actions auth (OIDC)

Publishing uses a GitHub **Environment** named `update-channel` so both
release tags and manual `workflow_dispatch` runs present the same OIDC subject
to Entra (a `refs/heads/main` federated credential does **not** match
`refs/tags/v…`).

#### A. Create the GitHub Environment

1. Repo → **Settings → Environments → New environment**
2. Name: `update-channel`
3. Leave protection rules empty for now (optional later: required reviewers)
4. Save

No environment secrets are required; the workflow still reads repo-level
Actions secrets/variables.

#### B. Entra federated credential

1. In Entra ID, open the App registration used for Actions (or create one).
2. **Certificates & secrets → Federated credentials → Add credential**
3. Scenario: **GitHub Actions deploying Azure resources**
4. Fill in:

| Field | Value |
|---|---|
| Organization | `jet-lag217712` |
| Repository | `monitoring-dashboard` |
| Entity type | **Environment** |
| GitHub environment name | `update-channel` |
| Name (credential display name) | `github-update-channel` |

That produces subject:

```text
repo:jet-lag217712/monitoring-dashboard:environment:update-channel
```

Issuer: `https://token.actions.githubusercontent.com`  
Audience: `api://AzureADTokenExchange`

If you already have a credential whose subject is `…:ref:refs/heads/main`,
**keep or delete it** — it is unused by this workflow. Add the environment
credential above (do not only edit the subject to a tag; tags change every
release).

#### C. Storage RBAC + GitHub secrets

1. On the storage account → **Access control (IAM)** → add role assignment:
   - Role: **Storage Blob Data Contributor**
   - Assign to the app’s service principal
2. In the GitHub repo → **Settings → Secrets and variables → Actions**:

| Type | Name | Value |
|---|---|---|
| Variable | `AZURE_STORAGE_ACCOUNT` | your storage account name (e.g. `equateupdate`) |
| Variable | `AZURE_UPDATE_CONTAINER` | `updates` (optional; default) |
| Secret | `AZURE_CLIENT_ID` | app (client) ID |
| Secret | `AZURE_TENANT_ID` | directory (tenant) ID |
| Secret | `AZURE_SUBSCRIPTION_ID` | subscription ID |
| Secret | `EQUATE_UPDATE_SIGNING_KEY` | hex contents of `equate-updates.priv` (single line) |

### Publish via GitHub Actions

Workflow: [`.github/workflows/appliance-update-channel.yml`](../../.github/workflows/appliance-update-channel.yml)

- **Actions → Publish appliance update channel (.eqa → Azure) → Run workflow**
  - Enter version / arch / channel / edition
- Or publish a GitHub Release: the workflow runs for `amd64` / `stable` /
  `standard` automatically

Manual (local) publish still works:

```bash
EQUATE_UPDATE_SIGNING_KEY=/secure/equate-updates.priv \
  make appliance-package ARCH=amd64 VERSION=1.0.3
./appliance/scripts/publish-update-channel-azure.sh \
  --storage-account <account> \
  --container updates \
  --channel stable \
  --edition standard \
  --arch amd64 \
  --version 1.0.3
```

Blob layout:

```text
v1/channel/<channel>/manifest.json
v1/channel/<channel>/<version>/Equate-<version>-<arch>.eqa
v1/channel/<channel>/<version>/Equate-<version>-<arch>.eqa.sha256
v1/channel/<channel>/<version>/Equate-<version>-<arch>.eqa.sig
```

Use a separate channel (and `edition=noauth`) for the NoAuth appliance line.
Cross-edition updates are rejected.

After the first successful publish, point appliances at:

```text
https://equateupdate.blob.core.windows.net/updates/v1/channel/stable/manifest.json
```

On the appliance:

```bash
sudo tee /etc/equate/update-channel.conf >/dev/null <<'EOF'
channel_url=https://equateupdate.blob.core.windows.net/updates/v1/channel/stable/manifest.json
edition=standard
EOF
sudo equate upgrade --check
sudo equate upgrade
```

## Channel manifest

See [`deployments/update-channel/examples/manifest.stable.json`](../../deployments/update-channel/examples/manifest.stable.json)
and the JSON Schema at
[`deployments/update-channel/channel-manifest.schema.json`](../../deployments/update-channel/channel-manifest.schema.json).

## CLI reference

| Command | Behavior |
|---|---|
| `equate upgrade` | Channel check → confirm → download → verify → upgrade |
| `equate upgrade --check` | Report only |
| `equate upgrade --bundle … --version …` | Offline path (unchanged) |
| `equate upgrade --url … --sha256 … --signature …` | Direct artifact (testing) |
| `equate upgrade --rollback` | Previous release |
| `equate upgrade --allow-insecure-http` | Local HTTP testing only |

## Security

- Production channel and artifact URLs must be HTTPS.
- SHA-256 and Ed25519 signature must verify before extract/apply.
- Trust anchor is the public key baked into `equate`, not fetched from Azure.
- Edition mismatch fails closed.
- Upgrade/rollback application remains `configure-vm.sh`.
