package siteConfig

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func Test_mergeExtraManifests(t *testing.T) {
	input := `
01-test-mc.yaml: |
  apiVersion: machineconfiguration.openshift.io/v1
  kind: MachineConfig
  metadata:
    labels:
      machineconfiguration.openshift.io/role: master
    name: 01-test-mc
  spec:
    config:
      ignition:
        version: 3.2.0
      storage:
        files:
        - contents:
            source: data:,%20
          mode: 384
          path: /root/test1
02-test-mc.yaml: |
  apiVersion: machineconfiguration.openshift.io/v1
  kind: MachineConfig
  metadata:
    labels:
      machineconfiguration.openshift.io/role: worker
    name: 02-test-mc
  spec:
    config:
      ignition:
        version: 3.2.0
      storage:
        files:
        - contents:
            source: data:,%20
          mode: 384
          path: /root/test2
03-test-mc.yaml: |
  apiVersion: machineconfiguration.openshift.io/v1
  kind: MachineConfig
  metadata:
    labels:
      machineconfiguration.openshift.io/role: master
    name: 03-test-mc
  spec:
    config:
      ignition:
        version: 3.2.0
      storage:
        files:
        - contents:
            source: data:,%20
          mode: 384
          path: /root/test3
`
	var dataMap map[string]interface{}
	doNotMerge := map[string]bool{"03-test-mc.yaml": true}
	err := yaml.Unmarshal([]byte(input), &dataMap)
	assert.NoError(t, err)

	dataMap, err = MergeManifests(dataMap, doNotMerge)
	assert.NoError(t, err)

	for _, role := range []string{"master", "worker"} {
		assert.NotNil(t, dataMap[fmt.Sprintf("%s-%s.yaml", McName, role)], "Expected predefined-extra-manifests for role %s", role)
	}
	assert.NotNil(t, dataMap["03-test-mc.yaml"], "Expected do not merge 03-test-mc")
}

func Test_notMergeNonMCManifest(t *testing.T) {
	input := `
test.yaml: |
  kind: Test
  metadata:
    name: test
01-test-mc.yaml: |
  apiVersion: machineconfiguration.openshift.io/v1
  kind: MachineConfig
  metadata:
    labels:
      machineconfiguration.openshift.io/role: worker
    name: 01-test-mc
  spec:
    config:
      ignition:
        version: 3.2.0
      storage:
        files:
        - contents:
            source: data:,%20
          mode: 384
          path: /root/test1
`
	var dataMap map[string]interface{}
	doNotMerge := map[string]bool{}
	err := yaml.Unmarshal([]byte(input), &dataMap)
	assert.NoError(t, err)

	dataMap, err = MergeManifests(dataMap, doNotMerge)
	assert.NoError(t, err)
	assert.NotNil(t, dataMap["test.yaml"], "Expected none MC extra-manifest")
}
