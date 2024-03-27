// Copyright Contributors to the Open Cluster Management project
// The content of this file mainly comes from https://github.com/open-cluster-management-io/policy-generator-plugin/blob/main/internal/utils.go
package patches

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// unmarshalManifestFile unmarshals the input object manifest/definition file into
// a slice in order to account for multiple YAML documents in the same file.
// If the file cannot be decoded or each document is not a map, an error will
// be returned.
func UnmarshalManifestFile(manifestPath string) ([]map[string]interface{}, error) {
	// #nosec G304
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read the manifest file %s", manifestPath)
	}

	rv, err := unmarshalManifestBytes(manifestBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the manifest file at %s: %w", manifestPath, err)
	}

	return rv, nil
}

// unmarshalManifestBytes unmarshals the input bytes slice of an object manifest/definition file
// into a slice of maps in order to account for multiple YAML documents in the bytes slice. If each
// document is not a map, an error will be returned.
func unmarshalManifestBytes(manifestBytes []byte) ([]map[string]interface{}, error) {
	yamlDocs := []map[string]interface{}{}
	d := yaml.NewDecoder(bytes.NewReader(manifestBytes))

	for {
		var obj interface{}

		err := d.Decode(&obj)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			//nolint:wrapcheck
			return nil, err
		}

		if _, ok := obj.(map[string]interface{}); !ok && obj != nil {
			err := errors.New("the input manifests must be in the format of YAML objects")

			return nil, err
		}

		if obj != nil {
			yamlDocs = append(yamlDocs, obj.(map[string]interface{}))
		}
	}

	return yamlDocs, nil
}

// verifyManifestPath verifies that the manifest path is in the directory tree under baseDirectory.
// An error is returned if it is not or the paths couldn't be properly resolved.
func VerifyManifestPath(baseDirectory, manifestPath string) error {
	absPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return fmt.Errorf("could not resolve the manifest path %s to an absolute path", manifestPath)
	}

	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return fmt.Errorf("could not resolve symlinks to the manifest path %s", manifestPath)
	}

	relPath, err := filepath.Rel(baseDirectory, absPath)
	if err != nil {
		return fmt.Errorf(
			"could not resolve the manifest path %s to a relative path from the kustomization.yaml file",
			manifestPath,
		)
	}

	if relPath == "." {
		return fmt.Errorf(
			"the manifest path %s may not refer to the same directory as the kustomization.yaml file",
			manifestPath,
		)
	}

	parDir := ".." + string(filepath.Separator)
	if strings.HasPrefix(relPath, parDir) || relPath == ".." {
		return fmt.Errorf(
			"the manifest path %s is not in the same directory tree as the kustomization.yaml file",
			manifestPath,
		)
	}

	return nil
}
