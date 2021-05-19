#!/bin/bash

function finish {
	if [ -f "$commit_msg_filename" ]; then
		rm -f $commit_msg_filename
	fi
}
trap finish EXIT

# list commits
for commitish in $(git log --oneline origin/master..HEAD | cut -d' ' -f 1); do
	commit_msg_filename=$(mktemp)
	git log --format=%B -n 1 $commitish > $commit_msg_filename
	if ! .githooks/commit-msg $commit_msg_filename; then
		echo validation failed for $commitish
		exit 10
	fi
done
