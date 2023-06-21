#!/bin/bash
# generate du ran artifacts based on the given PGTs

set -e

# Set ENV variables for ztp generator cli
export policyGenPath=$(pwd)/feature-configs/${FEATURES_ENVIRONMENT}/${FEATURES}/policyGen

feature=${FEATURES}
du_source_profile="ran-profile"
du_ci_std_profile="ran-profile-std-cloud"

# Supported DU features
du_features=($du_source_profile $du_ci_std_profile)

if [ "$FEATURES_ENVIRONMENT" != "cn-ran-overlays" ]; then
	echo "[ERROR]: FEATURES_ENVIRONMENT $FEATURES_ENVIRONMENT is not cn-ran-overlays"
	exit 1
fi

if [[ ! "${du_features[@]}" =~ "$feature" ]]; then
	echo "[ERROR]: FEATURES $FEATURES is not a DU feature"
	exit 1
fi

# Prepare resources required by ztp generator cli
ztp_generator_cli=ztp/resource-generator/entrypoints/generator
mkdir -p ${policyGenPath}
cp -r ztp/source-crs ${policyGenPath}
go build -o ${policyGenPath}/PolicyGenTemplate ./ztp/policygenerator

cluster_type="${CLUSTER_TYPE:-standard}"
if [ $feature == $du_ci_std_profile ]; then
    cluster_type="standard"
fi

feature_env_dir=$(pwd)/feature-configs/${FEATURES_ENVIRONMENT}/
feature_dir=${feature_env_dir}/${feature}/
source_du_dir=${feature_env_dir}/${du_source_profile}
output_artifacts_dir=${feature_dir}/cluster-config

# Determine source PGTs based on cluster type
source_pgts=("common-ranGen.yaml")
if [ $cluster_type == "standard" ];then
    source_pgts+=("group-du-standard-ranGen.yaml" "example-multinode-site.yaml")
elif [ $cluster_type == "sno" ]; then
    source_pgts+=("group-du-sno-ranGen.yaml" "example-sno-site.yaml")
elif [ $cluster_type == "3node" ]; then
    source_pgts+=("group-du-3node-ranGen.yaml" "example-multinode-site.yaml")
fi

# Call ztp generator cli to generate the non-policy wrapped cluster config CRs based on the source PGTs
# and export the generated CRs to the $feature_dir/cluster-config
for source_pgt in "${source_pgts[@]}"; do
    source_config_CRs="${ztp_generator_cli} config -N ${source_du_dir}/policygentemplates/${source_pgt} ${output_artifacts_dir}"
    if ! $source_config_CRs; then
        echo "[ERROR] Failed to generate the configuration CRs based on the source PGT: ${source_du_dir}/policygentemplates/${source_pgt}"
        exit 1
    fi
done

# Call ztp generator cli to generate the configuration CRs based on the overrided PGTs for CI environment
# and export the generated CRs to the $feature_dir/cluster-config.
# The PGTs in the CI feature repo don't overwrite everything but only define the least necessary updates
# for the CI test cluster and the newly generated CRs will replace the old ones generated based on the
# source PGTs above.
if [ $feature != $du_source_profile ]; then
    ci_config_CRs="${ztp_generator_cli} config -N ${feature_dir}/policygentemplates ${output_artifacts_dir}"
    if ! $ci_config_CRs; then
        echo "[ERROR] Failed to generate the configuration CRs based on the overrided PGTs for CI environment: ${feature_dir}/policygentemplates"
        exit 1
    fi
fi

if [[ ! -d ${output_artifacts_dir} ]]; then
  echo "${output_artifacts_dir} doesn't exist"
  exit 1
fi

excluded_configs="placeholder"
# Exclude the sriov configs for CI env running on cloud environment
if [ $feature == $du_ci_std_profile ]; then
    excluded_configs="SriovNetwork|SriovNetworkNodePolicy"
fi

# Auto-create kustomization yaml
cd ${feature_dir}
cat > kustomization.yaml << EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
EOF

find cluster-config/ -type f -name '*.yaml' | grep -viE ${excluded_configs} | while read config_yaml; do
  echo "- $config_yaml" >> kustomization.yaml
done
cd -

echo "[INFO] Display the generated kustomization.yaml"
cat ${feature_dir}/kustomization.yaml
