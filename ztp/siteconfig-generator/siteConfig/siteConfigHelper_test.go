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

func Test_addZTPAnnotationToCRs(t *testing.T) {
	inputArray := [2]string{`
apiVersion: v1
kind: Namespace
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"
  labels:
    name: cluster1
  name: cluster1
`, `
apiVersion: metal3.io/v1alpha1
kind: BareMetalHost
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
    bmac.agent-install.openshift.io/hostname: node1
  name: node1
  namespace: cluster1
spec:
`}
	var clusterCRs []interface{}

	for i := 0; i < len(inputArray); i++ {
		var cr map[string]interface{}
		err := yaml.Unmarshal([]byte(inputArray[i]), &cr)
		assert.NoError(t, err)
		clusterCRs = append(clusterCRs, cr)
	}

	clusterCRs, _ = addZTPAnnotationToCRs(clusterCRs)
	for _, v := range clusterCRs {
		strExpected := v.(map[string]interface{})["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})[ZtpAnnotation]
		assert.Equal(t, strExpected, ZtpAnnotationDefaultValue, "Expected ztp annotation")
	}
}

func Test_addZTPAnnotationToManifest(t *testing.T) {

	manifestFile, err := ReadExtraManifestResourceFile("testdata/user-extra-manifest/user-extra-manifest.yaml")
	assert.NoError(t, err)
	manifest, err := addZTPAnnotationToManifest(string(manifestFile))
	assert.NoError(t, err)

	var expectedResult map[string]interface{}
	err = yaml.Unmarshal([]byte(manifest), &expectedResult)
	assert.NoError(t, err)

	strExpected := expectedResult["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})[ZtpAnnotation]
	assert.Equal(t, strExpected, ZtpAnnotationDefaultValue, "Expected ztp annotation")
}

func Test_agentClusterInstallAnnotation(t *testing.T) {

	testcases := []struct {
		networkType, installConfigOverride string
		expected                           string
		error                              error
		name                               string
	}{
		{
			networkType:           "OVNKubernetes",
			installConfigOverride: "{\"controlPlane\":{\"hyperthreading\":\"Disabled\"}}",
			expected:              "{\"networking\":{\"networkType\":\"OVNKubernetes\"},\"controlPlane\":{\"hyperthreading\":\"Disabled\"}}",
			error:                 nil,
			name:                  "Single json object set at installConfigOverride",
		},

		{
			networkType:           "OVNKubernetes",
			installConfigOverride: "{feature:{test:abc}}",
			expected:              "",
			error:                 fmt.Errorf("Invalid json parameter set at installConfigOverride"),
			name:                  "Invalid JSON set in installConfigOverride at SiteConfig",
		},

		{
			networkType:           "OVNKubernetes",
			installConfigOverride: "{\"controlPlane\":{\"hyperthreading\":\"Disabled\"},\"fips\":\"true\"}",
			expected:              "{\"networking\":{\"networkType\":\"OVNKubernetes\"},\"controlPlane\":{\"hyperthreading\":\"Disabled\"},\"fips\":\"true\"}",
			error:                 nil,
			name:                  "Multiple json object set at installConfigOverride",
		},

		{
			networkType:           "OVNKubernetes",
			installConfigOverride: "",
			expected:              "{\"networking\":{\"networkType\":\"OVNKubernetes\"}}",
			error:                 nil,
			name:                  "Json object when installConfigOverride is not set",
		},

		{
			networkType:           "OVNKubernetes",
			installConfigOverride: "{\"networking\":{\"UserManagedNetworking\":\"True\",\"DeprecatedType\":\"test\"},\"features\":[{\"abc\":\"test\"},{\"xyz\":\"test1\"}]}",
			expected:              "{\"features\":[{\"abc\":\"test\"},{\"xyz\":\"test1\"}],\"networking\":{\"DeprecatedType\":\"test\",\"UserManagedNetworking\":\"True\",\"networkType\":\"OVNKubernetes\"}}",
			error:                 nil,
			name:                  "installConfigOverride contains non-overlapping networking settings",
		},

		{
			networkType:           "OVNKubernetes",
			installConfigOverride: "{\"networking\":{\"UserManagedNetworking\":\"True\",\"networkType\":\"default\"},\"features\":[{\"abc\":\"test\"},{\"xyz\":\"test1\"}]}",
			expected:              "{\"features\":[{\"abc\":\"test\"},{\"xyz\":\"test1\"}],\"networking\":{\"UserManagedNetworking\":\"True\",\"networkType\":\"OVNKubernetes\"}}",
			error:                 nil,
			name:                  "installConfigOverride contains bad networking settings",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := agentClusterInstallAnnotation(tc.networkType, tc.installConfigOverride)
			if err != nil {
				assert.Equal(t, tc.error, err, "The expected and actual value should be the same.")
			}
			assert.Equal(t, tc.expected, actual, "The expected and actual value should be the same.")
		})
	}

}

func Test_transformNodeLabelAnnotation(t *testing.T) {
	inputArray := [2]string{`
apiVersion: v1
kind: Namespace
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"
  labels:
    name: cluster1
  name: cluster1
`, `
apiVersion: metal3.io/v1alpha1
kind: BareMetalHost
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
    bmac.agent-install.openshift.io/hostname: node1
  name: node1
  namespace: cluster1
spec:
`}

	testcases := []struct {
		nodeLabelKey, nodelLabelVal string
		expected                    string
		error                       error
		name                        string
	}{
		{
			nodeLabelKey:  "node-role.kubernetes.io/master",
			nodelLabelVal: "",
			expected:      "bmac.agent-install.openshift.io.node-label.node-role.kubernetes.io/master",
			error:         nil,
			name:          "Node label key Only",
		},

		{
			nodeLabelKey:  "node-role.kubernetes.io/environment",
			nodelLabelVal: "production",
			expected:      "bmac.agent-install.openshift.io.node-label.node-role.kubernetes.io/environment",
			error:         nil,
			name:          "Node label key and value",
		},
	}
	var clusterCRs []interface{}

	for i := 0; i < len(inputArray); i++ {
		var cr map[string]interface{}
		err := yaml.Unmarshal([]byte(inputArray[i]), &cr)
		assert.NoError(t, err)
		clusterCRs = append(clusterCRs, cr)
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			for _, v := range clusterCRs {
				cr := v.(map[string]interface{})
				if cr["kind"] == "BareMetalHost" {
					metadata, _ := cr["metadata"].(map[string]interface{})
					metadata["annotations"].(map[string]interface{})[nodeLabelPrefix] = map[string]string{tt.nodeLabelKey: tt.nodelLabelVal}
					cr = transformNodeLabelAnnotation(cr)
					if val, ok := cr["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})[tt.expected]; ok {
						assert.Equal(t, tt.nodelLabelVal, val, "Expected label value")
					} else {
						assert.Error(t, fmt.Errorf("Expected node label annotation was not found"))
					}
				}
			}
		})
	}
}
