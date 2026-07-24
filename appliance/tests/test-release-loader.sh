#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
work="$(mktemp -d)"
trap 'chmod -R u+w "${work}" 2>/dev/null || true; rm -rf "${work}"' EXIT

releases="${work}/releases"
data="${work}/data"
release="${releases}/2.0.0"
mkdir -p "${release}/images" "${work}/bin"
printf '2.0.0\n' >"${release}/VERSION"
printf 'image archive\n' >"${release}/images/equate-api.tar"
(cd "${release}" && find . -type f ! -name checksums.txt -print0 | sort -z | xargs -0 sha256sum > checksums.txt)
cat >"${work}/bin/docker" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>"${work}/docker-loads"
EOF
chmod +x "${work}/bin/docker"

PATH="${work}/bin:${PATH}" EQUATE_RELEASES="${releases}" EQUATE_DATA="${data}" "${repo_root}/appliance/scripts/equate-release-load"
[[ "$(readlink "${releases}/current")" == '2.0.0' ]]
[[ -f "${data}/release-loader/2.0.0.loaded" ]]
[[ "$(wc -l <"${work}/docker-loads" | tr -d ' ')" == '1' ]]

# A second boot preserves the verified marker and does not reload the same OCI
# archive, while retaining the immutable release activation.
PATH="${work}/bin:${PATH}" EQUATE_RELEASES="${releases}" EQUATE_DATA="${data}" "${repo_root}/appliance/scripts/equate-release-load"
[[ "$(wc -l <"${work}/docker-loads" | tr -d ' ')" == '1' ]]
