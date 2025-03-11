#!/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

###
# Fetches the latest version of the docker files from the SuperMQ repository.
###

set -e
set -o pipefail

REPO_URL=https://github.com/absmach/supermq
TEMP_DIR="smq_temp"
ZIP_FILE="smq.zip"
DOCKER_FOLDER="docker"
DEST_FOLDER="../supermq"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR" || exit 1

if [ -n "$(git status --porcelain)" ]; then
    echo "There are uncommitted changes. Please commit or stash them before running this script."
    exit 1
fi

cleanup() {
    rm -rf "$TEMP_DIR" "$ZIP_FILE"
}
trap cleanup EXIT

curl -s -L "$REPO_URL/archive/refs/heads/main.zip" -o "$ZIP_FILE"
mkdir -p "$TEMP_DIR"
unzip -qq "$ZIP_FILE" -d "$TEMP_DIR"

EXTRACTED_FOLDER=$(find "$TEMP_DIR" -mindepth 1 -maxdepth 1 -type d)

if [ -d "$DEST_FOLDER" ]; then
    rm -r "$DEST_FOLDER"
fi
mkdir -p "$DEST_FOLDER"
mv -f "$EXTRACTED_FOLDER/$DOCKER_FOLDER" "$DEST_FOLDER/$DOCKER_FOLDER"
