#!/usr/bin/env bash
#MISE description="Create PR in a package-manager repository"

set -o errexit
set -o nounset
set -o pipefail

workspace="$MISE_PROJECT_ROOT"

repo="$1"
file="$2"

basename="${file##*/}" # basename
name="${basename%.*}"  # remove extension

# clone repo
repo_dir="$(mktemp -d)"
gh repo clone "$repo" "$repo_dir" -- --depth=1
cd "$repo_dir" || exit 1

vars="$workspace/template.vars"

# generate file from template
# shellcheck disable=SC2046
env $(xargs -0 <"$vars") \
	envsubst "$(sed 's/^/$/' "$vars")" \
	<"$workspace/.github/workflows/templates/${basename}" \
	>"${file}"

version=$(grep -m 1 -oP '^VERSION=.+$' "$vars" | cut -d '=' -f 2)

# create pr
branch="${name//-/_}_${version}"
git config user.name "NAIS team app"
git config user.email "devnull@nais.io"
gh auth setup-git
git switch -c "$branch"
git commit -am "$name $version"
git push --set-upstream origin "$branch"
until gh pr create --fill --base main --head "$branch"; do echo "Retrying gh pr create..."; sleep 1; done
until gh pr merge --auto --rebase; do echo "Retrying gh pr merge..."; sleep 1; done
