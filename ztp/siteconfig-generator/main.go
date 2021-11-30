package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"

	siteConfigs "github.com/openshift-kni/cnf-features-deploy/ztp/siteconfig-generator/siteConfig"
	"gopkg.in/yaml.v3"
)

func main() {
	// Parse command input
	flag.Parse()

	// Collect and parse siteconfig files paths
	siteConfigFiles := flag.Args()
	var outputBuffer bytes.Buffer
	scBuilder, _ := siteConfigs.NewSiteConfigBuilder()

	for _, siteConfigFile := range siteConfigFiles {
		fileData, err := siteConfigs.ReadFile(siteConfigFile)
		if err != nil {
			log.Fatalf("Error: could not read file %s: %s\n", siteConfigFile, err)
		}

		siteConfig := siteConfigs.SiteConfig{}
		err = yaml.Unmarshal(fileData, &siteConfig)
		if err != nil {
			log.Fatalf("Error: could not parse %s as yaml: %s\n", siteConfigFile, err)
		}

		clusters, err := scBuilder.Build(siteConfig)
		if err != nil {
			log.Fatalf("Error: could not build the entire SiteConfig defined by %s: %s", siteConfigFile, err)
		}

		for _, crs := range clusters {
			for _, crIntf := range crs {
				cr, err := yaml.Marshal(crIntf)
				if err != nil {
					outputBuffer.Reset()
					log.Fatalf("Error: could not marshal generated cr by %s: %s %s", siteConfigFile, crIntf, err)
				}

				outputBuffer.Write(siteConfigs.Separator)
				outputBuffer.Write(cr)
			}

			fmt.Println(outputBuffer.String())
			outputBuffer.Reset()
		}
	}
}
