#!/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

###
# Fetches the latest version of the docker files from the SuperMQ repository.
###

set -e
set -o pipefail

REPO_URL=https://github.com/absmach/supermq
TEMP_DIR="supermq"
DOCKER_DIR="docker"
DOCKER_DST_DIR="../docker"
DEST_DIR="../../docker/supermq-docker"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR" || exit 1

if [ -n "$(git status --porcelain "$DOCKER_DST_DIR")" ]; then
    echo "There are uncommitted changes in '$DOCKER_DIR' dir. Please commit or stash them before running this script."
    exit 1
fi

cleanup() {
    rm -rf "$TEMP_DIR"
}
cleanup
trap cleanup EXIT

git clone --depth 1 --filter=blob:none --sparse "$REPO_URL"
cd "$TEMP_DIR"
git sparse-checkout set "$DOCKER_DIR"

if [ -d "$DEST_DIR" ]; then
    rm -r "$DEST_DIR"
fi
mkdir -p "$DEST_DIR"
mv -f "$DOCKER_DIR"/.??* "$DOCKER_DIR"/* "$DEST_DIR"/
cd ..
rm -rf "$TEMP_DIR"
