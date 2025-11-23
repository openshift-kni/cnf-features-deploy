#!/usr/bin/env bash
IFS=$'\n\t'

test_script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="${test_script_dir}/../entrypoints"

# ensure generator is present
if [[ ! -f "${SRC}/generator" ]]; then
  echo "ERROR: generator script not found at ${SRC}/generator"
  exit 1
fi
# shellcheck source=../entrypoints/generator
source "${SRC}/generator"

function pass() { echo "PASS - $1"; }
function fail() { echo "FAIL - $1"; exit 1; }

# helper for running a command and matching its output
check_output() {
  local tc="$1"; shift
  local expected="$1"; shift
  local output
  output=$("$@" 2>&1) || true
  if [[ "${output}" =~ ${expected} ]]; then
    pass "${tc}"
  else
    fail "${tc} | expected '${expected}', got '${output}'"
  fi
}

# Set up mocks
function is_dir_mounted() {
    return 0
}
export -f is_dir_mounted

# Execute test cases
echo "Running tests in ${BASH_SOURCE[0]}:"

MOUNT_DIR="$test_script_dir/testData/siteconfigs"
check_output "test_install_with_non_exist_option" "unrecognized option" run_siteconfigGen --test example-sno.yaml
check_output "test_install_with_invalid_option_extra_manifest_only_value" "Expected value is true or false" run_siteconfigGen --extra-manifest-only=yes example-sno.yaml
check_output "test_install_src_path_does_not_exist" "is not found" run_siteconfigGen example.yaml
check_output "test_install_obsolute_src_path" "Processing SiteConfigs: ${MOUNT_DIR}/example-sno.yaml" run_siteconfigGen "${MOUNT_DIR}/example-sno.yaml"
check_output "test_install_relative_src_path" "Processing SiteConfigs: ${MOUNT_DIR}/example-sno.yaml" run_siteconfigGen example-sno.yaml
check_output "test_install_src_path_is_dir"    "Processing SiteConfigs: ${MOUNT_DIR}/example-sno.yaml" run_siteconfigGen .
check_output "test_install_dest_dir_not_given" "Generating installation CRs into ${MOUNT_DIR}/out/generated_installCRs" run_siteconfigGen example-sno.yaml
check_output "test_install_dest_dir_is_given"  "Generating installation CRs into ${MOUNT_DIR}/outDir" run_siteconfigGen example-sno.yaml outDir/

MOUNT_DIR="$test_script_dir/testData/policygentemplates"
check_output "test_config_with_non_exist_option" "unrecognized option" run_policyGen --test common-pgt.yaml
check_output "test_config_with_invalid_option_not_wrap_in_policy_value" "Expected value is true or false" run_policyGen --not-wrap-in-policy=no common-pgt.yaml
check_output "test_config_src_path_does_not_exist" "is not found" run_policyGen pgt.yaml
check_output "test_config_obsolute_src_path" "Processing PolicyGenTemplates: ${MOUNT_DIR}/common-pgt.yaml" run_policyGen "${MOUNT_DIR}/common-pgt.yaml"
check_output "test_config_relative_src_path" "Processing PolicyGenTemplates: ${MOUNT_DIR}/common-pgt.yaml" run_policyGen common-pgt.yaml
check_output "test_config_src_path_is_dir"    "Processing PolicyGenTemplates: ${MOUNT_DIR}/common-pgt.yaml" run_policyGen .
check_output "test_config_dest_dir_not_given" "Generating configuration CRs into ${MOUNT_DIR}/out/generated_configCRs" run_policyGen common-pgt.yaml
check_output "test_config_dest_dir_is_given"  "Generating configuration CRs into ${MOUNT_DIR}/outDir" run_policyGen common-pgt.yaml outDir/
