#!/bin/bash

DOCKER_REPO=$1
OPERATOR_NAME=$2
CATALOG_DIR_PATH=$3
VERSION=$4
GIT_TAG=$5

# Get entries and iterate
CHANNEL_BUNDLES=$(yq eval-all 'select(.schema == "olm.channel") | .entries[].name' "$CATALOG_DIR_PATH"/channels.yaml | grep -v '^---$' | sort | uniq)

# Get latest released version from git tags
LATEST_RELEASE=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
if [ -z "$LATEST_RELEASE" ]; then
    LATEST_RELEASE="0.0.0"
fi

# Clean up files
rm -rf "$CATALOG_DIR_PATH"/bundles.yaml
rm -rf "$CATALOG_DIR_PATH"/release/index.yaml

echo " catalog build start"
SHOULD_RELEASE="false"
for item in $CHANNEL_BUNDLES; do
  # Extract version from entry name (e.g., "hestia-operator.v0.1.0" -> "0.1.0")
  item_version="${item#${OPERATOR_NAME}.v}"

  # Check if entry version is newer than latest release
  is_newer="false"
  newest=$(printf '%s\n%s' "$LATEST_RELEASE" "$item_version" | sort -V | tail -1)
  if [ "$newest" = "$item_version" ] && [ "$item_version" != "$LATEST_RELEASE" ]; then
      SHOULD_RELEASE="true"
      is_newer="true"
  fi

  # Setup bundle from entries
  if [ -n "$GIT_TAG" ] && [ "$is_newer" = "true" ]; then
      bundle="${item//${OPERATOR_NAME}./${OPERATOR_NAME}-bundle:}${GIT_TAG}"
  else
      bundle="${item//${OPERATOR_NAME}./${OPERATOR_NAME}-bundle:}"
  fi

  opm render "$DOCKER_REPO/$bundle" --output=yaml >> "$CATALOG_DIR_PATH"/bundles.yaml
  echo "   >> rendered $bundle >> $CATALOG_DIR_PATH/bundles.yaml"
done

# Build catalog index if there should be a release
  if [ ${SHOULD_RELEASE} = "true" ]; then
      mkdir -p "$CATALOG_DIR_PATH"/release
      yq eval-all '.' "$CATALOG_DIR_PATH"/package.yaml "$CATALOG_DIR_PATH"/channels.yaml "$CATALOG_DIR_PATH"/bundles.yaml > "$CATALOG_DIR_PATH"/release/index.yaml
      echo "  >> created index >> $CATALOG_DIR_PATH/release/index.yaml"
  else
      echo "  >> release is not defined in ${CATALOG_DIR_PATH}/channels.yaml, will not create catalog index"
  fi

#rm -rf "$CATALOG_DIR_PATH"/bundles.yaml
echo " catalog build done!"
