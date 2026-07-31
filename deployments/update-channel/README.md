# Update channel (Azure)

Static HTTPS hosting for Equate appliance `.eqa` packages and channel manifests.
See [`docs/releases/appliance-updates.md`](../../docs/releases/appliance-updates.md).

Examples:

- [`examples/manifest.stable.json`](examples/manifest.stable.json)
- [`examples/update-channel.conf`](examples/update-channel.conf)
- Schema: [`channel-manifest.schema.json`](channel-manifest.schema.json)

Publish:

```bash
./appliance/scripts/publish-update-channel-azure.sh \
  --storage-account <account> \
  --container updates \
  --channel stable \
  --edition standard \
  --arch amd64 \
  --version <semver>
```
