#!/bin/bash

SKIP_DIRS=(
  "$BUILD_DIR"
  "$CACHE_DIR"
  "$SOURCE_DIR/.git"
  "$SOURCE_DIR/vendor"
  "$SOURCE_DIR/.gopath"
)

# build a little array of `find` args to filter out all the SKIP_DIRS
FIND_SKIP_DIRS=(-type d \( -false)
for d in "${SKIP_DIRS[@]}"; do
  FIND_SKIP_DIRS+=("-o" "-path" "$d")
done
FIND_SKIP_DIRS+=(\) -prune -o)

# all the top-level directories we have that old Go source code
GO_SOURCE_DIRS=($SOURCE_DIR/cmd $SOURCE_DIR/providers)

# all the directories that contain Go source code (including subdirectories)
GO_SOURCE_SUBDIRS=()
while IFS= read -r -d $'\0'; do
  GO_SOURCE_SUBDIRS+=("$REPLY")
done < <(find "${GO_SOURCE_DIRS[@]}" -type d -print0)

# all the Go source files
GO_SOURCE_FILES=()
while IFS= read -r -d $'\0'; do
  GO_SOURCE_FILES+=("$REPLY")
done < <(find "${GO_SOURCE_DIRS[@]}" "${FIND_SKIP_DIRS[@]}" -type f -iname "*.go" -print0)

# all JSON files
JSON_FILES=()
while IFS= read -r -d $'\0'; do
  JSON_FILES+=("$REPLY")
done < <(find "$SOURCE_DIR" "${FIND_SKIP_DIRS[@]}" -type f -iname "*.json" -print0)

# all YAML files
YAML_FILES=()
while IFS= read -r -d $'\0'; do
  YAML_FILES+=("$REPLY")
done < <(find "$SOURCE_DIR" "${FIND_SKIP_DIRS[@]}" -type f -iname "*.yaml" -print0)

# all shell scripts
SHELL_FILES=()
while IFS= read -r -d $'\0'; do
  SHELL_FILES+=("$REPLY")
done < <(find "$SOURCE_DIR" "${FIND_SKIP_DIRS[@]}" -type f -iname "*.sh" -print0)

# all Terraform module directories
# shellcheck disable=SC2034
TERRAFORM_MODULES=(
  $SOURCE_DIR/zone
  $SOURCE_DIR/zone/worker_pool
  $SOURCE_DIR/zone/border
  $SOURCE_DIR/zone/director
)

# all source files of all types
SOURCE_FILES=()
while IFS= read -r -d $'\0'; do
  SOURCE_FILES+=("$REPLY")
done < <(find "$SOURCE_DIR" "${FIND_SKIP_DIRS[@]}" -type f -print0)
