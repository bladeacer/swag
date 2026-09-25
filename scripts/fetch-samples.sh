#!/usr/bin/env sh
# SPDX-License-Identifier: Apache-2.0
#
# fetch-samples.sh downloads the YTSubConverter sample files that the
# end-to-end tests exercise. The files stay out of the repository and are
# ignored by git, so the clean-room statement in
# docs/third-party-notices.md keeps its meaning: the repository carries no
# code, comments, or tables from upstream.
#
# Usage:
#   scripts/fetch-samples.sh
#
# The end-to-end tests skip when the files are absent, so a developer
# without network access still gets a green test run.

set -eu

base="https://raw.githubusercontent.com/arcusmaximus/YTSubConverter/master"
dest="testdata/upstream"

mkdir -p "$dest"
# The YouTube pair. Add further names here when a later milestone needs
# them.
for name in ytt.ytt; do
    echo "fetching $name"
    curl -fsSL "$base/$name" -o "$dest/$name"
done
echo "Downloaded the upstream samples into $dest."
