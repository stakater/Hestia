#!/bin/bash

# Determines the next release version by comparing the version defined in
# channels.yaml against the latest git tag. If the channel entry version is
# newer, it is used. Otherwise, falls back to the auto-bumped patch version.

OPERATOR_NAME=${1:-hestia-operator}
CATALOG_DIR_PATH=${2:-catalog}

# Get latest released version from git tags
LATEST_RELEASE=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
if [ -z "$LATEST_RELEASE" ]; then
    LATEST_RELEASE="0.0.0"
fi

# Get the highest version from channel entries
CHANNEL_VERSION=$(yq eval-all 'select(.schema == "olm.channel") | .entries[].name' "$CATALOG_DIR_PATH"/channels.yaml \
    | grep -v '^---$' \
    | sed "s/${OPERATOR_NAME}\.v//" \
    | sort -V \
    | tail -1)

if [ -z "$CHANNEL_VERSION" ]; then
    echo "error: no version found in $CATALOG_DIR_PATH/channels.yaml" >&2
    exit 1
fi

# If the channel version is newer than the latest release, use it
newest=$(printf '%s\n%s' "$LATEST_RELEASE" "$CHANNEL_VERSION" | sort -V | tail -1)
if [ "$newest" = "$CHANNEL_VERSION" ] && [ "$CHANNEL_VERSION" != "$LATEST_RELEASE" ]; then
    echo "$CHANNEL_VERSION"
else
    # Auto-bump patch version from latest release
    IFS='.' read -r major minor patch <<< "$LATEST_RELEASE"
    patch=$((patch + 1))
    echo "${major}.${minor}.${patch}"
fi
