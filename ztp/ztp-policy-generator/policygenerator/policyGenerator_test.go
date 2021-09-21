package main

import (
	"github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/testSource"
	"os"
	"testing"
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

/* Section Test Functions Ends */

/* Section Test Trigger Functions Starts */
func generateACMResourceDefinitions(t *testing.T) {
	InitiatePolicyGen(testSource.GetTemplatePath(t), testSource.GetSourcePolicyPath(t), testSource.GetOutPath(t), true)
}

func generateCustomResourceDefinitions(t *testing.T) {
	InitiatePolicyGen(testSource.GetTemplatePath(t), testSource.GetSourcePolicyPath(t), testSource.GetOutPath(t), true)
}

/* Section Test Trigger Functions Ends */
