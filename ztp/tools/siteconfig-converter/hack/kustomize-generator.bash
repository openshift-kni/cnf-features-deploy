#!/bin/bash

# This script generates a kustomization.yaml file based on command-line arguments.
# It expects three arguments:
# 1. Name for the configMap
# 2. Namespace for the configMapG
# 3. Directory path to scan for .yaml files
# It then finds all .yaml files within the specified directory and
# includes them in the 'files' section of a configMapGenerator entry
# in the kustomization.yaml file.

echo "--- Kustomization.yaml Generator ---"

# Check if the correct number of arguments are provided
if [ "$#" -ne 3 ]; then
  echo "Usage: $0 <name> <namespace> <directory_path>"
  echo "Example: $0 my-config-map my-namespace ./extra-manifests"
  exit 1
fi

# Assign arguments to variables
config_name="$1"
config_namespace="$2"
dir_path="$3"

# Validate if the directory exists
if [ ! -d "$dir_path" ]; then
  echo "Error: Directory '$dir_path' not found."
  echo "Please ensure the path is correct and accessible."
  exit 1
fi

# Debug: Show the directory path being scanned
echo "Scanning directory: $(realpath "$dir_path")"

# Initialize the files array
yaml_files=()

# Find all .yaml and .yml files in the specified directory and its subdirectories
# The -print0 and while IFS= read -r -d $'\0' are used to handle filenames with spaces or special characters.
# The '< <(command)' syntax (process substitution) ensures the while loop runs in the
# current shell, allowing the 'yaml_files' array to be populated correctly.
while IFS= read -r -d $'\0' file; do
  # Add each found file to the array, relative to the current directory
  yaml_files+=("  - $file")
  # Debug: Show each file being added
  echo "Found and adding: $file"
done < <(find "$dir_path" -type f \( -name "*.yaml" -o -name "*.yml" \) -print0)

# Check if any YAML files were found
if [ ${#yaml_files[@]} -eq 0 ]; then
  echo "No .yaml or .yml files found in '$dir_path'. Generating kustomization.yaml with empty files list."
fi

# Define the output file name
output_file="kustomization.yaml"

# Write the content to the kustomization.yaml file using heredoc
cat <<EOF > $output_file
configMapGenerator:
- files:
$(for file_entry in "${yaml_files[@]}"; do
  echo "$file_entry"
done)
  name: $config_name
  namespace: $config_namespace
generatorOptions:
  disableNameSuffixHash: true
EOF

# Capture the content for display
kustomization_content=$(cat "$output_file")

echo "------------------------------------"
echo "kustomization.yaml generated successfully at: $output_file"
echo "Content:"
echo "$kustomization_content"
echo "------------------------------------"

