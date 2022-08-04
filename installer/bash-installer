#!/usr/bin/env bash
# Symfony CLI installer.
set -euo pipefail

CLI_LATEST_VERSION_URL="https://get.symfony.com/cli/LATEST"
CLI_CONFIG_DIR=".symfony"
CLI_EXECUTABLE="symfony"
CLI_TMP_NAME="$CLI_EXECUTABLE-"$(date +"%s")
CLI_NAME="Symfony CLI"
CLI_DOWNLOAD_URL_PATTERN="https://github.com/symfony/cli/releases/download/v~latest_version~/symfony_~platform~.gz"

function output {
    style_start=""
    style_end=""
    if [ "${2:-}" != "" ]; then
    case $2 in
        "success")
            style_start="\033[0;32m"
            style_end="\033[0m"
            ;;
        "error")
            style_start="\033[31;31m"
            style_end="\033[0m"
            ;;
        "info"|"warning")
            style_start="\033[33m"
            style_end="\033[39m"
            ;;
        "heading")
            style_start="\033[1;33m"
            style_end="\033[22;39m"
            ;;
    esac
    fi

    builtin echo -e "${style_start}${1}${style_end}"
}

output "${CLI_NAME} installer" "heading"

binary_dest="${HOME}/${CLI_CONFIG_DIR}/bin"
custom_dir="false"

# Getops does not support long option names
while [[ $# -gt 0 ]]; do
case $1 in
    --install-dir=*)
        binary_dest="${1#*=}"
        custom_dir="true"
        shift # past argument=value
        ;;
    --install-dir)
        binary_dest="${2:-}"
        custom_dir="true"
        shift # past argument
        shift # past value
        ;;
    --channel=*)
        channel="${1#*=}"
        shift # past argument=value
        CLI_DOWNLOAD_URL_PATTERN="${channel}/v~latest_version~/symfony_~platform~"
        CLI_LATEST_VERSION_URL="${channel}/LATEST"
        ;;
    *)
        output "Unknown option $1" "error"
        output "Usage: ${0} [--install-dir=dir]"
        exit 1
        ;;
esac
done

# Run environment checks.
output "\nEnvironment check" "heading"

# Check that cURL or wget is installed.
downloader=""
if command -v curl >/dev/null 2>&1; then
    downloader="curl"
    output "  [*] cURL is installed" "success"
elif command -v wget >/dev/null 2>&1; then
    downloader="wget"
    output "  [*] wget is installed" "success"
else
    output "  [ ] ERROR: cURL or wget is required for installation." "error"
    exit 1
fi

# Check that gzip is installed.
if command -v gzip >/dev/null 2>&1; then
    output "  [*] Gzip is installed" "success"
else
    output "  [ ] ERROR: Gzip is required for installation." "error"
    exit 1
fi

# Check that Git is installed.
if command -v git >/dev/null 2>&1; then
    output "  [*] Git is installed" "success"
else
    output "  [ ] Warning: Git will be needed." "warning"
fi

kernel=$(uname -s 2>/dev/null || /usr/bin/uname -s)
case ${kernel} in
    "Linux"|"linux")
        kernel="linux"
        ;;
    "Darwin"|"darwin")
        kernel="darwin"
        ;;
    *)
        output "OS '${kernel}' not supported" "error"
        exit 1
        ;;
esac

machine=$(uname -m 2>/dev/null || /usr/bin/uname -m)
case ${machine} in
    arm|armv7*)
        machine="arm"
        ;;
    aarch64*|armv8*|arm64)
        machine="arm64"
        ;;
    i[36]86)
        machine="386"
        ;;
    x86_64)
        machine="amd64"
        ;;
    *)
        output "  [ ] Your architecture (${machine}) is not currently supported" "error"
        exit 1
        ;;
esac

platform="${kernel}_${machine}"

if [ "darwin_386" = "${platform}" ]; then
    output "  [ ] Your architecture (${machine}) is not supported anymore" "error"
    exit 1
fi

if [ "darwin_arm64" = "${platform}" ]; then
    platform="darwin_amd64"
fi

output "  [*] Your architecture (${machine}) is supported" "success"

# The necessary checks have passed. Start downloading the right version.
output "\nDownload" "heading"

output "  Finding the latest version (platform: \"${platform}\")...";

case ${downloader} in
    "curl")
        latest_version=$(curl --fail "${CLI_LATEST_VERSION_URL}" -s)
        ;;
    "wget")
        latest_version=$(wget -q "${CLI_LATEST_VERSION_URL}" -O - 2>/dev/null)
        ;;
esac
# shellcheck disable=SC2181
if [ $? != 0 ]; then
    output "  Failed to download LATEST version file: ${CLI_LATEST_VERSION_URL}" "error"
    exit 1
fi

latest_url=${CLI_DOWNLOAD_URL_PATTERN/~latest_version~/${latest_version}}
latest_url=${latest_url/~platform~/${platform}}
output "  Downloading version ${latest_version} (${latest_url})...";
case $downloader in
    "curl")
        curl --fail --location "${latest_url}" > "/tmp/${CLI_TMP_NAME}.gz"
        ;;
    "wget")
        wget -q --show-progress "${latest_url}" -O "/tmp/${CLI_TMP_NAME}.gz"
        ;;
esac

# shellcheck disable=SC2181
if [ $? != 0 ]; then
    output "  The download failed." "error"
    exit 1
fi

output "  Uncompress binary..."
gzip -d "/tmp/${CLI_TMP_NAME}.gz"

output "  Making the binary executable..."
chmod 755 "/tmp/${CLI_TMP_NAME}"

if [ ! -d "${binary_dest}" ]; then
    if ! mkdir -p "${binary_dest}"; then
        binary_dest="."
    fi
fi

if [ "${custom_dir}" == "true" ]; then
    output "  Installing the binary into ${binary_dest} ..."
else
    output "  Installing the binary into your home directory..."
fi

if mv "/tmp/${CLI_TMP_NAME}" "${binary_dest}/${CLI_EXECUTABLE}"; then
    output "  The binary was saved to: ${binary_dest}/${CLI_EXECUTABLE}"
else
    output "  Failed to move the binary to ${binary_dest}." "error"
    rm "/tmp/${CLI_TMP_NAME}"
    exit 1
fi

#output "  Installing the shell auto-completion..."
#"${binary_dest}/${CLI_EXECUTABLE}" self:shell-setup --silent
#if [ $? != 0 ]; then
#    output "  Failed to install the shell auto-completion." "warning"
#fi

output "\nThe ${CLI_NAME} v${latest_version} was installed successfully!" "success"

if [ "${custom_dir}" == "false" ]; then
    output "\nUse it as a local file:" "info"
    output "  ${binary_dest}/${CLI_EXECUTABLE}"
    output "\nOr add the following line to your shell configuration file:" "info"
    output "  export PATH=\"\$HOME/${CLI_CONFIG_DIR}/bin:\$PATH\""
    output "\nOr install it globally on your system:" "info"
    output "  mv ${binary_dest}/${CLI_EXECUTABLE} /usr/local/bin/${CLI_EXECUTABLE}"
    output "\nThen start a new shell and run '${CLI_EXECUTABLE}'" "info"
fi
