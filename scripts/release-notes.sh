#!/usr/bin/env bash

set -euo pipefail

tag=${1:?Usage: release-notes.sh TAG OUTPUT [BASE_REF]}
output=${2:?Usage: release-notes.sh TAG OUTPUT [BASE_REF]}
base_ref=${3:-}

if [[ ! $tag =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  echo "Invalid release tag: $tag (expected vMAJOR.MINOR.PATCH)" >&2
  exit 1
fi

version=${tag#v}
major=${BASH_REMATCH[1]}
minor=${BASH_REMATCH[2]}

git rev-parse --verify --quiet "${tag}^{commit}" >/dev/null || {
  echo "Release tag does not resolve to a commit: $tag" >&2
  exit 1
}

if [[ -z $base_ref ]]; then
  base_ref=$(git describe --tags --abbrev=0 "${tag}^" 2>/dev/null || true)
fi

if [[ -n $base_ref ]]; then
  git rev-parse --verify --quiet "${base_ref}^{commit}" >/dev/null || {
    echo "Base ref does not resolve to a commit: $base_ref" >&2
    exit 1
  }
  range="${base_ref}..${tag}"
else
  range=$tag
fi

repository=${GITHUB_REPOSITORY:-}
if [[ -z $repository ]]; then
  repository=$(git config --get remote.origin.url)
  repository=${repository%.git}
  repository=${repository#git@github.com:}
  repository=${repository#https://github.com/}
fi
server_url=${GITHUB_SERVER_URL:-https://github.com}
repository_url="${server_url}/${repository}"

changelog=$(awk -v version="$version" '
  index($0, "## [" version "]") == 1 { found = 1; next }
  found && /^## \[/ { exit }
  found { print }
  END { if (!found) exit 2 }
' CHANGELOG.md) || {
  echo "CHANGELOG.md has no section for ${version}" >&2
  exit 1
}

contributors=$(mktemp)
trap 'rm -f "$contributors"' EXIT
{
  git log --format='%aN%x09%aE' "$range"
  git log --format='%b' "$range" |
    sed -nE 's/^[Cc]o-authored-by:[[:space:]]*(.*)[[:space:]]+<([^>]+)>$/\1\t\2/p'
} | sort -fu >"$contributors"

{
  printf 'Watchtower **%s** is available as a multi-architecture container image and as downloadable binaries.\n\n' "$version"
  printf '## Container image\n\n'
  printf '%s\n' '```bash'
  printf 'docker pull patbaumgartner/watchtower:%s\n' "$version"
  printf '%s\n\n' '```'
  printf "Also published as \`latest\`, \`%s.%s\`, and \`%s\` for \`linux/amd64\`, \`linux/arm64\`, \`linux/arm/v7\`, and \`linux/386\`.\n\n" "$major" "$minor" "$major"
  printf '%s\n\n' "$changelog"
  printf '## Commits\n\n'
  git log --reverse --no-merges --format="- [\`%h\`](${repository_url}/commit/%H) %s" "$range"
  printf '\n\n## Contributors\n\n'
  while IFS=$'\t' read -r name email; do
    if [[ $email =~ ^[0-9]+\+([^@]+)@users\.noreply\.github\.com$ ]]; then
      printf -- '- [%s](https://github.com/%s)\n' "$name" "${BASH_REMATCH[1]}"
    elif [[ $email =~ ^([^@]+)@users\.noreply\.github\.com$ ]]; then
      printf -- '- [%s](https://github.com/%s)\n' "$name" "${BASH_REMATCH[1]}"
    else
      printf -- '- %s\n' "$name"
    fi
  done <"$contributors"
  printf '\n## Verification\n\n'
  printf "Release archives include \`checksums.txt\` and GitHub build provenance attestations. Container images include SBOM and provenance metadata.\n\n"
  if [[ -n $base_ref ]]; then
    printf '**Full Changelog**: %s/compare/%s...%s\n' "$repository_url" "$base_ref" "$tag"
  else
    printf '**Full Changelog**: %s/commits/%s\n' "$repository_url" "$tag"
  fi
} >"$output"