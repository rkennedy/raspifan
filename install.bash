#!/bin/bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
readonly script_dir

readonly install_args=(
    --preserve-timestamps
    --target-directory /usr/local/bin
    --verbose
)

set -x

install "${install_args[@]}" "${script_dir}/raspifan"
/usr/local/bin/raspifan install
