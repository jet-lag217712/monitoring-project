# Equate Appliance OVA Release Runbook

**Status:** Engineering runbook  
**Audience:** Release engineering, infrastructure engineers  
**Authority:** [`.ai/project-context/appliance.md`](../../.ai/project-context/appliance.md), [`appliance-1`](../../.ai/decisions/appliance-1.md)

The supported artifact is a **full-stack on-premises appliance**: UI/nginx, Backend API, PostgreSQL,
Mosquitto, Ingestion, and one generated collector container per configured site
on a single Debian 12 VM.

## Release gates

| Phase | Platform | Artifact |
|---|---|---|
| **1 — validate first** | VMware Fusion on Apple Silicon (ARM64) | `Equate-Appliance-<version>-arm64.ova` |
| **2 — client release** | VMware ESXi / Workstation / Fusion on x86 (AMD64) | `Equate-Appliance-<version>-amd64.ova` |

ARM64 Fusion output is an engineering acceptance gate only. Do not ship it as the
AMD64 client release. GitHub release publication is deferred; accepted outputs
are the OVA, checksum, and release manifest.

## VM baseline

Create a clean Debian 12 VM before every build attempt:

- UEFI firmware, one virtual NIC, DHCP
- Initial sizing: **4 vCPU, 8 GB RAM, 64 GB disk** (validate with
  representative site/device load before the AMD64 client release)
- No snapshots, no guest customization templates, and no pre-seeded customer data
- Fusion: use **ARM64** guests for phase 1; repeat the same process on an
  x86 VMware-capable host for AMD64

Filesystem layout on the installed appliance:

| Path | Purpose |
|---|---|
| `/opt/equate/releases/<version>` | Immutable release bundle (images, compose, migrations) |
| `/opt/equate/releases/current` | Symlink to the active release |
| `/etc/equate` | Customer-owned configuration |
| `/var/lib/equate` | PostgreSQL, collector state, backups |
| `/run/equate` | Rendered secrets, collector sockets, auth broker socket |

## 1. Build the offline release bundle

On the build host, produce an architecture-matched bundle with pinned binaries,
OCI image tars, migrations, image digests, source revision, checksums, and an
SBOM:

```bash
make appliance-bundle ARCH=arm64 VERSION=<version>
```

The bundle lands under `build/appliance/release/` and is packaged for transfer as
`Equate-Appliance-<version>-<arch>.tar.gz` plus `checksums.txt`.

## 2. Stage the bundle over SCP

From the build host, copy the checksummed bundle and installer scripts into a
clean Debian VM. Verify the remote architecture matches the bundle before
installing.

```bash
# On the build host — verify checksums locally first
shasum -a 256 -c Equate-Appliance-<version>-arm64.tar.gz.sha256

# Copy bundle and appliance scripts to the VM staging area
scp Equate-Appliance-<version>-arm64.tar.gz \
    Equate-Appliance-<version>-arm64.tar.gz.sha256 \
    appliance/scripts/configure-vm.sh \
    appliance/scripts/verify-appliance.sh \
    appliance/scripts/prepare-ova.sh \
    root@<vm-ip>:/root/equate-staging/
```

Do not copy customer configuration, SNMP communities, or operator credentials
into the build VM. Internal database, MQTT, machine, and TLS credentials are
generated on first boot.

## 3. Configure the VM

SSH into the staging VM and run the configurator. It verifies Debian 12 and
architecture, installs pinned Docker Engine and Compose, Open VM Tools, and
runtime packages, loads OCI images, installs `equate` binaries and systemd
units, creates least-privilege directories, enables the stack, and runs
post-install validation.

```bash
ssh root@<vm-ip>
cd /root/equate-staging
tar -xzf Equate-Appliance-<version>-arm64.tar.gz
./configure-vm.sh --bundle ./release --version <version>
```

On success, `appliance/scripts/verify-appliance.sh` runs automatically. Re-run
it after any manual changes:

```bash
/usr/local/lib/equate/verify-appliance.sh
```

## 4. First-boot TUI and acceptance smoke test

Power the VM off and back on (or proceed to the console on first boot after
OVA re-import). The console presents the Equate setup TUI.

**Security:** the OVA ships with **no shared or default password**. First boot
requires creation of the initial administrator. The TUI also manages additional
PAM-backed appliance users in a dedicated OS group. Those accounts authenticate
to both the restricted VM appliance interface and the local dashboard.

Complete the minimum acceptance configuration on the staging VM before
finalization:

