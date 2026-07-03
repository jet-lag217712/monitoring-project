export function resolveSelectedInterface(device, selectedKey) {
  const interfaces = device?.interfaces ?? []
  if (interfaces.length === 0) return null

  if (!selectedKey) return interfaces[0]

  const match = interfaces.find(
    iface => iface.name === selectedKey || String(iface.if_index) === String(selectedKey),
  )

  return match ?? interfaces[0]
}

export function getInterfaceSelectionKey(iface) {
  return iface?.name ?? String(iface?.if_index ?? '')
}
