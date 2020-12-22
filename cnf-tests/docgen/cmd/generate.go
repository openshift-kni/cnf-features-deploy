package cmd

import (
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var (
	validationDescriptions string
	e2eDescriptions        string
	markdownFile           string
)

var toClean = [](*regexp.Regexp){
	regexp.MustCompile("\\[rfe_id:\\w+\\]"),
	regexp.MustCompile("\\[crit:\\w+\\]"),
	regexp.MustCompile("\\[level:\\w+\\]"),
	regexp.MustCompile("\\[test_id:\\d+\\]"),
	regexp.MustCompile("\\[vendor:cnf-qe@redhat.com\\]"),
}

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate generates the markdown doc starting from the description json file",
	Long: `generate takes a json file containing the mapping between 
test names and their descriptions, and fills out a markdown with test classification and
name clean up.`,
	Run: func(cmd *cobra.Command, args []string) {
		updateTestList(markdownFile, e2eDescriptions, validationDescriptions)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVar(&markdownFile, "target", "", "The path of the markdown file")
	generateCmd.Flags().StringVar(&e2eDescriptions, "testsjson", "", "The json file containing the descriptions of the e2e tests")
	generateCmd.Flags().StringVar(&validationDescriptions, "validationjson", "", "The json file containing the descriptions of the validation tests")
	cobra.MarkFlagRequired(generateCmd.LocalFlags(), "target")
	cobra.MarkFlagRequired(generateCmd.LocalFlags(), "testsjson")
	cobra.MarkFlagRequired(generateCmd.LocalFlags(), "validationjson")
}

func updateTestList(dest, e2eFile, validationsFile string) {
	e2e, err := readCurrentDescriptions(e2eFile)
	if err != nil {
		log.Fatalf("Failed to read e2e file %s %v", e2eFile, err)
	}
	validations, err := readCurrentDescriptions(validationsFile)
	if err != nil {
		log.Fatalf("Failed to read validations file %s %v", e2eFile, err)
	}

	data := TemplateData{}
	data.Features = make([]Feature, 0)
	data.ValidationList = descriptionsToList(validations, func(s string) bool {
		return true
	})
	dpdk := Feature{Name: "DPDK"}
	dpdk.Tests = descriptionsToList(e2e, func(name string) bool {
		return strings.Contains(name, "dpdk")
	})
	sriov := Feature{Name: "SR-IOV"}
	sriov.Tests = descriptionsToList(e2e, func(name string) bool {
		return strings.Contains(name, "sriov")
	})
	sctp := Feature{Name: "SCTP"}
	sctp.Tests = descriptionsToList(e2e, func(name string) bool {
		return strings.Contains(name, "sctp")
	})
	performance := Feature{Name: "Performance"}
	performance.Tests = descriptionsToList(e2e, func(name string) bool {
		return strings.Contains(name, "performance")
	})
	ptp := Feature{Name: "PTP"}
	ptp.Tests = descriptionsToList(e2e, func(name string) bool {
		return strings.Contains(name, "ptp")
	})
	others := Feature{Name: "Others"}
	others.Tests = descriptionsToList(e2e, func(name string) bool {
		return !strings.Contains(name, "dpdk") &&
			!strings.Contains(name, "sriov") &&
			!strings.Contains(name, "sctp") &&
			!strings.Contains(name, "performance") &&
			!strings.Contains(name, "ptp")
	})

	data.Features = append(data.Features,
		dpdk,
		sriov,
		sctp,
		performance,
		ptp,
		others)

	tmpl, err := template.New("test").Parse(testListTemplate)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(dest)
	if err != nil {
		log.Fatalf("Failed to open file %s - %v", dest, err)
	}
	defer f.Close()

	err = tmpl.Execute(f, data)
	if err != nil {
		log.Fatalf("Failed to execute template %v", err)
	}
}

func descriptionsToList(descriptions map[string]string, matcher func(string) bool) []TestDescription {
	res := []TestDescription{}
	// Since iterating over a map returns the list in random order, we maintain the order
	// while inserting into the list, so that the output is always the same
	for n, d := range descriptions {
		if !matcher(n) {
			continue
		}
		sanitizedName := sanitizeName(n)
		i := sort.Search(len(res), func(i int) bool { return res[i].Name >= sanitizedName })
		res = append(res, TestDescription{})
		copy(res[i+1:], res[i:])
		res[i] = TestDescription{sanitizedName, d}
	}
	return res
}

func sanitizeName(name string) string {
	res := name
	for _, r := range toClean {
		res = r.ReplaceAllString(res, "")
	}
	return res
}
