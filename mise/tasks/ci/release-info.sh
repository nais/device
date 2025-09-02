#!/usr/bin/env bash
#MISE description="Generate release information using git-cliff"

main() {
	local repository token changelog version
	repository="$1"
	token="$2"

	changelog="$(git-cliff \
		--bump \
		--github-repo "$repository" \
		--github-token "$token" \
		--unreleased \
		--verbose \
		--strip all)"
	output "changelog" "$changelog"

	version="$(grep -m 1 -oP '(?<=^##\s)v\d+\.\d+\.\d+(?=\s-\s)' <<<"$changelog")"
	if [[ -z "$version" ]]; then
		echo "Unable to read version from changelog, abort"
		return
	fi
	output "version" "$version"

	{
		if [[ -n "$changelog" ]]; then
			cat <<-EOF
				# :pencil: Changelog preview
				Below is a preview of the Changelog that will be added to the next release. Only commit messages that follow the [Conventional Commits specification](https://www.conventionalcommits.org/) will be included in the Changelog.

				$changelog
			EOF
		else
			cat <<-EOF
				# :disappointed: No release for you
				There are no commits in your branch that follow the [Conventional Commits specification](https://www.conventionalcommits.org/), so no release will be created.

				If you want to create a release from this pull request, please reword your commit messages to replace this message with a preview of a beautiful Changelog."
			EOF
		fi
	} | if [[ "$GITHUB_EVENT_NAME" == "pull_request" ]]; then
		gh pr comment "${3:-${GITHUB_REF_NAME%%/merge}}" \
			--edit-last --create-if-none \
			--repo "$repository" \
			--body-file=-
		# else
		# TODO: Decide if we want this outside of PRs.
		# echo
		# echo "This would have been posted if you ran this on GitHub:"
		# cat
	fi
	# common functions for CI tasks
}

output() {
	local key value
	key="$1"
	value="$2"

	if [[ -z "$GITHUB_OUTPUT" ]]; then
		echo "output: $key => $value"
		return
	fi

	if [[ -z "$key" ]]; then
		echo "output: missing key " >&2
	elif [[ -z "$value" ]]; then
		echo "output: missing value for key '$key'" >&2
	else
		{
			echo "$key<<EOF"
			echo "$value"
			echo "EOF"
		} >>"$GITHUB_OUTPUT"
	fi
}

repository="${1:-$GITHUB_REPOSITORY}"
token="${2:-$GITHUB_TOKEN}"
if [[ -z "$repository" || -z "$token" ]]; then
	echo "Usage: $0 <repository> <token>"
	exit 1
fi

main "$repository" "$token"
