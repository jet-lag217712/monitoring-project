# Equate update signing keys

`equate-updates.pub` is the Ed25519 public key embedded in `equate` for verifying
`.eqa` packages. Keep it in sync with
`services/snmp-collector/internal/update.EmbeddedPublicKeyHex`.

Generate a new pair:

```bash
./appliance/scripts/generate-update-keys.sh --out-dir /secure/equate-keys
```

Commit only the `.pub` file (and update `EmbeddedPublicKeyHex`). Store the
`.priv` file in your secret store and set:

```bash
export EQUATE_UPDATE_SIGNING_KEY=/secure/equate-keys/equate-updates.priv
```

Never commit `equate-updates.priv`.
