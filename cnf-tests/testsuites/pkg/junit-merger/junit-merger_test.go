package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestJUnitMerger(t *testing.T) {
	suites, err := loadJUnitFiles([]string{"testdata/junit1.xml", "testdata/junit2.xml"})
	if err != nil {
		t.Fatalf("Could not load JUnit files. %s", err)
	}

	result := mergeJUnitFiles(suites)

	writer, err := createOutputWriter("testdata/result.xml")
	if err != nil {
		panic(fmt.Sprintf("Failed to prepare the output file. %s", err))
	}

	err = writeJUnitFile(writer, result)
	if err != nil {
		panic(fmt.Sprintf("Failed to write the merged junit report. %s", err))
	}

	expected, err := os.ReadFile("testdata/merged.golden")
	if err != nil {
		t.Fatalf("test failed reading .golden file: %s", err)
	}

	got, err := os.ReadFile("testdata/result.xml")
	if err != nil {
		t.Fatalf("test failed reading the result file: %s", err)
	}

	if !cmp.Equal(string(expected), string(got)) {
		t.Fatalf("test failed. (-want +got):\n%s", cmp.Diff(string(expected), string(got)))
	}
}

func TestJUnitMegerNoFiles(t *testing.T) {
	_, err := loadJUnitFiles([]string{})
	if err == nil {
		t.Fatalf("loadJUnitFiles didn't return expected error.")
	}
}

func TestJUnitMegerFilesNotFound(t *testing.T) {
	_, err := loadJUnitFiles([]string{"notfound.xml"})
	if err == nil {
		t.Fatalf("loadJUnitFiles didn't return expected error.")
	}
}
