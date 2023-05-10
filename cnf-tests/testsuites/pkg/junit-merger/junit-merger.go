package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2/reporters"
	flag "github.com/spf13/pflag"
)

func main() {
	var output string
	flag.StringVarP(&output, "output", "o", "-", "File to write the resulting junit file to, defaults to stdout (-)")
	flag.Parse()
	junitFiles := flag.Args()

	if len(junitFiles) == 0 {
		panic("No JUnit files to merge provided.")
	}

	suites, err := loadJUnitFiles(junitFiles)
	if err != nil {
		panic(fmt.Sprintf("Could not load JUnit files. %s", err))
	}

	result := mergeJUnitFiles(suites)

	writer, err := prepareOutputWriter(output)
	if err != nil {
		panic(fmt.Sprintf("Failed to prepare the output file. %s", err))
	}

	err = writeJunitFile(writer, result)
	if err != nil {
		panic(fmt.Sprintf("Failed to write the merged junit report. %s", err))
	}
}

func loadJUnitFiles(fileGlobs []string) ([]reporters.JUnitTestSuites, error) {
	result := []reporters.JUnitTestSuites{}
	for _, fileglob := range fileGlobs {
		fileglob = filepath.Clean(fileglob)
		files, err := filepath.Glob(fileglob)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			f, err := os.Open(file)
			if err != nil {
				return nil, fmt.Errorf("failed to open file %s: %v", file, err)
			}
			suites := reporters.JUnitTestSuites{}
			err = xml.NewDecoder(f).Decode(&suites)
			if err != nil {
				return nil, fmt.Errorf("failed to decode suite %s: %v", file, err)
			}
			result = append(result, suites)
		}
	}

	return result, nil
}

func mergeJUnitFiles(suitesSlice []reporters.JUnitTestSuites) *reporters.JUnitTestSuites {
	result := &reporters.JUnitTestSuites{}

	for _, suites := range suitesSlice {
		for _, suite := range suites.TestSuites {
			result.TestSuites = append(result.TestSuites, suite)
			result.Time += suite.Time
			result.Tests += suite.Tests
			result.Failures += suite.Failures
			result.Errors += suite.Errors
		}
	}

	return result
}

func prepareOutputWriter(output string) (io.Writer, error) {
	writer := os.Stdout
	var err error
	if output != "-" && output != "" {
		writer, err = os.Create(output)
		if err != nil {
			return nil, err
		}
	}

	return writer, nil
}

func writeJunitFile(writer io.Writer, suite *reporters.JUnitTestSuites) error {
	writer.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(writer)
	encoder.Indent("  ", "	")
	err := encoder.Encode(suite)
	if err != nil {
		return err
	}

	return nil
}
