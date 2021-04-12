package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	utils "github.com/serngawy/cnf-features-deploy/ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/utils"
	policyGen "github.com/serngawy/cnf-features-deploy/ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/policyGen"
)

var sourcePoliciesPath string
var ranGenPath string
var outPath string
var genACM bool
var genK8sRes bool
var stdout bool


func main() {
	flag.StringVar(&sourcePoliciesPath, "sourcePath", "", "Path to source policies")
	flag.StringVar(&ranGenPath, "ranGenPath", "", "Path to Ran generator source Path")
	flag.StringVar(&outPath, "outPath", "", "Path to output the generated policies files")
	flag.BoolVar(&genK8sRes, "generateK8sResources", false, "Generate K8s resources")
	flag.BoolVar(&genACM, "generateACM", false, "Generate ACM policies")
	flag.BoolVar(&stdout, "stdout", false, "Print generated files to stdout")
	flag.Parse()

	fHandler := utils.NewFilesHandler(sourcePoliciesPath, ranGenPath, outPath)
	for _, file := range fHandler.GetRanGenTemplates() {
		ranGenTemp :=  utils.RanGenTemplate{}
		yamlFile := fHandler.ReadRanGenTempFile(file.Name())
		err := yaml.Unmarshal(yamlFile, &ranGenTemp)
		if err != nil {
			panic(err)
		}
		pBuilder := policyGen.NewPolicyBuilder(ranGenTemp, sourcePoliciesPath)

		for k, v := range pBuilder.Build() {
			policy, _ := yaml.Marshal(v)
			if stdout {
				fmt.Println(string(policy))
			}
			fHandler.WriteFile(k + utils.FileExt, policy)
		}
	}
}
