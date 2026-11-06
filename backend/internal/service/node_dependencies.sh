#!/bin/sh
# Run before Python, including on minimal images that do not provide sudo.
set -eu

fail() { printf 'error:%s\n' "$1"; exit 1; }
privilege=root
if [ "$(id -u)" != 0 ]; then
    command -v sudo >/dev/null 2>&1 && sudo -n true </dev/null >/dev/null 2>&1 || fail privilege
    privilege=sudo
fi
as_root() {
    if [ "$privilege" = root ]; then "$@"; else sudo -n "$@"; fi
}
python_ready() {
    as_root python3 -c 'import ssl,sys; sys.exit(sys.version_info < (3,8))' </dev/null >/dev/null 2>&1
}
certificates_ready() {
    as_root python3 -c 'import ssl; assert ssl.create_default_context().get_ca_certs()' </dev/null >/dev/null 2>&1
}
curl_ready() { as_root sh -c 'command -v curl >/dev/null' </dev/null >/dev/null 2>&1; }

packages=
python_ready || packages="$packages python3"
certificates_ready || packages="$packages ca-certificates"
if [ "${1:-}" = curl ]; then curl_ready || packages="$packages curl"; fi
if [ -n "$packages" ]; then
    command -v timeout >/dev/null 2>&1 || fail timeout_unavailable
    log=$(mktemp) || fail temporary_file
    trap 'rm -f "$log"' EXIT HUP INT TERM
    status=0
    # Explicit package names only; no full upgrade or proxy-service installation.
    # List services needing restart instead of allowing needrestart to restart them.
    if command -v apt-get >/dev/null 2>&1; then
        as_root env LC_ALL=C DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=l timeout -k 5 90 sh -c '
            apt-get -o Acquire::Retries=0 -o Acquire::http::Timeout=15 -o Acquire::https::Timeout=15 -o APT::Update::Error-Mode=any -o DPkg::Lock::Timeout=10 update &&
            apt-get -o DPkg::Lock::Timeout=10 -y --no-upgrade --no-install-recommends install "$@"
        ' sh $packages </dev/null >"$log" 2>&1 || status=$?
    elif command -v dnf >/dev/null 2>&1; then
        as_root env LC_ALL=C timeout -k 5 90 dnf -y --setopt=install_weak_deps=False install $packages </dev/null >"$log" 2>&1 || status=$?
    elif command -v yum >/dev/null 2>&1; then
        as_root env LC_ALL=C timeout -k 5 90 yum -y install $packages </dev/null >"$log" 2>&1 || status=$?
    else
        fail package_manager
    fi
    if [ "$status" != 0 ]; then
        case "$status" in 124|137) fail install_timeout ;; esac
        if grep -Eqi 'Could not get lock|Unable to acquire.*lock|another app.*lock|another process.*lock' "$log"; then fail package_lock; fi
        if grep -Eqi 'No space left|not enough free space|insufficient disk space' "$log"; then fail disk_space; fi
        if grep -Eqi 'Could not resolve|Temporary failure resolving|Failed to fetch|Failed to download|Cannot download|Connection.*(failed|timed out)|certificate verification|does not have a Release file' "$log"; then fail package_network; fi
        fail install_failed
    fi
fi
python_ready || fail python_version
certificates_ready || fail certificates
if [ "${1:-}" = curl ]; then curl_ready || fail curl; fi
printf '%s\n' "$privilege"
