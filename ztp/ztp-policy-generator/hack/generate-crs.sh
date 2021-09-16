#!/bin/bash

. $(dirname $0)/common.sh

PGTDIR=testPolicyGenTemplate
cp -a $PGTDIR $PGTDIR.backup

restorePgt() {
    rm -rf $PGTDIR
    mv $PGTDIR.backup $PGTDIR
}
trap restorePgt EXIT

sed -i -e 's/policyName: ".*"/policyName: ""/' testPolicyGenTemplate/*
./hack/generate-policy.sh
