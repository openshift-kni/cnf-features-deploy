#!/bin/bash

set -e -o pipefail

function finish {
	if [ -f "$commit_msg_filename" ]; then
		rm -f $commit_msg_filename
	fi
}
trap finish EXIT

function is_merge_commit {
	commitid=$1
	local sha="$1"
	msha=$(git rev-list -1 --merges ${sha}~1..${sha})
	[ -z "$msha" ] && return 1
	return 0
}

if [[ -z "$UPSTREAM_COMMIT" ]]; then
	# CI=true is set by prow as a way to detect we are running uder the ci
	if [[ ! -z "$CI" ]]; then

              #  Under Prow, apply commit verification only for presubmit jobs
              if [[ "$JOB_TYPE" != "presubmit" ]]; then
                  echo "Not a Prow presubmit job. SKIPPING commit verification!"
                  exit 0
              fi

               cp -a /go/src/github.com/openshift-kni/cnf-features-deploy /tmp/
               cd /tmp/cnf-features-deploy

               # Prow PULL_BASE_REF to determine the branch: https://docs.prow.k8s.io/docs/jobs/#job-environment-variables
               echo "name of the base branch: $PULL_BASE_REF"
	       latest_upstream_commit=$(curl -H "Accept: application/vnd.github.v3+json" "https://api.github.com/repos/openshift-kni/cnf-features-deploy/commits?per_page=1&sha=$PULL_BASE_REF" | jq -r '.[0].sha')
	else
		if [[ -z "$UPSTREAM_BRANCH" ]]; then
			latest_upstream_commit="origin/master"
		else
			latest_upstream_commit=$UPSTREAM_BRANCH
		fi

		if [[ ! $(git branch --list --all $latest_upstream_commit) ]]; then
			echo WARN: could not find $latest_upstream_commit, consider using a different UPSTREAM_BRANCH value
		fi

	fi
else
	latest_upstream_commit=$UPSTREAM_COMMIT
	if [[ ! $(git cat-file -t $UPSTREAM_COMMIT) == "commit" ]]; then
		echo WARN: $UPSTREAM_COMMIT commitish could not be found in repo
	fi
fi

commits_diff_count=$(git log --oneline $latest_upstream_commit..HEAD | wc -l)
if [[ $commits_diff_count -eq 0 ]]; then
	echo "WARN: no changes detected"
	exit 0
fi

echo commits between $latest_upstream_commit..HEAD:
git log --oneline $latest_upstream_commit..HEAD
echo

# check if we must skip any check for upstream commits
. .githooks/check-skipped-files.sh
filenames=$(git diff --name-only $latest_upstream_commit..HEAD | tr '\n' ' ')
echo checking upstream filenames: $filenames

if check_skipped_files "$filenames"; then
    echo "Files checks were skipped for upstream commits.Exiting"
    exit 0
else
    echo "No files checks were skipped for upstream commits"
fi

restricted_dirs=("ztp/source-crs/" "ztp/gitops-subscriptions/" "ztp/extra-manifests-builder/" "ztp/kube-compare-reference/")
echo checking upstream filenames in restricted-dirs: $filenames
# check if commits contain files in restricted directories
for filename in $filenames; do
    for dir in "${restricted_dirs[@]}"; do
        if [[ "$filename" == "$dir"* ]]; then
            echo "ERROR: $filename is in restricted directory $dir"
            echo "The following ztp directories have been moved (for release-4.19 and later): source-crs, extra-manifests, gitops-subscriptions and kube-compare-reference. Please create a pull request in https://github.com/openshift-kni/telco-reference"
            exit 16
        fi
    done
done

# list commits
for commitish in $(git log --oneline $latest_upstream_commit..HEAD | cut -d' ' -f 1); do
	commit_msg_filename=$(mktemp)
	if [[ $(git rev-list --no-walk --count --merges $commitish) == 0 ]]; then
		git log --format=%B -n 1 $commitish > $commit_msg_filename
		if ! .githooks/commit-msg $commit_msg_filename; then
			echo validation failed for $commitish
			exit 20
		fi
	fi
done
