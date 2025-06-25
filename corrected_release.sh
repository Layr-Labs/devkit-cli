#!/usr/bin/env bash
set -e

# Parse command line arguments for version support
CONFIG_DIR=""
VERSION_OVERRIDE=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --config-dir)
      CONFIG_DIR="$2"
      shift 2
      ;;
    --version)
      VERSION_OVERRIDE="$2"
      shift 2
      ;;
    *)
      echo "Unknown option $1" >&2
      exit 1
      ;;
  esac
done

# Read operator set mappings from devnet.yaml
echo "Reading operator set mappings from devnet.yaml..." >&2

# Extract aggregator info
aggregator_operator_set_id=$(yq -r '.aggregator.operatorSetId' .hourglass/context/devnet.yaml)
aggregator_digest=$(yq -r '.aggregator.digest' .hourglass/context/devnet.yaml)
aggregator_registry_url=$(yq -r '.aggregator.registry_url' .hourglass/context/devnet.yaml)

# Extract executor info
executor_operator_set_id=$(yq -r '.executor.operatorSetId' .hourglass/context/devnet.yaml)
executor_digest=$(yq -r '.executor.digest' .hourglass/context/devnet.yaml)
executor_registry_url=$(yq -r '.executor.registry_url' .hourglass/context/devnet.yaml)

# Create JSON structs for operator sets
aggregator_json=$(jq -n \
  --arg digest "$aggregator_digest" \
  --arg registry_url "$aggregator_registry_url" \
  '{digest: $digest, registry_url: $registry_url}')

executor_json=$(jq -n \
  --arg digest "$executor_digest" \
  --arg registry_url "$executor_registry_url" \
  '{digest: $digest, registry_url: $registry_url}')

echo "Operator Set Mapping:" >&2
echo "  OperatorSet $aggregator_operator_set_id (aggregator): [$aggregator_json]" >&2
echo "  OperatorSet $executor_operator_set_id (executor): [$executor_json]" >&2

# Function to get cached version
get_cached_version() {
  if [ -n "$CONFIG_DIR" ]; then
    local project_name=$(basename "$(pwd)")
    local version_cache_file="$CONFIG_DIR/versions/${project_name}_version"
    if [ -f "$version_cache_file" ]; then
      cat "$version_cache_file"
    else
      echo ""
    fi
  else
    echo ""
  fi
}

# Build params
buildParams=$(cat ./.hourglass/build.yaml)
registry=$(echo "$buildParams" | yq -r '.container.registry')
image=$(echo "$buildParams" | yq -r '.container.image')
config_tag=$(echo "$buildParams" | yq -r '.container.version')

# Determine final version/tag
if [ -n "$VERSION_OVERRIDE" ]; then
  tag="$VERSION_OVERRIDE"
else
  # Try to use cached version if available
  cached_version=$(get_cached_version)
  if [ -n "$cached_version" ]; then
    tag="$cached_version"
  else
    tag="$config_tag"
  fi
fi

# Construct original and temporary image names
if [ -n "$registry" ] && [ "$registry" != "null" ]; then
  originalImage="${registry}/${image}:${tag}"
  tempImage="${registry}/${image}:${tag}-release"
else
  originalImage="${image}:${tag}"
  tempImage="${image}:${tag}-release"
fi

echo "Rebuilding container for release comparison..." >&2
echo "Original image: $originalImage" >&2
echo "Temporary image: $tempImage" >&2

# Get the original image ID (from the build)
ORIGINAL_IMAGE_ID=$(docker images --format "table {{.ID}}" --no-trunc "$originalImage" | tail -1)

if [ -z "$ORIGINAL_IMAGE_ID" ]; then
  echo "Error: Original image $originalImage not found. Run 'devkit avs build' first." >&2
  exit 1
fi

echo "Original Image ID: $ORIGINAL_IMAGE_ID" >&2

# Build with temporary tag
docker build -t "$tempImage" . >&2

# Get the new image ID
NEW_IMAGE_ID=$(docker images --format "table {{.ID}}" --no-trunc "$tempImage" | tail -1)

echo "New Image ID: $NEW_IMAGE_ID" >&2

# Compare image IDs and set result
if [ "$ORIGINAL_IMAGE_ID" = "$NEW_IMAGE_ID" ]; then
  echo "Image unchanged - no rebuild needed" >&2
  echo "Both images have the same ID: $ORIGINAL_IMAGE_ID" >&2
  IMAGE_CHANGED="false"
else
  echo "Image changed - rebuild detected" >&2
  echo "Original: $ORIGINAL_IMAGE_ID" >&2
  echo "New:      $NEW_IMAGE_ID" >&2
  IMAGE_CHANGED="true"
fi

# Clean up temporary image
docker rmi "$tempImage" >/dev/null 2>&1 || true

# Create the final operator set mapping JSON output
operator_set_mapping_json=$(jq -n \
  --arg agg_id "$aggregator_operator_set_id" \
  --argjson agg_data "[$aggregator_json]" \
  --arg exec_id "$executor_operator_set_id" \
  --argjson exec_data "[$executor_json]" \
  '{
    ($agg_id): $agg_data,
    ($exec_id): $exec_data
  }')

# Export the result for the calling script
echo "IMAGE_CHANGED=$IMAGE_CHANGED" > /tmp/release_result
echo "ORIGINAL_IMAGE_ID=$ORIGINAL_IMAGE_ID" >> /tmp/release_result
echo "NEW_IMAGE_ID=$NEW_IMAGE_ID" >> /tmp/release_result
echo "OPERATOR_SET_MAPPING=$operator_set_mapping_json" >> /tmp/release_result

# Output the operator set mapping as JSON to stdout (ONLY this should go to stdout)
echo "$operator_set_mapping_json"

# Return the boolean result (0 = no change, 1 = changed)
if [ "$IMAGE_CHANGED" = "true" ]; then
  exit 1
else
  exit 0
fi 