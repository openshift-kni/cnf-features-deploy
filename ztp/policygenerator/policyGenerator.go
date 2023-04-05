package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/policyGen"
	"github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/utils"
	"gopkg.in/yaml.v3"
	"log"
	"reflect"
	"strings"
)

func main() {

	sourceCRsPath := flag.String("sourcePath", utils.SourceCRsPath, "Directory where source-crs files exist")
	pgtPath := flag.String("pgtPath", utils.UnsetStringValue, "Directory where policyGenTemp files exist")
	outPath := flag.String("outPath", utils.UnsetStringValue, "Directory to write the generated policies")
	wrapInPolicy := flag.Bool("wrapInPolicy", true, "Wrap the CRs in acm Policy")

	// Parse command input
	flag.Parse()

	// Collect and parse policyGenTemp files paths
	policyGenTemps := flag.Args()

	fHandler := utils.NewFilesHandler(*sourceCRsPath, *pgtPath, *outPath)
	if fHandler.PgtDir != utils.UnsetStringValue {
		files, err := fHandler.GetTempFiles()
		if err != nil {
			log.Fatalf("Could not get file list from %s: %s", fHandler.PgtDir, err)
		}
		for _, file := range files {
			if file.Name()[0] == '.' {
				// Skip hidden files (for example, .gitignore or editor swap files)
				continue
			}
			policyGenTemps = append(policyGenTemps, fHandler.PgtDir+"/"+file.Name())
		}

	}

	InitiatePolicyGen(fHandler, policyGenTemps, *wrapInPolicy)
}

func InitiatePolicyGen(fHandler *utils.FilesHandler, pgtFiles []string, wrapInPolicy bool) {
	for _, file := range pgtFiles {

		kindType := utils.KindType{}
		yamlFile, err := fHandler.ReadFile(file)
		if err != nil {
			log.Fatalf("Could not read %s: %s\n", file, err)
		}

		err = yaml.Unmarshal(yamlFile, &kindType)
		if err != nil {
			log.Fatalf("Could not parse %s as yaml: %s\n", file, err)
		}

		if kindType.Kind == "PolicyGenTemplate" {
			policyGenTemp := utils.PolicyGenTemplate{}
			err := yaml.Unmarshal(yamlFile, &policyGenTemp)
			if err != nil {
				log.Fatalf("Could not unmarshal PolicyGenTemplate data from %s: %s", file, err)
			}
			// overwrite template setting with optional command line argument
			if !wrapInPolicy {
				policyGenTemp.Spec.WrapInPolicy = wrapInPolicy
			}

			pBuilder := policyGen.NewPolicyBuilder(fHandler)
			policies, err := pBuilder.Build(policyGenTemp)
			if err != nil {
				log.Fatalf("Could not build the entire policy defined by %s: %s", file, err)
			}
			for k, v := range policies {
				var policy []byte
				var pErr error
				t := reflect.ValueOf(v)
				switch t.Kind() {
				case reflect.Slice:
					var buf bytes.Buffer
					for i := 0; i < t.Len(); i++ {
						b, pErr := yaml.Marshal(t.Index(i).Interface())
						if pErr != nil {
							log.Fatalf("Error marshalling yaml for %s[%d]: %s\n", k, i, pErr)
						} else {
							if t.Len() > 0 {
								buf.WriteString("---\n")
							}
							buf.Write(b)
						}
					}
					policy = buf.Bytes()
				default:
					policy, pErr = yaml.Marshal(v)
					if pErr != nil {
						log.Fatalf("Error marshalling yaml for %s: %s", k, pErr)
					}
				}
				if pErr == nil {
					// write to file if CRs are unwrapped
					if !policyGenTemp.Spec.WrapInPolicy {
						if fHandler.OutDir == utils.UnsetStringValue {
							fHandler.OutDir = utils.DefaultOutDir
						}
					}
					// write to file when out dir is provided, otherwise write to standard output
					if fHandler.OutDir != utils.UnsetStringValue {
						err := fHandler.WriteFile(k+utils.FileExt, policy)
						if err != nil {
							log.Fatalf("Error: could not write file %s: %s", fHandler.OutDir+"/"+k+utils.FileExt, err)
						}
					} else {
						strPolicy := string(policy)
						if !strings.HasPrefix(strPolicy, "---\n") {
							fmt.Println("---")
						}
						fmt.Println(strPolicy)
					}
				}
			}
		} else {
			log.Printf("Unsupported yaml structure kind in %s: %s", file, kindType)
		}
	}
}
