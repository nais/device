#!/usr/bin/env bash
#MISE description="Create PR in a package-manager repository"

set -o errexit
set -o nounset
set -o pipefail

# shellcheck disable=SC2153
version="$VERSION"
workspace="$MISE_PROJECT_ROOT"

repo="$1"
file="$2"

basename="${file##*/}" # basename
name="${basename%.*}"  # remove extension

git config set user.name="NAIS team app"
git config set user.email="devnull@nais.io"

# clone repo
repo_dir="$(mktemp -d)"
gh repo clone "$repo" "$repo_dir" -- --depth=1
echo "repo cloned"
cd "$repo_dir" || exit 1

# generate file from template
# shellcheck disable=SC2046
env $(xargs -0 <"$workspace/template.vars") \
	envsubst "$(sed 's/^/$/' "$workspace/template.vars")" \
	<"$workspace/.github/workflows/templates/${basename}" \
	>"${file}"

# create pr
git switch -c "${name//-/_}_${version}"
git commit -am "$name $version"
gh pr create --fill
echo "pr created"
