package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	policyGen "github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/policyGen"
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/utils"
	"gopkg.in/yaml.v3"
)

func main() {

	var sourcePath string
	var tempPath string
	var outPath string
	var stdout bool
	var wrapInPolicy bool

	wrapInPolicy = true // default generate wrapped policies
	tempPath = os.Args[2]
	sourcePath = os.Args[3]
	outPath = os.Args[4]
	stdout = (os.Args[5] == "true")
	if len(os.Args) >= 7 {
		wrapInPolicy = os.Args[6] == "true"
	}
	InitiatePolicyGen(tempPath, sourcePath, outPath, stdout, wrapInPolicy)
}

func InitiatePolicyGen(tempPath string, sourcePath string, outPath string, stdout bool, wrapInPolicy bool) {

	fHandler := utils.NewFilesHandler(sourcePath, tempPath, outPath)
	files, err := fHandler.GetTempFiles()
	if err != nil {
		log.Printf("Could not get file list from %s: %s", tempPath, err)
		// The 'files' slice will be empty, so just continue on.
	}

	for _, file := range files {
		if file.Name()[0] == '.' {
			// Skip hidden files (for example, .gitignore or editor swap files)
			continue
		}

		kindType := utils.KindType{}
		yamlFile, err := fHandler.ReadTempFile(file.Name())
		if err != nil {
			log.Printf("Could not read %s: %s\n", file.Name(), err)
			continue
		}

		err = yaml.Unmarshal(yamlFile, &kindType)
		if err != nil {
			log.Printf("Could not parse %s as yaml: %s\n", file.Name(), err)
			continue
		}

		if kindType.Kind == "PolicyGenTemplate" {
			policyGenTemp := utils.PolicyGenTemplate{}
			err := yaml.Unmarshal(yamlFile, &policyGenTemp)
			if err != nil {
				log.Printf("Could not unmarshal PolicyGenTemplate data from %s: %s", file.Name(), err)
				continue
			}
			// overwrite template setting with optional command line argument
			if !wrapInPolicy {
				policyGenTemp.Spec.WrapInPolicy = wrapInPolicy
			}

			pBuilder := policyGen.NewPolicyBuilder(fHandler)
			policies, err := pBuilder.Build(policyGenTemp)
			if err != nil {
				log.Printf("Could not build the entire policy defined by %s: %s", file.Name(), err)
				// The error will be raised after we write out whatever policy we can
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
							log.Printf("Error marshalling yaml for %s[%d]: %s\n", k, i, pErr)
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
						log.Printf("Error marshalling yaml for %s: %s", k, pErr)
					}
				}
				if pErr == nil {
					if stdout {
						strPolicy := string(policy)
						if !strings.HasPrefix(strPolicy, "---\n") {
							fmt.Println("---")
						}
						fmt.Println(strPolicy)
					}
					fHandler.WriteFile(k+utils.FileExt, policy)
				}
			}
			if err != nil {
				panic(err)
			}
		} else {
			log.Printf("Unsupported yaml file kind in %s: %s", file.Name(), kindType)
		}
	}
}
