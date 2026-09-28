#!/bin/sh
set -eu

. /opt/agm/normalize-base-path.sh

base=$(normalize_base_path "${VITE_BASE_PATH:-}")
wget -qO- "http://127.0.0.1${base}" >/dev/null
