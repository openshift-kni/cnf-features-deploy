package main

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	sriov "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	performanceprofile "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v1"
	tuned "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	ptp "github.com/openshift/ptp-operator/api/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"log"
	configurationPolicyv1 "open-cluster-management.io/config-policy-controller/api/v1"
	policyv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	"os"
	"path"
	"runtime"
	"sigs.k8s.io/yaml"
	"strings"
)

var (
	sch    = k8sruntime.NewScheme()
	_      = configurationPolicyv1.AddToScheme(sch)
	_      = policyv1.AddToScheme(sch)
	decode = serializer.NewCodecFactory(sch).UniversalDeserializer().Decode
)

type CR struct {
	u        unstructured.Unstructured
	filePath string
	hasMatch bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	setA := make(map[string][]CR)

	setB := make(map[string][]CR)

	/*
		process set A
	*/
	populateMap(setA, getenv("A_PATH", "./example/acmpolicy"))
	/*	log.Println("wasn't able to reduce to 1 CR for the following")
		for key, val := range setA {
			if len(val) > 1 {
				log.Println(key)
			}
		}
		log.Printf("done A CRs ------->>")*/

	/*
		process set B
	*/
	populateMap(setB, getenv("B_PATH", "./example/ztppolicy"))
	/*	log.Println("wasn't able to reduce to 1 CR for the following")
		for key, val := range setB {
			if len(val) > 1 {
				log.Println(key)
			}
		}
		log.Printf("done B CRs ------->>")*/

	/*
		Diff set A and set B
	*/
	var globalthereIsDiff bool
	for name, crs := range setA {
		// key into set B
		curToCR, ok := setB[name]
		if ok {
			if len(crs) == 1 {
				// can be diffed without any issues
				thereIsDiff, diffString := diffCRs(crs[0].u, curToCR[0].u)
				if thereIsDiff {
					log.Printf("files to checkout: %s<---->%s\n", crs[0].filePath, curToCR[0].filePath)
					printDiff(diffString)
					globalthereIsDiff = true
				}
			} else {
				// attempt to diff against all the CRs against that whatever is in
				sawLastTime := make(map[string]bool)
				var (
					isDiff     bool
					diffString string
				)
				for _, setACr := range crs {
					// keep an internal count to change depending on hit or miss
					var diffCount int
					for _, setBCr := range setB[name] {
						// if true that means there's a diff but see the next one for a match
						/*
							e.g
							setA : [cr1, cr2, cr3]
							setB : [cr4, cr5, cr6]
							cr1 matches with cr5 ...good. now check if cr2 and cr3 matches with any of setB CRs
						*/
						isDiff, diffString = diffCRs(setACr.u, setBCr.u)
						if !isDiff {
							diffCount = 0
							break
						} else {
							// did you see this diff last time? count just one time for a diff...
							_, saw := sawLastTime[diffString]
							if !saw {
								sawLastTime[diffString] = true
								diffCount += 1
							}
						}
					}
					if diffCount != 0 {
						printDiff(diffString)
						globalthereIsDiff = true
						log.Printf("may want to check out what's going here...multiple CRs of the same kind %s\n", setACr.filePath)
					}
				}
			}
			delete(setB, name)
			delete(setA, name)
		}
	}
	if len(setB) > 0 {
		log.Printf("CRs coming from SetB that's not in SetA")
		for key, val := range setB {
			log.Println(key)
			printCR(val[0].u)
		}
	}

	if len(setA) > 0 {
		log.Printf("CRs coming from SetA that's not in SetB")
		for key, val := range setA {
			log.Println(key)
			printCR(val[0].u)
		}
	}

	if !globalthereIsDiff && len(setB) == 0 && len(setA) == 0 {
		log.Println("match")
		return nil
	}

	return fmt.Errorf("no match")
}

