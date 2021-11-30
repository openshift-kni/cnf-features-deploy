package main

import (
	"os"
	"testing"

	"github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/testSource"
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/utils"
	"github.com/stretchr/testify/assert"
)

/* Section Test Setup Functions Starts */
func testSetup(t *testing.T) {
	os.RemoveAll(testSource.GetOutPath(t))
}

func testCleanup(t *testing.T) {
	os.RemoveAll(testSource.GetOutPath(t))
}

/* Section Test Setup Functions Ends */

/* Section Test Functions Starts */

/*func TestExample(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}*/

func TestNamespace(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	testSource.ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	testSource.CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestOperatorGroup(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	testSource.ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	testSource.CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestSubscription(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	testSource.ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	testSource.CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestMachineConfigPool(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	testSource.ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	testSource.CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestPtpConfig(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	testSource.ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	testSource.CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestSriovNetwork(t *testing.T) {
	testSetup(t)
	generateACMResourceDefinitions(t)
	testSource.ACMResourceDefinitionAssertions(t)
	generateCustomResourceDefinitions(t)
	testSource.CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

func TestUnwrappedNamespace(t *testing.T) {
	testSetup(t)
	generateCustomResourceDefinitions(t)
	testSource.CustomResourceDefinitionAssertions(t)
	testCleanup(t)
}

/* Section Test Functions Ends */

/* Section Test Trigger Functions Starts */
func generateACMResourceDefinitions(t *testing.T) {
	fHandler := utils.NewFilesHandler(testSource.GetSourcePolicyPath(t), testSource.GetTemplatePath(t), testSource.GetOutPath(t))
	policyGenTemps := make([]string, 0)
	files, err := fHandler.GetTempFiles()
	assert.Equal(t, err, nil)
	for _, file := range files {
		policyGenTemps = append(policyGenTemps, testSource.GetTemplatePath(t)+"/"+file.Name())
	}

	InitiatePolicyGen(fHandler, policyGenTemps, true)
}

func generateCustomResourceDefinitions(t *testing.T) {
	fHandler := utils.NewFilesHandler(testSource.GetSourcePolicyPath(t), testSource.GetTemplatePath(t), testSource.GetOutPath(t))
	policyGenTemps := make([]string, 0)
	files, err := fHandler.GetTempFiles()
	assert.Equal(t, err, nil)
	for _, file := range files {
		policyGenTemps = append(policyGenTemps, testSource.GetTemplatePath(t)+"/"+file.Name())
	}

	InitiatePolicyGen(fHandler, policyGenTemps, false)
}

/* Section Test Trigger Functions Ends */
