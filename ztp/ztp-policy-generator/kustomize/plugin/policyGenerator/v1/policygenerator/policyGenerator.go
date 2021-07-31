package main

import (
	"bytes"
	"fmt"
	policyGen "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/policyGen"
	siteConfigs "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/siteConfig"
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	"gopkg.in/yaml.v3"
	"os"
)

var sourcePoliciesPath string
var policyGenTempPath string
var outPath string
var stdout bool
var customResources bool
var siteConfigFlag bool

func main() {

	policyGenTempPath = os.Args[2]
	sourcePoliciesPath = os.Args[3]
	outPath = os.Args[4]
	stdout = (os.Args[5] == "true")
	customResources = (os.Args[6] == "true")
	siteConfigFlag = (os.Args[7] == "true")

	fHandler := utils.NewFilesHandler(sourcePoliciesPath, policyGenTempPath, outPath)

	if siteConfigFlag {
		scBuilder := siteConfigs.NewSiteConfigBuilder(fHandler)

		var buffer bytes.Buffer
		for _, file := range fHandler.GetPolicyGenTemplates() {
			siteConfig := siteConfigs.SiteConfig{}
			yamlFile := fHandler.ReadPolicyGenTempFile(file.Name())
			err := yaml.Unmarshal(yamlFile, &siteConfig)
			if err != nil {
				panic(err)
			}

			for clusterName, crs := range scBuilder.Build(siteConfig) {
				for _, crIntf := range crs {
					cr, err := yaml.Marshal(crIntf)
					if err != nil {
						panic(err)
					}
					buffer.Write(siteConfigs.Separator)
					buffer.Write(cr)
				}

				if stdout {
					fmt.Println(buffer.String())
				}
				fHandler.WriteFile(clusterName+utils.FileExt, buffer.Bytes())
				buffer.Reset()
			}
		}
	} else {
		for _, file := range fHandler.GetPolicyGenTemplates() {
			policyGenTemp := utils.PolicyGenTemplate{}
			yamlFile := fHandler.ReadPolicyGenTempFile(file.Name())
			err := yaml.Unmarshal(yamlFile, &policyGenTemp)
			if err != nil {
				panic(err)
			}
			pBuilder := policyGen.NewPolicyBuilder(policyGenTemp, fHandler, customResources)

			for k, v := range pBuilder.Build() {
				policy, _ := yaml.Marshal(v)
				if stdout {
					fmt.Println("---")
					fmt.Println(string(policy))
				}
				fHandler.WriteFile(k+utils.FileExt, policy)
			}
		}
	}
}
