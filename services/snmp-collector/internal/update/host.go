package update

import "runtime"

// HostArch returns the appliance architecture label (amd64 or arm64).
func HostArch() string {
	return normalizeArch(runtime.GOARCH)
}
