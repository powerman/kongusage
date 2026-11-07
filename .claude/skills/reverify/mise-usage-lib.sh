#!/bin/bash
# Print "<tag> <date> <usage-lib version>" for every mise release published since <YYYY-MM-DD>,
# newest first; the usage-lib version is read from Cargo.lock of the tag.
set -euo pipefail
[[ $# == 1 ]] || { echo "usage: $0 <YYYY-MM-DD>" >&2; exit 2; }
since=$1
gh api --paginate 'repos/jdx/mise/releases?per_page=100' -q "
    .[] | select(.draft or .prerelease | not) | select(.published_at >= \"$since\")
    | .tag_name + \" \" + .published_at[:10]" |
    while read -r tag date; do
        # awk reads all input: an early exit would kill gh by SIGPIPE, failing the script.
        lib=$(gh api -H 'Accept: application/vnd.github.raw' \
            "repos/jdx/mise/contents/Cargo.lock?ref=$tag" </dev/null |
            awk '!found && /^name = "usage-lib"$/ {
                getline; gsub(/"/, "", $3); print $3; found = 1
            }')
        echo "$tag $date $lib"
    done
