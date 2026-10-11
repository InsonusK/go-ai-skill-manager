#!/bin/sh

set -eu

repository="InsonusK/go-ai-skill-manager"
install_dir="${AISM_INSTALL_DIR:-/usr/local/bin}"

fail() {
	printf 'aism installer: %s\n' "$*" >&2
	exit 1
}

for dependency in curl sha256sum awk mktemp install readlink; do
	command -v "$dependency" >/dev/null 2>&1 || fail "required command not found: $dependency"
done

# Stops the processes started from the installed file (an MCP server a
# client keeps running would stay on the old version), and only those:
# another aism elsewhere on the machine is left alone. Other users'
# processes can't be seen and are skipped.
stop_running() {
	running_file=$(readlink -f "$1" 2>/dev/null) || return 0
	pids=""
	for exe in /proc/[0-9]*/exe; do
		[ "$(readlink "$exe" 2>/dev/null)" = "$running_file" ] || continue
		pid=${exe#/proc/}
		pids="$pids ${pid%/exe}"
	done
	[ -n "$pids" ] || return 0
	printf 'Stopping running aism processes:%s\n' "$pids"
	# shellcheck disable=SC2086
	kill $pids 2>/dev/null || true
	for _ in 1 2 3 4 5; do
		alive=""
		for pid in $pids; do
			if kill -0 "$pid" 2>/dev/null; then alive="$alive $pid"; fi
		done
		[ -n "$alive" ] || break
		sleep 1
	done
	# shellcheck disable=SC2086
	[ -z "$alive" ] || kill -9 $alive 2>/dev/null || true
	printf 'An MCP client that used aism has lost its server: reconnect it (Claude Code: /mcp) or restart the client.\n'
}

[ "$(uname -s)" = "Linux" ] || fail "this installer supports Linux only"
case "$(uname -m)" in
	x86_64 | amd64) ;;
	*) fail "unsupported architecture: $(uname -m) (only amd64 is published)" ;;
esac

latest_url=$(curl -fsSIL -o /dev/null -w '%{url_effective}' \
	"https://github.com/${repository}/releases/latest")
latest_url=${latest_url%/}
tag=${latest_url##*/}
case "$tag" in
	v?*) ;;
	*) fail "could not determine the latest release from: $latest_url" ;;
esac

version=${tag#v}
destination="$install_dir/aism"
if [ -x "$destination" ] && [ "$("$destination" --version 2>/dev/null || true)" = "$version" ]; then
	printf 'ai-skill-manager %s is already installed: %s\n' "$version" "$destination"
	exit 0
fi
asset="ai-skill-manager_${version}_linux_amd64"
checksums="ai-skill-manager_${version}_checksums.txt"
download_base="https://github.com/${repository}/releases/download/${tag}"

temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT
trap 'exit 1' HUP INT TERM

printf 'Downloading ai-skill-manager %s for linux/amd64...\n' "$version"
curl -fsSLo "$temp_dir/$asset" "$download_base/$asset"
curl -fsSLo "$temp_dir/$checksums" "$download_base/$checksums"

expected_hash=$(awk -v name="$asset" '$2 == name || $2 == "*" name { print $1; exit }' \
	"$temp_dir/$checksums")
[ -n "$expected_hash" ] || fail "checksum for $asset is missing"
actual_hash=$(sha256sum "$temp_dir/$asset" | awk '{ print $1 }')
[ "$actual_hash" = "$expected_hash" ] || fail "checksum verification failed for $asset"

stop_running "$destination"

if mkdir -p "$install_dir" 2>/dev/null && [ -w "$install_dir" ]; then
	install -m 0755 "$temp_dir/$asset" "$destination"
else
	command -v sudo >/dev/null 2>&1 || \
		fail "cannot write to $install_dir; set AISM_INSTALL_DIR or install sudo"
	sudo mkdir -p "$install_dir"
	sudo install -m 0755 "$temp_dir/$asset" "$destination"
fi

printf 'Installed %s\n' "$destination"
"$install_dir/aism" --version
