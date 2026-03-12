#!/usr/bin/env bash
# Comment on PRs from a GitHub PR list URL.
# Requires: gh CLI, logged in (gh auth login).
#
# Usage:
#   pr-comment.sh [--filterout-failed] <github_pr_list_url> <comment>
#
# Example:
#   pr-comment.sh "https://github.com/openshift-kni/cnf-features-deploy/pulls?q=is%3Apr+is%3Aopen" "/lgtm"
#   pr-comment.sh --filterout-failed "https://github.com/owner/repo/pulls?q=is%3Aopen" "/lgtm
# /approve"

set -e

FILTEROUT_FAILED=false
URL=""
COMMENT=""

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --filterout-failed)
      FILTEROUT_FAILED=true
      shift
      ;;
    *)
      if [[ -z "$URL" ]]; then
        URL="$1"
      elif [[ -z "$COMMENT" ]]; then
        COMMENT="$1"
      else
        echo "Unexpected argument: $1" >&2
        exit 1
      fi
      shift
      ;;
  esac
done

if [[ -z "$URL" || -z "$COMMENT" ]]; then
  echo "Usage: $0 [--filterout-failed] <github_pr_list_url> <comment>" >&2
  echo "Example: $0 --filterout-failed 'https://github.com/owner/repo/pulls?q=is%3Aopen' '/lgtm'" >&2
  exit 1
fi

# Parse repo from URL (e.g. https://github.com/owner/repo/pulls?q=... -> owner/repo)
if [[ "$URL" =~ https?://github\.com/([^/]+)/([^/?]+) ]]; then
  REPO="${BASH_REMATCH[1]}/${BASH_REMATCH[2]}"
else
  echo "Error: could not parse owner/repo from URL: $URL" >&2
  exit 1
fi

# Parse search query from ?q= (URL-decode)
SEARCH=""
if [[ "$URL" =~ q=([^&\"]*) ]]; then
  QUERY_RAW="${BASH_REMATCH[1]}"
  SEARCH=$(python3 -c "import urllib.parse, sys; print(urllib.parse.unquote_plus(sys.argv[1]))" "$QUERY_RAW")
fi
# Default to open PRs if no query
if [[ -z "$SEARCH" ]]; then
  SEARCH="is:open"
fi

# gh search can choke on parentheses in title words (e.g. "chore(deps):"); normalize to words
SEARCH=$(echo "$SEARCH" | sed 's/([^)]*)//g' | sed 's/  */ /g' | sed 's/^ *//;s/ *$//')

echo "Repo:  $REPO"
echo "Query: $SEARCH"
echo "Filter out failed CI: $FILTEROUT_FAILED"
echo "Comment: $COMMENT"
echo "---"

# List PR numbers
PR_JSON=$(gh pr list --repo "$REPO" --search "$SEARCH" --json number --limit 500)
PR_NUMBERS=($(python3 -c "import json,sys; d=json.load(sys.stdin); print(' '.join(str(p['number']) for p in d))" <<< "$PR_JSON"))

if [[ ${#PR_NUMBERS[@]} -eq 0 ]]; then
  echo "No PRs found."
  exit 0
fi

echo "Found ${#PR_NUMBERS[@]} PR(s)."

if [[ "$FILTEROUT_FAILED" == true ]]; then
  FILTERED=()
  for pr in "${PR_NUMBERS[@]}"; do
    if gh pr checks "$pr" --repo "$REPO" 2>/dev/null | grep -q "fail"; then
      echo "  #$pr - skipped (failing CI)"
    else
      FILTERED+=("$pr")
    fi
  done
  PR_NUMBERS=("${FILTERED[@]}")
  echo "After filtering: ${#PR_NUMBERS[@]} PR(s)."
fi

for pr in "${PR_NUMBERS[@]}"; do
  echo -n "Commenting on PR #$pr... "
  if gh pr comment "$pr" --repo "$REPO" --body "$COMMENT"; then
    echo "OK"
  else
    echo "Failed"
  fi
done

echo "Done."
