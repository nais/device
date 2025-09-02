#!/usr/bin/env bash
# Add to git hooks as a commit-msg hook to enforce semantic commit messages.
# ln -s ../../script/semantic-commit-hook.sh .git/hooks/commit-msg

function validate_title {
  grep -qE '^(Merge|((feat|fix|ci|docs|refactor|perf|test|build|style)(\([a-z0-9\s\-\_\,]+\))?!?:\s\w))' <<<"$1"
}

function explain_scheme {
  echo "Your commit message title did not follow semantic versioning:"
  echo ""
  echo "  $1"
  echo ""
  echo "Format:   <type>[(<scope>)]: <subject>"
  echo ""
  echo "Type     | Description"
  echo "---------+--------------------------------------------------------------------------"
  echo "feat     | Introduces a new feature"
  echo "fix      | Patches a bug"
  echo "ci       | CI configuration files and scripts (i.e. .github/**, some mise tasks)"
  echo "docs     | Documentation only changes (i.e. README, code comments)"
  echo "refactor | Neither bugfix nor adds a feature (i.e. rename package, move code)"
  echo "perf     | Improves performance (i.e. removes a time.Sleep)"
  echo "test     | Adding / correcting tests"
  echo "build    | Build system or external dependencies (i.e. go.mod, mise tasks)"
  echo "style    | Changes to output formatting / colors (i.e. changing wording in an error)"
  echo ""
  echo "Examples:"
  echo "- feat(api): add endpoint"
  echo "- build(deps): bump dependencies"
  echo "- docs: fix typo in README"
  echo ""
  echo "Please see"
  echo "- https://www.conventionalcommits.org/en/v1.0.0/#summary"
}

if [ -z "$1" ]; then
  echo "Missing argument (commit message). Did you try to run this script manually?"
  exit 1
fi

commit_title="$(head "$1" -n1)"
if ! validate_title "$commit_title"; then
  explain_scheme "$commit_title"
  exit 1
fi