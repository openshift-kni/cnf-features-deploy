#!/usr/bin/env bash
# Comment "/close" on all abandoned PRs in openshift-kni/cnf-features-deploy
# Run: gh auth login   (first, if not already logged in)
# Then: ./scripts/close-abandoned-prs.sh

REPO="openshift-kni/cnf-features-deploy"
# Abandoned PR numbers from https://github.com/openshift-kni/cnf-features-deploy/pulls?q=is%3Apr+is%3Aopen+abandoned
PRs=(3695 3693 3691 3636 3635 3629 3628 3625 3622 3610 3608 3606 3605 3604 3600 3599 3597 3593 3592 3389 3073)

for pr in "${PRs[@]}"; do
  echo "Commenting /close on PR #$pr..."
  gh pr comment "$pr" --repo "$REPO" --body "/close" || echo "  Failed for #$pr"
done

echo "Done."
