#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != Linux || "$(uname -m)" != x86_64 ]] ||
   [[ -z "${WSL_DISTRO_NAME:-}" && ! -S "${WSL_INTEROP:-}" ]] ||
   ! command -v wslpath powershell.exe >/dev/null; then
    echo "Run this script in x86_64 WSL with Windows interop enabled." >&2
    exit 1
fi

repo_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
fixture_dir="$repo_dir/test/resource/bin"
mkdir -p -- "$fixture_dir"

install_fixture() (
    local name=$1 url=$2 checksum=$3
    local target="$fixture_dir/$name"
    if [[ -x "$target" ]] &&
       printf '%s  %s\n' "$checksum" "$target" | sha256sum --check --status; then
        printf 'Verified %s\n' "$target"
        return
    fi

    local staged
    staged=$(mktemp "$fixture_dir/.$name.XXXXXXXX")
    trap 'rm -f -- "$staged"' EXIT
    curl --fail --silent --show-error --location --retry 2 --max-time 120 --output "$staged" "$url"
    printf '%s  %s\n' "$checksum" "$staged" | sha256sum --check
    chmod 0755 -- "$staged"
    mv -fT -- "$staged" "$target"
    printf 'Installed %s\n' "$target"
)

install_fixture devcontainer \
    https://github.com/mikoto2000/devcontainers-cli/releases/download/v0.85.0-1/devcontainer-linux-x64-v0.85.0-1 \
    5c2946c8f3e37e92da8e4873438d68bcf09e6b4601a0dc2672f7a88a6980e0c2

# The integration tests look up this Windows binary without an .exe suffix.
install_fixture clipboard-data-receiver \
    https://github.com/mikoto2000/clipboard-data-receiver/releases/download/v3.1.0/clipboard-data-receiver.windows-amd64.exe \
    b5c554c50b25cacc85ab12850559caa269259467c7a9ca7a194008ab2e70aaeb
