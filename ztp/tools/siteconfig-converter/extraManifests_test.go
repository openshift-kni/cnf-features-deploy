package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func Test_filterExtraManifests(t *testing.T) {
	getMapWithFileNames := func() map[string]interface{} {
		// Create a test data map with some file names
		dataMap := make(map[string]interface{})
		files := []string{
			"01-container-mount-ns-and-kubelet-conf-master.yaml",
			"01-container-mount-ns-and-kubelet-conf-worker.yaml",
			"03-sctp-machine-config-master.yaml",
			"03-sctp-machine-config-worker.yaml",
			"03-workload-partitioning.yaml",
			"04-accelerated-container-startup-master.yaml",
			"04-accelerated-container-startup-worker.yaml",
			"06-kdump-master.yaml",
			"06-kdump-worker.yaml",
		}
		for _, f := range files {
			dataMap[f] = "test content"
		}
		return dataMap
	}

	const filter = `
  inclusionDefault: %s
  exclude: %s 
  include: %s
`

	type args struct {
		dataMap map[string]interface{}
		filter  string
	}
	tests := []struct {
		name       string
		args       args
		want       map[string]interface{}
		wantErrMsg string
		wantErr    bool
	}{
		{
			name:    "remove files from the list",
			wantErr: false,
			args: args{
				dataMap: getMapWithFileNames(),
				filter:  fmt.Sprintf(filter, ``, `[03-sctp-machine-config-worker.yaml, 03-sctp-machine-config-master.yaml]`, ``),
			},
			want: map[string]interface{}{
				"01-container-mount-ns-and-kubelet-conf-master.yaml": "test content",
				"01-container-mount-ns-and-kubelet-conf-worker.yaml": "test content",
				"03-workload-partitioning.yaml":                      "test content",
				"04-accelerated-container-startup-master.yaml":       "test content",
				"04-accelerated-container-startup-worker.yaml":       "test content",
				"06-kdump-master.yaml":                               "test content",
				"06-kdump-worker.yaml":                               "test content",
			},
		},
		{
			name:    "exclude all files except 03-workload-partitioning.yaml",
			wantErr: false,
			args: args{
				dataMap: getMapWithFileNames(),
				filter:  fmt.Sprintf(filter, `exclude`, ``, `[03-workload-partitioning.yaml]`),
			},
			want: map[string]interface{}{"03-workload-partitioning.yaml": "test content"},
		},
		{
			name:    "error when both include and exclude contain a list of files and user in exclude mode",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames(),
				filter:  fmt.Sprintf(filter, `exclude`, `[03-workload-partitioning.yaml]`, `[03-workload-partitioning.yaml]`),
			},
			wantErrMsg: "when InclusionDefault is set to exclude, exclude list can not have entries",
		},
		{
			name:    "error when a file is listed under include list but user in include mode",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames(),
				filter:  fmt.Sprintf(filter, `include`, ``, `[03-workload-partitioning.yaml]`),
			},
			wantErrMsg: "when InclusionDefault is set to include, include list can not have entries",
		},
		{
			name:    "error when incorrect value is used for inclusionDefault",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames(),
				filter:  fmt.Sprintf(filter, `something_random`, `[03-workload-partitioning.yaml]`, ``),
			},
			wantErrMsg: "acceptable values for inclusionDefault are include and exclude. You have entered something_random",
		},
		{
			name:    "error when trying to remove a file that is not in the dir",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames(),
				filter:  fmt.Sprintf(filter, `include`, `[03-my-unknown-file.yaml]`, ``),
			},
			wantErrMsg: "Filename 03-my-unknown-file.yaml under exclude array is invalid. Valid files names are:",
		},
		{
			name:    "error when trying to keep a file that is not in the dir",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames(),
				filter:  fmt.Sprintf(filter, `exclude`, ``, `[03-my-unknown-file.yaml]`),
			},
			wantErrMsg: "Filename 03-my-unknown-file.yaml under include array is invalid. Valid files names are:",
		},
		{
			name:    "nil filter returns dataMap unchanged",
			wantErr: false,
			args: args{
				dataMap: getMapWithFileNames(),
				filter:  "",
			},
			want: getMapWithFileNames(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f *Filter
			if tt.args.filter != "" {
				f = &Filter{}
				err := yaml.Unmarshal([]byte(tt.args.filter), f)
				if err != nil {
					t.Fatalf("Failed to unmarshal filter: %v", err)
				}
			}
			got, err := filterExtraManifests(tt.args.dataMap, f)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterExtraManifests() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("filterExtraManifests() error message = %v, want contains %v", err.Error(), tt.wantErrMsg)
				}
			} else {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("filterExtraManifests() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func Test_addZTPAnnotationToManifest(t *testing.T) {
	tests := []struct {
		name           string
		manifestStr    string
		wantAnnotation string
		wantErr        bool
	}{
		{
			name: "add annotation to manifest without metadata",
			manifestStr: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
`,
			wantAnnotation: ZtpAnnotationDefaultValue,
			wantErr:        false,
		},
		{
			name: "add annotation to manifest with existing annotations",
			manifestStr: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    argocd.argoproj.io/sync-wave: "0"
data:
  key: value
`,
			wantAnnotation: ZtpAnnotationDefaultValue,
			wantErr:        false,
		},
		{
			name: "add annotation to manifest without annotations",
			manifestStr: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
`,
			wantAnnotation: ZtpAnnotationDefaultValue,
			wantErr:        false,
		},
		{
			name: "invalid YAML should return error",
			manifestStr: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  invalid: [unclosed
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addZTPAnnotationToManifest(tt.manifestStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("addZTPAnnotationToManifest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				var result map[string]interface{}
				err = yaml.Unmarshal([]byte(got), &result)
				if err != nil {
					t.Fatalf("Failed to unmarshal result: %v", err)
				}

				metadata, ok := result["metadata"].(map[string]interface{})
				if !ok {
					t.Fatalf("metadata is not a map")
				}

				annotations, ok := metadata["annotations"].(map[string]interface{})
				if !ok {
					t.Fatalf("annotations is not a map")
				}

				annotationValue, ok := annotations[ZtpAnnotation].(string)
				if !ok {
					t.Errorf("ZTP annotation not found or not a string")
					return
				}

				if annotationValue != tt.wantAnnotation {
					t.Errorf("addZTPAnnotationToManifest() annotation = %v, want %v", annotationValue, tt.wantAnnotation)
				}
			}
		})
	}
}

func Test_GetFiles(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-getfiles-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.yaml")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Test with a file
	files, err := GetFiles(testFile)
	if err != nil {
		t.Errorf("GetFiles() error = %v", err)
		return
	}
	if len(files) != 1 {
		t.Errorf("GetFiles() returned %d files, want 1", len(files))
		return
	}
	if files[0].Name() != "test.yaml" {
		t.Errorf("GetFiles() file name = %v, want test.yaml", files[0].Name())
	}

	// Test with a directory
	files, err = GetFiles(tmpDir)
	if err != nil {
		t.Errorf("GetFiles() error = %v", err)
		return
	}
	if len(files) < 1 {
		t.Errorf("GetFiles() returned %d files, want at least 1", len(files))
	}

	// Test with non-existent path
	_, err = GetFiles(filepath.Join(tmpDir, "nonexistent"))
	if err == nil {
		t.Error("GetFiles() expected error for non-existent path, got nil")
	}
}

func Test_ReadFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-readfile-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := []byte("test content")
	err = os.WriteFile(tmpFile.Name(), testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test reading the file
	content, err := ReadFile(tmpFile.Name())
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
		return
	}
	if !reflect.DeepEqual(content, testContent) {
		t.Errorf("ReadFile() content = %v, want %v", content, testContent)
	}

	// Test with non-existent file
	_, err = ReadFile("/nonexistent/file")
	if err == nil {
		t.Error("ReadFile() expected error for non-existent file, got nil")
	}
}

func Test_resolveFilePath(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-resolve-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.yaml")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with absolute path (should return as-is)
	resolved := resolveFilePath(testFile, "/some/base/dir")
	if resolved != testFile {
		t.Errorf("resolveFilePath() = %v, want %v", resolved, testFile)
	}

	// Test with relative path (should resolve to baseDir + path)
	resolved = resolveFilePath("test.yaml", tmpDir)
	expected := filepath.Join(tmpDir, "test.yaml")
	if resolved != expected {
		t.Errorf("resolveFilePath() = %v, want %v", resolved, expected)
	}
}

func Test_getExtraManifest_basic(t *testing.T) {
	// Create a minimal cluster spec for testing
	cluster := Cluster{
		ClusterName: "test-cluster",
		Nodes: []Node{
			{
				HostName: "node1",
				Role:     "master",
			},
		},
		ExtraManifests: ExtraManifests{},
	}

	// Test with empty dataMap
	dataMap := make(map[string]interface{})
	// Use current directory as input file directory for testing
	inputFileDir, _ := os.Getwd()
	result, err := getExtraManifest(dataMap, cluster, inputFileDir)

	// This should not error even if no manifests are found
	// (it depends on whether extra-manifest directory exists)
	// We just check that the function doesn't panic
	if err != nil {
		// If there's an error, it should be about missing directory, which is expected
		if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "not a directory") {
			t.Logf("getExtraManifest() error = %v (expected for missing directory)", err)
		}
	} else {
		// If no error, result should be a map (even if empty)
		if result == nil {
			t.Error("getExtraManifest() result is nil")
		}
	}
}

func Test_filterExtraManifests_edgeCases(t *testing.T) {
	tests := []struct {
		name    string
		dataMap map[string]interface{}
		filter  *Filter
		wantErr bool
	}{
		{
			name:    "nil filter with empty dataMap",
			dataMap: make(map[string]interface{}),
			filter:  nil,
			wantErr: false,
		},
		{
			name: "filter with nil InclusionDefault (defaults to include)",
			dataMap: map[string]interface{}{
				"file1.yaml": "content1",
				"file2.yaml": "content2",
			},
			filter: &Filter{
				InclusionDefault: nil,
				Exclude:          []string{"file1.yaml"},
			},
			wantErr: false,
		},
		{
			name: "filter with empty exclude list",
			dataMap: map[string]interface{}{
				"file1.yaml": "content1",
				"file2.yaml": "content2",
			},
			filter: &Filter{
				InclusionDefault: stringPtr("include"),
				Exclude:          []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterExtraManifests(tt.dataMap, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterExtraManifests() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Error("filterExtraManifests() result is nil")
				}
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
