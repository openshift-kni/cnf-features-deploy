package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2/reporters"
)

func main() {
	output := flag.String("output", "-", "The output file name for the merged junit, defaults to stdout (-)")

	flag.Parse()

	junitFiles := flag.Args()

	suites, err := loadJUnitFiles(junitFiles)
	if err != nil {
		panic(fmt.Sprintf("Could not load JUnit files: %s", err))
	}

	mergedReport := mergeJUnitFiles(suites)

	writer, err := createOutputWriter(*output)
	if err != nil {
		panic(fmt.Sprintf("Failed to create the output file: %s", err))
	}
	defer writer.Close()

	err = writeJUnitFile(writer, mergedReport)
	if err != nil {
		panic(fmt.Sprintf("Failed to write the merged junit report: %s", err))
	}
}

func loadJUnitFiles(junitFiles []string) ([]reporters.JUnitTestSuites, error) {
	if len(junitFiles) == 0 {
		return nil, fmt.Errorf("no JUnit files provided")
	}

	result := []reporters.JUnitTestSuites{}

	for _, junitFile := range junitFiles {
		cleaned := filepath.Clean(junitFile)
		files, err := filepath.Glob(cleaned)
		if err != nil {
			return nil, err
		}
		if files == nil {
			return nil, fmt.Errorf("couldn't find files for %s", cleaned)
		}

		for _, file := range files {
			suites, err := decodeSuite(file)
			if err != nil {
				return nil, err
			}

			result = append(result, suites)
		}
	}

	return result, nil
}

func decodeSuite(fileName string) (reporters.JUnitTestSuites, error) {
	suites := reporters.JUnitTestSuites{}

	f, err := os.Open(fileName)
	if err != nil {
		return reporters.JUnitTestSuites{}, fmt.Errorf("failed to open file %s: %s", fileName, err)
	}
	defer f.Close()

	err = xml.NewDecoder(f).Decode(&suites)
	if err != nil {
		return reporters.JUnitTestSuites{}, fmt.Errorf("failed to decode suite %s: %s", fileName, err)
	}

	return suites, nil
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

func createOutputWriter(output string) (*os.File, error) {
	if output == "-" || output == "" {
		return os.Stdout, nil
	}

	writer, err := os.Create(output)
	if err != nil {
		return nil, err
	}

	return writer, nil
}

func writeJUnitFile(writer io.Writer, suite *reporters.JUnitTestSuites) error {
	writer.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(writer)
	encoder.Indent("  ", "	")
	err := encoder.Encode(suite)
	if err != nil {
		return err
	}

	return nil
}
