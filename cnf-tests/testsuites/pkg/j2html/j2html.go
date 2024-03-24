package main

import (
	"encoding/xml"
	"flag"
	"html/template"
	"io"
	"log"
	"os"
	"sort"
)

type TestSuites struct {
	XMLName xml.Name    `xml:"testsuites"`
	Suites  []TestSuite `xml:"testsuite"`
}

type TestSuite struct {
	Name      string     `xml:"name,attr"`
	TestCases []TestCase `xml:"testcase"`
}

type TestCase struct {
	Name      string   `xml:"name,attr"`
	ClassName string   `xml:"classname,attr"`
	Failure   *Failure `xml:"failure,omitempty"`
	Skipped   *Skipped `xml:"skipped,omitempty"`
	SystemErr string   `xml:"system-err,omitempty"`
}

type Failure struct {
	Message string `xml:"message,attr"`
	Data    string `xml:",chardata"`
}

type Skipped struct {
	Message string `xml:"message,attr"`
}
type SuiteStats struct {
	Total   int
	Failed  int
	Passed  int
	Skipped int
}

type TestSuitesStats struct {
	Suites []SuiteStats
	Total  SuiteStats
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Test Report</title>
    <style>
        body {
            font-family: Arial, sans-serif;
        }
        .summary {
            margin-bottom: 20px;
        }
        .suite-stats {
            font-weight: bold;
        }
        .suite {
            margin-bottom: 10px;
			width: 80%;
        }
		.suite-table {
			width: 100%; /* Takes the full width of the parent container */
			/* Your table styles */
		}
        .test-case {
            cursor: pointer;
        }
        .test-case:hover {
            text-decoration: underline;
        }
        .test-case.failed {
            background-color: #ffcccc;
        }
        .test-case.passed {
            background-color: #ccffcc;
        }
        .test-case.skipped {
            background-color: #cccccc;
        }
        .details {
            display: none;
            padding-left: 20px;
        }
        .collapsible {
            cursor: pointer;
        }
        .collapsible:hover {
            text-decoration: underline;
        }
		.collapsible-failed {
			background: #ffcccc; /* Light red background for failed suites */
			border: 1px solid #dcdcdc;
			padding: 10px;
			margin-bottom: 5px;
			border-radius: 5px;
		}

		.collapsible-skipped {
			background: #f0f0f0; /* Light gray background for skipped suites */
			border: 1px solid #dcdcdc;
			padding: 10px;
			margin-bottom: 5px;
			border-radius: 5px;
		}

		.collapsible-passed {
			background: #ccffcc; /* Light green background for passed suites (no failures) */
			border: 1px solid #dcdcdc;
			padding: 10px;
			margin-bottom: 5px;
			border-radius: 5px;
		}

    </style>
</head>
<body>

<div class="summary">
    <h1>Test Report Summary</h1>
    <p>Total tests: {{.TotalStats.Total}}, Failed: {{.TotalStats.Failed}}, Passed: {{.TotalStats.Passed}}, Skipped: {{.TotalStats.Skipped}}</p>
</div>

{{range $index, $suite := .Suites}}
    <div class="suite">
	<h2 class="collapsible {{if gt (index $.SuiteStats $index).Failed 0}}collapsible-failed
	{{else if eq (index $.SuiteStats $index).Total (index $.SuiteStats $index).Skipped}}collapsible-skipped{{else}}collapsible-passed
	{{end}} {{if eq (index $.SuiteStats $index).Skipped (index $.SuiteStats $index).Total}}collapsed{{end}}" onclick="toggleSuite(this)">
		{{ $suite.Name }} - Total Tests: <u>{{ (index $.SuiteStats $index).Total }}</u> |
							Failed: <span style="color:red">{{ (index $.SuiteStats $index).Failed }}</span> |
							Passed: <span style="color:limegreen">{{ (index $.SuiteStats $index).Passed }}</span> |
							Skipped: <span style="color:gray">{{ (index $.SuiteStats $index).Skipped }}</span>
	</h2>

        <div class="suite-details {{if eq (index $.SuiteStats $index).Skipped (index $.SuiteStats $index).Total}}collapsed{{end}}">
            <table class="suite-table">
                <tbody>
                    {{range $suite.TestCases}}
                        <tr class="test-case {{if .Failure}}failed{{else if .Skipped}}skipped{{else}}passed{{end}}" onclick="toggleDetails('{{.Name}}')">
                            <td>{{.Name}}</td>
						</tr>
						<tr id="{{.Name}}" class="details">

							<td colspan="2">
							{{if .Failure}}{{.Failure.Message}}<br/>{{.Failure.Data}}{{else if .Skipped}}{{.Skipped.Message}}{{else}}Test passed.{{end}}
							{{if .SystemErr}}<pre>{{.SystemErr}}</pre>{{end}}
							</td>
						</tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>
{{end}}

<script>
    function toggleDetails2(testCaseRow) {
        var detailsRow = testCaseRow.nextElementSibling;
        detailsRow.style.display = detailsRow.style.display === 'none' ? '' : 'none';
    }
	function toggleDetails(id) {
		var x = document.getElementById(id);
		if (x.style.display === "none" || x.style.display === "") {
		x.style.display = "block";
		} else {
		x.style.display = "none";
		}
	}

    function toggleSuite(suiteHeader) {
        var suiteDetails = suiteHeader.nextElementSibling;
        if (suiteDetails.style.display === 'none') {
            suiteDetails.style.display = '';
            suiteHeader.classList.remove('collapsed');
        } else {
            suiteDetails.style.display = 'none';
            suiteHeader.classList.add('collapsed');
        }
    }

    // Collapse all suites where all tests are skipped
    document.addEventListener('DOMContentLoaded', function() {
        var collapsibleSuites = document.querySelectorAll('.suite .collapsed');
        collapsibleSuites.forEach(function(header) {
            toggleSuite(header);
        });
    });
</script>

</body>
</html>

`

func main() {

	// Define command-line flags
	inputFilePath := flag.String("input", "", "Path to the input JUnit XML report file")
	outputFilePath := flag.String("output", "", "Path to the output HTML report file")

	// Parse command-line flags
	flag.Parse()

	var reader io.Reader
	var writer io.Writer
	var err error

	// If an input file is provided, read from it; otherwise, read from STDIN
	if *inputFilePath != "" {
		file, err := os.Open(*inputFilePath)
		if err != nil {
			log.Fatalf("Failed to open input file: %s", err)
		}
		defer file.Close()
		reader = file
	} else {
		reader = os.Stdin
	}

	xmlData, err := io.ReadAll(reader)
	if err != nil {
		log.Fatalf("Failed to read input: %s", err)
	}

	var testSuites TestSuites
	if err := xml.Unmarshal(xmlData, &testSuites); err != nil {
		log.Fatal(err)
	}

	// Statistics

	var testSuitesStats TestSuitesStats

	// Calculating statistics for each suite and total
	for _, suite := range testSuites.Suites {
		var suiteStats SuiteStats
		for _, testCase := range suite.TestCases {
			suiteStats.Total++
			if testCase.Failure != nil {
				suiteStats.Failed++
			} else if testCase.Skipped != nil {
				suiteStats.Skipped++
			} else {
				suiteStats.Passed++
			}
		}
		testSuitesStats.Suites = append(testSuitesStats.Suites, suiteStats)
		testSuitesStats.Total.Total += suiteStats.Total
		testSuitesStats.Total.Failed += suiteStats.Failed
		testSuitesStats.Total.Passed += suiteStats.Passed
		testSuitesStats.Total.Skipped += suiteStats.Skipped
	}

	// end of statistics

	// Sorting test cases within each suite to have failures first, then passed, then skipped, with alphabetical order within each group
	for _, suite := range testSuites.Suites {
		sort.SliceStable(suite.TestCases, func(i, j int) bool {
			// Check failure status
			firstFailed := suite.TestCases[i].Failure != nil
			secondFailed := suite.TestCases[j].Failure != nil
			if firstFailed != secondFailed {
				return firstFailed
			}

			// Check skip status if neither are failed or both are failed
			firstSkipped := suite.TestCases[i].Skipped != nil
			secondSkipped := suite.TestCases[j].Skipped != nil
			if firstSkipped != secondSkipped {
				return secondSkipped // Prioritize non-skipped
			}

			// If both are of the same status, sort alphabetically by test case name
			return suite.TestCases[i].Name < suite.TestCases[j].Name
		})
	}

	// If an output file is provided, write to it; otherwise, write to STDOUT
	if *outputFilePath != "" {
		file, err := os.Create(*outputFilePath)
		if err != nil {
			log.Fatalf("Failed to create output file: %s", err)
		}
		defer file.Close()
		writer = file
	} else {
		writer = os.Stdout
	}

	// Execute the template and write to the chosen output destination
	tmpl := template.Must(template.New("report").Parse(htmlTemplate))
	if err := tmpl.Execute(writer, struct {
		Suites     []TestSuite
		SuiteStats []SuiteStats
		TotalStats SuiteStats
	}{
		Suites:     testSuites.Suites,
		SuiteStats: testSuitesStats.Suites,
		TotalStats: testSuitesStats.Total,
	}); err != nil {
		log.Fatalf("Failed to execute template: %s", err)
	}

}
