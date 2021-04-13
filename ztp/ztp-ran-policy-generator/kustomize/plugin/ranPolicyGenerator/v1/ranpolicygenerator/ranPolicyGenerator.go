package main

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
	utils "github.com/serngawy/cnf-features-deploy/ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/utils"
	policyGen "github.com/serngawy/cnf-features-deploy/ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/policyGen"
)

var sourcePoliciesPath string
var ranGenPath string
var outPath string
var stdout bool



func main() {

	ranGenPath = os.Args[2]
	sourcePoliciesPath = os.Args[3]
	outPath = os.Args[4]
	stdout = (os.Args[5] == "true")


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
