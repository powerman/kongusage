#!/bin/bash
# Print the sorted "<file>:<key>" pairs which usage-lib of <tag> accepts in match arms
# of lib/src/spec/*.rs, i.e. the nodes and properties a spec may use.
# <clone> is a clone of jdx/usage, e.g. by `git clone --filter=blob:none --no-checkout`.
# The key set is an approximation: check a key found here by `usage lint` before relying on it.
set -euo pipefail
[[ $# == 2 ]] || { echo "usage: $0 <clone> <tag>" >&2; exit 2; }
clone=$1 tag=$2
git -C "$clone" ls-tree --name-only "$tag" lib/src/spec/ | grep '\.rs$' |
    while read -r f; do
        # A partial clone fetches the blob by a subprocess which would read the file list.
        git -C "$clone" show "$tag:$f" </dev/null | sed '/#\[cfg(test)\]/,$d' |
            grep -oE '^\s*"[a-z_][a-z0-9_]*"(\s*\|\s*"[a-z_][a-z0-9_]*")*\s*=>' |
            grep -oE '"[a-z_][a-z0-9_]*"' | tr -d '"' | sed "s|^|$(basename "$f" .rs):|" ||
            true # A file without match arms.
    done | sort -u
