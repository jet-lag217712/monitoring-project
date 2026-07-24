# Equate Appliance

The appliance is the production, single-node deployment described by the
canonical [appliance architecture](../../.ai/project-context/appliance.md).

Build the client ESXi installer ISO with:

```sh
make iso ARCH=amd64 VERSION=2.0.0 EQUATE_GOOGLE_CLIENT_ID=...apps.googleusercontent.com
```

The build creates an immutable release bundle, then remasters a checksum-pinned
official Debian 12.10 DVD with the Equate installer and exports
`dist/Equate-Appliance-<version>-amd64.iso`. The Debian DVD is downloaded once
per architecture and cached under `build/appliance/debian-dvd/`; subsequent ISO
assembly is native `xorriso` work rather than a Simple-CDD package mirror. The
media is UEFI-only and fully offline: it includes Docker, Compose, Open VM
Tools, appliance tools, and all initial OCI images. Docker Buildx builds the
OCI images; `xorriso` performs ISO assembly.

For local Fusion testing on an Apple Silicon Mac, build the matching ARM64
installer with `make iso ARCH=arm64 VERSION=2.0.0 EQUATE_GOOGLE_CLIENT_ID=...apps.googleusercontent.com`; it exports
`dist/Equate-Appliance-<version>-arm64.iso`. The application release is the
same; only the Linux/OCI architecture differs. The ARM64 ISO is for Fusion
testing only. Client deployment uses the AMD64 ISO on a UEFI ESXi/vSphere 7.0
U2 or later VM.

To build both artifacts serially:

```sh
make iso-all VERSION=2.0.0 EQUATE_GOOGLE_CLIENT_ID=...apps.googleusercontent.com
```

Create a blank UEFI VM, attach the applicable ISO, and choose `Install Equate
Appliance - Erases First Disk`. Installation uses only the ISO, leaves any
additional disks untouched, and reboots into the terminal setup console. Eject
the ISO after the installer has completed.

To produce a distributable update package, provide an external Ed25519 private
key only in the release environment:

```sh
EQUATE_SIGNING_KEY=/secure/path/release-ed25519.pem make update-package VERSION=1.0.1
```

This emits `dist/Equate-<version>.eqa`. The matching public key is appliance
configuration. The private key is never included in an OVA, image, or source
tree.

On first boot, Ethernet is configured by DHCP and the UI automatically starts
on ports 80/443. Use the VM console's SNMP setup TUI to import `tls.crt` and
`tls.key` from read-only media labeled `EQUATE_TLS`, then enter the Google
Workspace domain allow-list and SNMP discovery settings. The certificate must
have one concrete `*.equatecloud.tech` DNS SAN; clients require internal DNS
and client-CA trust for that hostname. Equate operations must register the
same exact HTTPS origin in the managed Google client before users sign in.
