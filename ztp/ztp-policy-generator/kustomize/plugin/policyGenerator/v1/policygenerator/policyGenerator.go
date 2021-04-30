package main

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	policyGen "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/policyGen"
)

var sourcePoliciesPath string
var policyGenTempPath string
var outPath string
var stdout bool
var customResources bool

func main() {

	policyGenTempPath = os.Args[2]
	sourcePoliciesPath = os.Args[3]
	outPath = os.Args[4]
	stdout = (os.Args[5] == "true")
	customResources = (os.Args[6] == "true")

	fHandler := utils.NewFilesHandler(sourcePoliciesPath, policyGenTempPath, outPath)

	for _, file := range fHandler.GetPolicyGenTemplates() {
		policyGenTemp :=  utils.PolicyGenTemplate{}
		yamlFile := fHandler.ReadPolicyGenTempFile(file.Name())
		err := yaml.Unmarshal(yamlFile, &policyGenTemp)
		if err != nil {
			panic(err)
		}
		pBuilder := policyGen.NewPolicyBuilder(policyGenTemp, sourcePoliciesPath)

		for k, v := range pBuilder.Build(customResources) {
			policy, _ := yaml.Marshal(v)
			if stdout {
				fmt.Println("---")
				fmt.Println(string(policy))
			}
			fHandler.WriteFile(k + utils.FileExt, policy)
		}
	}
}
