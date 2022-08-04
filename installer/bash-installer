#!/usr/bin/env bash
# Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
#
# Symfony CLI installer: this file is part of Symfony CLI project.
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program. If not, see <http://www.gnu.org/licenses/>.
#
set -euo pipefail

CLI_CONFIG_DIR=".symfony5"
CLI_EXECUTABLE="symfony"
CLI_TMP_NAME="$CLI_EXECUTABLE-"$(date +"%s")
CLI_NAME="Symfony CLI"
CLI_DOWNLOAD_URL_PATTERN="https://github.com/symfony-cli/symfony-cli/releases/latest/download/symfony-cli_~platform~.tar.gz"

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

# Check that tar is installed.
if command -v tar >/dev/null 2>&1; then
    output "  [*] Tar is installed" "success"
else
    output "  [ ] ERROR: Tar is required for installation." "error"
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
        if [ "darwin" = "${kernel}" ]; then
            output "  [ ] Your architecture (${machine}) is not supported anymore" "error"
            exit 1
        fi
        ;;
    x86_64)
        machine="amd64"
        ;;
    *)
        output "  [ ] Your architecture (${machine}) is not currently supported" "error"
        exit 1
        ;;
esac

output "  [*] Your architecture (${machine}) is supported" "success"

if [ "darwin" = "${kernel}" ]; then
    machine="all"
fi

platform="${kernel}_${machine}"

# The necessary checks have passed. Start downloading the right version.
output "\nDownload" "heading"

latest_url=${CLI_DOWNLOAD_URL_PATTERN/~platform~/${platform}}
output "  Downloading ${latest_url}...";
case $downloader in
    "curl")
        curl --fail --location "${latest_url}" > "/tmp/${CLI_TMP_NAME}.tar.gz"
        ;;
    "wget")
        wget -q --show-progress "${latest_url}" -O "/tmp/${CLI_TMP_NAME}.tar.gz"
        ;;
esac

# shellcheck disable=SC2181
if [ $? != 0 ]; then
    output "  The download failed." "error"
    exit 1
fi

output "  Uncompress binary..."
tar -xz --directory "/tmp" -f "/tmp/${CLI_TMP_NAME}.tar.gz"
rm "/tmp/${CLI_TMP_NAME}.tar.gz"

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

if mv "/tmp/${CLI_EXECUTABLE}" "${binary_dest}/${CLI_EXECUTABLE}"; then
    output "  The binary was saved to: ${binary_dest}/${CLI_EXECUTABLE}"
else
    output "  Failed to move the binary to ${binary_dest}." "error"
    rm "/tmp/${CLI_EXECUTABLE}"
    exit 1
fi

#output "  Installing the shell auto-completion..."
#"${binary_dest}/${CLI_EXECUTABLE}" self:shell-setup --silent
#if [ $? != 0 ]; then
#    output "  Failed to install the shell auto-completion." "warning"
#fi

output "\nThe ${CLI_NAME} was installed successfully!" "success"

if [ "${custom_dir}" == "false" ]; then
    output "\nUse it as a local file:" "info"
    output "  ${binary_dest}/${CLI_EXECUTABLE}"
    output "\nOr add the following line to your shell configuration file:" "info"
    output "  export PATH=\"\$HOME/${CLI_CONFIG_DIR}/bin:\$PATH\""
    output "\nOr install it globally on your system:" "info"
    output "  mv ${binary_dest}/${CLI_EXECUTABLE} /usr/local/bin/${CLI_EXECUTABLE}"
    output "\nThen start a new shell and run '${CLI_EXECUTABLE}'" "info"
fi
