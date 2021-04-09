package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"gopkg.in/yaml.v3"
	utils "github.com/cnf-features-deploy/ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/utils"
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

	files, err := ioutil.ReadDir(ranGenPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		ranGenTemp :=  utils.RanGenTemplate{}
		fmt.Println(ranGenPath + "/" + file.Name())
		yamlFile, _ := ioutil.ReadFile(ranGenPath + "/" + file.Name())
		//fmt.Println(string(yamlFile))
		err = yaml.Unmarshal(yamlFile, &ranGenTemp)
		if err != nil {
			fmt.Println(err)
		}
		pBuilder := NewPolicyBuilder(ranGenTemp, sourcePoliciesPath)

		//fmt.Println(ranGenTemp)
		for k, v := range pBuilder.build() {
			fmt.Println(k)
			policy, _ := yaml.Marshal(v)
			fmt.Println(string(policy))
			ioutil.WriteFile( outPath + "/" + k + utils.FileExt, policy, 0644)
		}
	}
}