func populateMap(record map[string][]CR, path string) {
	fromFilePaths := getFilePaths(path) // cli
	for _, filePath := range fromFilePaths {
		crs := extractCR(filePath)
		for _, cr := range crs {
			name := getUniqueName(cr)
			if name != "" {
				_, recordedAlready := record[name]
				if !recordedAlready {
					record[name] = []CR{
						{
							u:        cr,
							filePath: filePath,
						},
					}
				} else {
					// here...that means this CR was previously recorded. Seeing again maybe because different cluster type CRs mixed together (SNO+3Node+Standard)
					// try another diff between new and original. If it's same ignore the new one (?)
					alreadyThere := record[name]
					isDiff, _ := diffCRs(alreadyThere[0].u, cr)
					if isDiff {
						/*						printDiff(diffS)
																		printCR(alreadyThere)
																		printCR(cr)
												log.Printf("check files %s and %s", recordFileName[name], filePath)
						*/
						sl := record[name]
						sl = append(sl, CR{
							u:        cr,
							filePath: filePath,
						})
						record[name] = sl
					}
				}
			}
		}
	}
}

func printCR(cr unstructured.Unstructured) {
	f, _ := cr.MarshalJSON()
	fY, _ := yaml.JSONToYAML(f)
	log.Printf("\n%s", string(fY))
}

func diffCRs(prev unstructured.Unstructured, cur unstructured.Unstructured) (bool, string) {
	/*
		TODO: use custom reporter instead
				var r DiffReporter
				cmp.Equal(prev.Object, cur.Object, cmp.Reporter(&r))
			    fmt.Print(r.String())
	*/

	if diff := cmp.Diff(prev, cur); diff != "" {
		return true, diff
	}

	return false, ""
}

func printDiff(diff string) {
	log.Println("cDiff starts-------------------->")
	log.Printf("unexpected diff (-want +got):\n%s", diff)
	log.Println("<---------------Diff ends--------------------->\n\n")
}

func getString(u unstructured.Unstructured) string {
	marshalJSON, err := u.MarshalJSON()
	if err != nil {
		return ""
	}
	toYAML, err := yaml.JSONToYAML(marshalJSON)
	if err != nil {
		return ""
	}
	return string(toYAML)
}

func getFilePaths(dirOrFilePath string) []string {
	var filePaths []string
	if isDirectory(dirOrFilePath) {
		filePaths, err := readDir(dirOrFilePath, filePaths)
		if err != nil {
			return nil
		}
		return filePaths
	}

	filePaths = append(filePaths, dirOrFilePath)
	return filePaths
}

func extractCR(filePath string) []unstructured.Unstructured {
	curYaml, _ := getCR(filePath)
	policies := getPolicy(curYaml)
	if policies == nil {
		return []unstructured.Unstructured{}
	}
	configurationPolicy := getConfigurationPolicy(policies)

	var allCRs []unstructured.Unstructured
	for _, cP := range configurationPolicy {
		crs := getObjectDefinitionFromConfigurationPolicy(cP)
		allCRs = append(allCRs, crs...)
	}

	return allCRs
}

func getUniqueName(cr unstructured.Unstructured) string {
	var (
		ns = cr.GetNamespace()
	)
	if ns == "" {
		ns = "MISSING-NS"
	}
	return fmt.Sprintf("%s-%s-%s-%s", cr.GetKind(), cr.GetAPIVersion(), ns, cr.GetName())
}

func getPolicy(fileR []byte) []policyv1.Policy {
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	retVal := make([]policyv1.Policy, 0, len(sepYamlfiles))

	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}
		curP := policyv1.Policy{}
		_, _, err := decode([]byte(f), nil, &curP)
		if err != nil || curP.Kind != "Policy" {
			//log.Println(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
			return nil
		}

		retVal = append(retVal, curP)
	}
	return retVal
}

