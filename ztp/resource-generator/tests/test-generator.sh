#!/bin/bash

test_script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

SRC="${test_script_dir}/../entrypoints"
source ${SRC}/generator 2>/dev/null

function pass() {
    echo "PASS - $1"
}

function fail() {
    echo "FAIL - $1"
    exit 1
}

# Set up mocks
function is_dir_mounted() {
    return 0
}
export -f is_dir_mounted

# Execute test cases
echo "Running tests in ${BASH_SOURCE[0]}:"

# install test cases
MOUNT_DIR="$test_script_dir/testData/siteconfigs"
tc="test_install_with_non_exist_option"
result=$(run_siteconfigGen --test example-sno.yaml 2>&1)
[[ $result =~ "unrecognized option" ]] && pass $tc || fail $tc

tc="test_install_with_invalid_option_extra_manifest_only_value"
result=$(run_siteconfigGen --extra-manifest-only=yes example-sno.yaml 2>&1)
[[ $result =~ "Expected value is true or false" ]] && pass $tc || fail $tc

tc="test_install_src_path_does_not_exist"
result=$(run_siteconfigGen example.yaml 2>&1)
[[ $result =~ "is not found" ]] && pass $tc || fail $tc

tc="test_install_absolute_src_path"
result=$(run_siteconfigGen ${MOUNT_DIR}/example-sno.yaml 2>/dev/null)
[[ $result =~ "Processing SiteConfigs: ${MOUNT_DIR}/example-sno.yaml" ]] && pass $tc || fail $tc

tc="test_install_relative_src_path"
result=$(run_siteconfigGen example-sno.yaml 2>/dev/null)
[[ $result =~ "Processing SiteConfigs: ${MOUNT_DIR}/example-sno.yaml" ]] && pass $tc || fail $tc

tc="test_install_src_path_is_dir"
result=$(run_siteconfigGen . 2>/dev/null)
[[ $result =~ "Processing SiteConfigs: ${MOUNT_DIR}/example-sno.yaml" ]] && pass $tc || fail $tc

tc="test_install_dest_dir_not_given"
result=$(run_siteconfigGen example-sno.yaml 2>/dev/null)
[[ $result =~ "Generating installation CRs into ${MOUNT_DIR}/out/generated_installCRs" ]] && pass $tc || fail $tc

tc="test_install_dest_dir_is_given"
result=$(run_siteconfigGen example-sno.yaml outDir/ 2>/dev/null)
[[ $result =~ "Generating installation CRs into ${MOUNT_DIR}/outDir" ]] && pass $tc || fail $tc

# config test cases
MOUNT_DIR="$test_script_dir/testData/policygentemplates"
tc="test_config_with_non_exist_option"
result=$(run_policyGen --test common-pgt.yaml 2>&1)
[[ $result =~ "unrecognized option" ]] && pass $tc || fail $tc

tc="test_config_with_invalid_option_not_wrap_in_policy_value"
result=$(run_policyGen --not-wrap-in-policy=no common-pgt.yaml 2>&1)
[[ $result =~ "Expected value is true or false" ]] && pass $tc || fail $tc

tc="test_config_src_path_does_not_exist"
result=$(run_policyGen pgt.yaml 2>&1)
[[ $result =~ "is not found" ]] && pass $tc || fail $tc

tc="test_config_absolute_src_path"
result=$(run_policyGen ${MOUNT_DIR}/common-pgt.yaml 2>/dev/null)
[[ $result =~ "Processing PolicyGenTemplates: ${MOUNT_DIR}/common-pgt.yaml" ]] && pass $tc || fail $tc

tc="test_config_relative_src_path"
result=$(run_policyGen common-pgt.yaml 2>/dev/null)
[[ $result =~ "Processing PolicyGenTemplates: ${MOUNT_DIR}/common-pgt.yaml" ]] && pass $tc || fail $tc

tc="test_config_src_path_is_dir"
result=$(run_policyGen . 2>/dev/null)
[[ $result =~ "Processing PolicyGenTemplates: ${MOUNT_DIR}/common-pgt.yaml" ]] && pass $tc || fail $tc

tc="test_config_dest_dir_not_given"
result=$(run_policyGen common-pgt.yaml 2>/dev/null)
[[ $result =~ "Generating configuration CRs into ${MOUNT_DIR}/out/generated_configCRs" ]] && pass $tc || fail $tc

tc="test_config_dest_dir_is_given"
result=$(run_policyGen common-pgt.yaml outDir/ 2>/dev/null)
[[ $result =~ "Generating configuration CRs into ${MOUNT_DIR}/outDir" ]] && pass $tc || fail $tc
