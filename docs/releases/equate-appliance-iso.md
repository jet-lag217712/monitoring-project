# Equate Appliance ISO deployment

The appliance is delivered as a fully offline UEFI installer ISO, not an OVA.

1. Create a UEFI virtual machine with its intended appliance disk first on the
   virtual controller. Add any optional data disk only after it.
2. Attach `Equate-Appliance-<version>-amd64.iso` for ESXi/vSphere, or the
   ARM64 ISO only for Apple Silicon Fusion testing.
3. Keep the VM network disconnected if validating offline installation.
4. At the ISO menu, explicitly choose `Install Equate Appliance - Erases First
   Disk`. The default menu selection cancels/reboots instead.
5. When installation completes, eject the ISO and boot from the installed
   disk. The appliance obtains DHCP and brings up the dashboard automatically
   on ports 80/443. The local SNMP setup TUI appears on the VM console.
6. Attach read-only media labeled `EQUATE_TLS` containing `tls.crt` and
   `tls.key`. In the TUI, import the matching certificate/key, enter the
   Workspace domain allow-list and SNMP discovery settings, then explicitly
   accept discovery candidates before they are managed. The certificate has to
   have exactly one concrete `*.equatecloud.tech` DNS SAN. Ensure internal DNS
   and client-CA trust are in place and Equate has registered the matching
   Google JavaScript origin.

The installer never uses an Internet package mirror. It installs the Docker
runtime and appliance dependencies from the ISO, then the first installed boot
verifies the immutable release bundle and loads its OCI images before starting
the Compose stack.
