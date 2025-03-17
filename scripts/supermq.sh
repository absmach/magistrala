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
DOCKER_FOLDER="docker"
DEST_FOLDER="../../docker/supermq-docker"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR" || exit 1

if [ -n "$(git status --porcelain)" ]; then
    echo "There are uncommitted changes. Please commit or stash them before running this script."
    exit 1
fi

cleanup() {
    rm -rf "$TEMP_DIR" "../$TEMP_DIR"
}
trap cleanup EXIT

git clone --depth 1 --filter=blob:none --sparse "$REPO_URL"
cd "$TEMP_DIR"
git sparse-checkout set "$DOCKER_FOLDER"

if [ -d "$DEST_FOLDER" ]; then
    rm -r "$DEST_FOLDER"
fi
mkdir -p "$DEST_FOLDER"
mv -f "$DOCKER_FOLDER"/{*,.*} "$DEST_FOLDER"
cd ..
rm -rf "$TEMP_DIR"