1. Create the initial administrator and a second appliance user.
2. Configure at least two named sites with discovery CIDRs and SNMP communities.
3. Run discovery and **explicitly review** candidates before enrollment.
4. Confirm local telemetry appears in the dashboard.
5. Reboot and verify persistence.
6. Exercise a failed reconfiguration and confirm rollback.

Run the post-install verifier after the stack is configured:

```bash
/usr/local/lib/equate/verify-appliance.sh
```

## 5. Finalize for OVA export

When the staging VM passes acceptance, run the OVA-safe finalizer in
**release mode**. It stops services, verifies no customer data or default
credentials remain, removes build accounts and staging material, clears machine
identity, SSH host keys, DHCP leases, logs, and shell history, and powers off.
Finalization **fails closed** if clone-specific or sensitive material remains.

```bash
./prepare-ova.sh
```

Expected removals include `/root/equate-staging`, build SSH keys, populated
`/var/lib/equate` state, rendered secrets under `/run/equate`, and any
operator-created PAM accounts. Only immutable release content under
`/opt/equate/releases/<version>` is preserved.

## 6. Manual OVA export in VMware Fusion

After `prepare-ova.sh` completes, power off the VM:

1. Confirm the VM has **no snapshots**.
2. In Fusion: **File → Export to OVF/OVA**.
3. Choose **OVA** format and save as `Equate-Appliance-<version>-arm64.ova`.
4. Record the SHA-256 checksum:

   ```bash
   shasum -a 256 Equate-Appliance-<version>-arm64.ova > Equate-Appliance-<version>-arm64.ova.sha256
   ```

5. Archive the release manifest and image digests alongside the OVA.

The build guest is not assumed to export directly as a valid VMware OVA. Use
Fusion or another VMware-capable exporter for phase 1.

## 7. Re-import verification

Import the OVA into a **new** Fusion VM (do not boot the export source). Use a
fresh VM name and disk.

### Artifact checks (build host)

```bash
appliance/scripts/verify-ova-import.sh --artifact Equate-Appliance-<version>-arm64.ova
```

### First-boot checks (re-imported VM)

Power on the imported VM and run from the guest (or over SSH once the initial
administrator exists):

```bash
/usr/local/lib/equate/verify-ova-import.sh
```

After completing the first-boot TUI and site configuration:

```bash
/usr/local/lib/equate/verify-ova-import.sh --configured
/usr/local/lib/equate/verify-appliance.sh
```

### Re-import checklist

- [ ] OVA checksum matches the published `*.sha256`
- [ ] Imported VM architecture matches the bundle (`arm64` for phase 1)
- [ ] VM boots to the first-boot TUI without Internet access
- [ ] No pre-created appliance users or default passwords
- [ ] Only TCP **80** and **443** are reachable from the customer network
- [ ] PostgreSQL, MQTT, ingestion/API admin ports, and collector control sockets
      are not published externally
- [ ] Initial administrator can be created in the TUI
- [ ] Second PAM user can sign into both VM and dashboard
- [ ] At least two named sites configure successfully
- [ ] Discovery requires explicit review before enrollment
- [ ] Local telemetry is visible in the dashboard
- [ ] State survives reboot
- [ ] Failed reconfiguration rolls back cleanly

## Security requirements

- **No default password.** Every installation creates its own administrator on
  first boot.
- **Published surface:** only TCP 80/443 on the appliance VM. Port 80 redirects
  to HTTPS once TLS is configured.
- **Internal only:** PostgreSQL, MQTT, service administration endpoints, metrics,
  and collector control sockets stay on private container networks, Unix
  sockets, or loopback.
- **Secrets:** SNMP communities and generated credentials must not appear in
  logs, diagnostics, manifests, or source control.
- **Auth boundary:** dashboard authentication uses the host PAM broker at
  `/run/equate/auth.sock`; the API container never mounts `/etc/shadow`.

## AMD64 client release (phase 2)

After ARM64 Fusion acceptance:

1. Repeat sections 1–7 on an x86 VMware-capable build host with
   `ARCH=amd64`.
2. Export `Equate-Appliance-<version>-amd64.ova` and checksum.
3. Re-import and run the same verification checklist on ESXi or Fusion AMD64.

## Related documentation

- Appliance architecture: [`.ai/project-context/appliance.md`](../../.ai/project-context/appliance.md)
- Production stack: [`deployments/production/appliance/`](../../deployments/production/appliance/)
- Post-install verifier: [`appliance/scripts/verify-appliance.sh`](../../appliance/scripts/verify-appliance.sh)
- Re-import verifier: [`appliance/scripts/verify-ova-import.sh`](../../appliance/scripts/verify-ova-import.sh)
