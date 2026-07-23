package fileutils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommentOutLinesWithPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "scalar placeholder",
			input:    "apiVersion: v1\nmetadata:\n  name: $name\n",
			expected: "apiVersion: v1\nmetadata:\n#   name: $name\n",
		},
		{
			name:     "list item placeholder",
			input:    "items:\n  - $item\n",
			expected: "# items:\n#   - $item\n",
		},
		{
			name:     "bare placeholder",
			input:    "parent:\n  $value\n",
			expected: "# parent:\n#   $value\n",
		},
		{
			name: "list item with key-value placeholder",
			input: `status:
  conditions:
    - reason: $reason
      status: $status
      type: $type
`,
			expected: `status:
  conditions:
#     - reason: $reason
#       status: $status
#       type: $type
`,
		},
		{
			name: "mixed list items with and without placeholders",
			input: `spec:
  ports:
    - name: http
      port: 80
    - name: $portName
`,
			expected: `spec:
  ports:
    - name: http
      port: 80
#     - name: $portName
`,
		},
		{
			name:     "no placeholders unchanged",
			input:    "apiVersion: v1\nkind: ConfigMap\n",
			expected: "apiVersion: v1\nkind: ConfigMap\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(tmpFile, []byte(tt.input), 0o600); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			if err := CommentOutLinesWithPlaceholders(tmpFile); err != nil {
				t.Fatalf("CommentOutLinesWithPlaceholders returned error: %v", err)
			}

			got, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("failed to read result file: %v", err)
			}

			if string(got) != tt.expected {
				t.Errorf("mismatch\ngot:\n%s\nexpected:\n%s", string(got), tt.expected)
			}
		})
	}
}
