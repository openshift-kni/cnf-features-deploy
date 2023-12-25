// All credit goes to
// https://github.com/kitproj/junit2html/blob/main/main.go
package main

import (
	_ "embed"
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jstemmer/go-junit-report/formatter"
)

//go:embed style.css
var styles string

func printTest(s formatter.JUnitTestSuite, c formatter.JUnitTestCase) {
	id := fmt.Sprintf("%s.%s.%s", s.Name, c.Classname, c.Name)
	class, text := "passed", "Pass"
	f := c.Failure
	if f != nil {
		class, text = "failed", "Fail"
	}
	k := c.SkipMessage
	if k != nil {
		class, text = "skipped", "Skip"
	}
	fmt.Printf("<div class='%s' id='%s'>\n", class, id)
	fmt.Printf("<a href='#%s'>%s <span class='badge'>%s</span></a>\n", id, c.Name, text)
	fmt.Printf("<div class='expando'>\n")
	if f != nil {
		fmt.Printf("<div class='content'>%s</div>\n", f.Contents)
	} else if k != nil {
		fmt.Printf("<div class='content'>%s</div>\n", k.Message)
	}
	d, _ := time.ParseDuration(c.Time)
	fmt.Printf("<p class='duration' title='Test duration'>%v</p>\n", d)
	fmt.Printf("</div>\n")
	fmt.Printf("</div>\n")
}

func main() {
	suites := &formatter.JUnitTestSuites{}

	err := xml.NewDecoder(os.Stdin).Decode(suites)
	if err != nil {
		panic(err)
	}

	fmt.Println("<html>")
	fmt.Println("<head>")
	fmt.Println("<meta charset=\"UTF-8\">")
	fmt.Println("<style>")
	fmt.Println(styles)
	fmt.Println("</style>")
	fmt.Println("</head>")
	fmt.Println("<body>")
	failures, total := 0, 0
	for _, s := range suites.Suites {
		failures += s.Failures
		total += len(s.TestCases)
	}
	fmt.Printf("<p>%d of %d tests failed</p>\n", failures, total)
	for _, s := range suites.Suites {
		if s.Failures > 0 {
			printSuiteHeader(s)
			for _, c := range s.TestCases {
				if f := c.Failure; f != nil {
					printTest(s, c)
				}
			}
		}
	}
	for _, s := range suites.Suites {
		printSuiteHeader(s)
		for _, c := range s.TestCases {
			if c.Failure == nil {
				printTest(s, c)
			}
		}
	}
	fmt.Println("</body>")
	fmt.Println("</html>")
}

func printSuiteHeader(s formatter.JUnitTestSuite) {
	fmt.Println("<h4>")
	fmt.Println(s.Name)
	for _, p := range s.Properties {
		if strings.HasPrefix(p.Name, "coverage.") {
			v, _ := strconv.ParseFloat(p.Value, 10)
			fmt.Printf("<span class='coverage' title='%s'>%.0f%%</span>\n", p.Name, v)
		}
	}
	fmt.Println("</h4>")
}
