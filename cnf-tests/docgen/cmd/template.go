package cmd

type TestDescription struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Feature struct {
	Name  string
	Tests []TestDescription
}

type TemplateData struct {
	ValidationList []TestDescription
	Features       []Feature
}

const testListTemplate = `
<!--- IMPORTANT!
This file is generated manually. To add a new description please run
hack/fill-empty-docs.sh, check the json description files and fill the missing descriptions (the placeholder is XXXXXX)
--->

# Validation Test List

The validation tests are preliminary tests intended to verify that the instrumented features are available on the cluster.
| Test Name | Description |
| -- | ----------- |{{ with .ValidationList }}{{ range . }}
| {{ .Name }} | {{ .Description }} | {{ end }}{{ end }}

# CNF Tests List
The cnf tests instrument each different feature required by CNF. Following, a detailed description for each test.

{{ with .Features }}{{ range . }}
## {{ .Name }}

| Test Name | Description |
| -- | ----------- |{{ with .Tests }}{{ range . }}
| {{ .Name }} | {{ .Description }} | {{ end }}{{ end }}
{{ end }}{{ end }}
`
