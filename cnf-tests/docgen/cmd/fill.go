package cmd

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"
)

type TestSuite struct {
	XMLName  xml.Name `xml:"testsuite"`
	Text     string   `xml:",chardata"`
	Name     string   `xml:"name,attr"`
	Tests    string   `xml:"tests,attr"`
	Failures string   `xml:"failures,attr"`
	Errors   string   `xml:"errors,attr"`
	Time     string   `xml:"time,attr"`
	Testcase []struct {
		Text      string `xml:",chardata"`
		Name      string `xml:"name,attr"`
		Classname string `xml:"classname,attr"`
		Time      string `xml:"time,attr"`
	} `xml:"testcase"`
}

const emptyPlaceHolder = "XXXXXXXX"
const errMissing = "Found tests with no description"
const errRemoved = "Found tests that were removed"

var (
	junit       string
	description string
)

// fillCmd represents the fill command
var fillCmd = &cobra.Command{
	Use:   "fill",
	Short: "Fills the given json with the tests provided as xml",
	Long: `
fill checks if the description file contains all the tests, and eventually
add the new ones with an empty description. In that case, the command fails.`,
	Run: func(cmd *cobra.Command, args []string) {
		fill(junit, description)
	},
}

func init() {
	rootCmd.AddCommand(fillCmd)
	fillCmd.Flags().StringVar(&junit, "junit", "", "The junit file to use as an input")
	fillCmd.Flags().StringVar(&description, "description", "", "The json file containing the descriptions of the tests in the xml")
	cobra.MarkFlagRequired(fillCmd.LocalFlags(), "junit")
	cobra.MarkFlagRequired(fillCmd.LocalFlags(), "description")
}

func fill(xmlFile, descriptionsFile string) {
	data, err := ioutil.ReadFile(xmlFile)
	if err != nil {
		log.Fatalf("Failed reading file %s - %v", xmlFile, err)
	}

	var tests TestSuite
	err = xml.Unmarshal(data, &tests)
	if err != nil {
		log.Fatalf("xml.Unmarshal failed with '%s'\n", err)
	}

	currentDescriptions, err := readCurrentDescriptions(descriptionsFile)
	if err != nil {
		log.Fatalf("Failed to read current descriptions '%s'\n", err)
	}
	err = fillDescriptions(descriptionsFile, tests, currentDescriptions)
	if err != nil {
		log.Fatalf("Failed to fill missing descriptions '%s'\n", err)
	}
}

func fillDescriptions(fileName string, tests TestSuite, currentDescriptions map[string]string) error {
	missing := false
	removed := false
	allTests := make(map[string]bool)
	for _, t := range tests.Testcase {
		if _, ok := currentDescriptions[t.Name]; !ok {
			currentDescriptions[t.Name] = emptyPlaceHolder
			fmt.Printf("The test %s does not have a valid description\n", t.Name)
			missing = true
		}
		allTests[t.Name] = true
	}
	for k := range currentDescriptions {
		if _, ok := allTests[k]; !ok {
			delete(currentDescriptions, k)
			fmt.Printf("The test %s was removed from the test suite\n", k)
			removed = true
		}
	}
	jsonData, err := json.MarshalIndent(currentDescriptions, "", "    ")
	err = ioutil.WriteFile(fileName, jsonData, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open the descriptions file %v", err)
	}

	switch {
	case missing && removed:
		return fmt.Errorf("%s, %s", errMissing, errRemoved)
	case missing:
		return fmt.Errorf(errMissing)
	case removed:
		return fmt.Errorf(errRemoved)
	default:
		return nil
	}
}