func getObjectDefinitionFromConfigurationPolicy(configurationPolicy configurationPolicyv1.ConfigurationPolicy) []unstructured.Unstructured {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	var uStructObj []unstructured.Unstructured

	for _, objectTemplate := range configurationPolicy.Spec.ObjectTemplates {
		// here we dynamic...though can be typed too.

		var obj k8sruntime.Object
		var scope conversion.Scope
		k8sruntime.Convert_runtime_RawExtension_To_runtime_Object(&objectTemplate.ObjectDefinition, &obj, scope)
		innerObj, _ := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(obj)
		u := unstructured.Unstructured{Object: innerObj}

		// specific to a CR type
		if u.GroupVersionKind().Kind == "PtpConfig" {
			u = patchCR(*objectTemplate, path.Join(path.Dir(filename), "./override/PtpConfig.yaml"), ptp.PtpConfig{})
		}

		if u.GroupVersionKind().Kind == "SriovOperatorConfig" {
			u = patchCR(*objectTemplate, path.Join(path.Dir(filename), "./override/SriovOperatorConfig.yaml"), sriov.SriovOperatorConfig{})
		}

		if u.GroupVersionKind().Kind == "PerformanceProfile" {
			u = patchCR(*objectTemplate, path.Join(path.Dir(filename), "./override/PerformanceProfile.yaml"), performanceprofile.PerformanceProfile{})
		}

		if u.GroupVersionKind().Kind == "Tuned" {
			u = patchCR(*objectTemplate, path.Join(path.Dir(filename), "./override/Tuned.yaml"), tuned.Tuned{})
		}

		if u.GroupVersionKind().Kind == "SriovNetworkNodePolicy" {
			u = patchCR(*objectTemplate, path.Join(path.Dir(filename), "./override/SriovNetworkNodePolicy.yaml"), sriov.SriovNetworkNodePolicy{})
		}

		// common clean cleanup across of CRs
		u = removeAnnotation(u, "ran.openshift.io/ztp-deploy-wave")

		uStructObj = append(uStructObj, u)
	}

	return uStructObj
}

func patchCR(objectTemplate configurationPolicyv1.ObjectTemplate, crPath string, myI interface{}) unstructured.Unstructured {
	original, _ := objectTemplate.ObjectDefinition.MarshalJSON()
	target, _ := getCR(crPath)

	//safely convert to json, no op if json and generate the update CR
	target, _ = yaml.YAMLToJSON(target)

	patch, _ := strategicpatch.StrategicMergePatch(original, target, myI)

	//convert back to unstructured json -> ustruct
	result := make(map[string]interface{})
	json.Unmarshal(patch, &result)

	innerObj, _ := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(&result)

	return unstructured.Unstructured{Object: innerObj}
}

func removeAnnotation(u unstructured.Unstructured, key string) unstructured.Unstructured {
	anno := u.GetAnnotations()
	delete(anno, key)
	u.SetAnnotations(anno)

	if len(u.GetAnnotations()) == 0 {
		u.SetAnnotations(nil)
	}

	return u
}

func getConfigurationPolicy(o []policyv1.Policy) []configurationPolicyv1.ConfigurationPolicy {
	var cPs []configurationPolicyv1.ConfigurationPolicy
	for _, p := range o {
		for _, policyTemplate := range p.Spec.PolicyTemplates {
			var configurationPolicy configurationPolicyv1.ConfigurationPolicy
			policyTemplateBytes, _ := policyTemplate.ObjectDefinition.MarshalJSON()
			_, _, err := decode(policyTemplateBytes, nil, &configurationPolicy)
			//log.Printf(b.String())
			if err != nil {
				//log.Println(fmt.Sprintf("Error while decoding configurationPolicy. Err was: %s", err))
				return []configurationPolicyv1.ConfigurationPolicy{}
			}
			//log.Printf("now processing configurationPolicy: %s", configurationPolicy.Name)
			cPs = append(cPs, configurationPolicy)
		}
	}

	return cPs
}

// isDirectory determines if a file represented
// by `path` is a directory or not
func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}

func readDir(path string, filePaths []string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return filePaths, err
	}
	defer file.Close()
	names, _ := file.Readdirnames(0)
	for _, name := range names {
		filePath := fmt.Sprintf("%v/%v", path, name)
		filePaths = append(filePaths, filePath)
		file, err := os.Open(filePath)
		if err != nil {
			return filePaths, err
		}
		defer file.Close()
		fileInfo, err := file.Stat()
		if err != nil {
			return filePaths, err
		}
		if fileInfo.IsDir() {
			readDir(filePath, filePaths)
		}
	}
	return filePaths, nil
}

func getCR(path string) ([]byte, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
		return nil, err
	}

	return body, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
