# Policy Generator Unit Tests
We use golang built in testing framework along with testify to test the policy generator.
We are not adding unit tests per method rather executing different code paths through different inputs to the policyGenerator. 

The current input parameters to the PolicyGenerator are:
1. ___policyGenTempPath___:   Path to the Policy Generator Templates directory.
2. ___sourcePoliciesPath___:  Path to the source Policy CRs
3. ___outPath___: Path to the output directory where the generated CRs are maintained.
4. ___stdout___:  Boolean to allow or disallow the generated specs to be piped to stdout.
5. ___customResources___: Boolean to generate ACM compatible CRs or CRs that can be applied directly using Openshift.

Every test currently runs the policy generator twice for both the ACM compatible CRs and CRs to run directly with Openshift. The framework currently supports only 1 template CR and 1 source policy CR per test.

The next section goes over details of how you can add a new test.

###   Adding a New Test:

 Steps to add new Unit Test
1. Add a new method under `policyGenerator_test.go`. For convention use _Test\<CR Kind\>_ as the method name.
   Refer to the _TestExample_ method for the base test run and assertions.
2. Create a new directory under _testFiles_ directory with the same name as the method name from step 1.
3. Create 2 directories under the newly created test directory named _sourcePolicies_ and _templates_
4. Add the source Policy CRD under the _sourcePolicies_  directory. Rename the source Policy CRD file as `<TestMethodName>.yaml`.
5. Add the template CRD under the _templates_ directory. Rename the source Policy CRD file as `<TestMethodName>.yaml`.
6. Make sure you update the `sourceFiles -> fileName` in the template to match the source Policy CR file name 
7. You can add any custom assertions to the test, but would be better if we can add more general assertions which can be reused in other test cases going forward. Adding a new assertion will be covered in the next section.  

Directory hierarchy for testFiles
```
testFiles
├── <TestName1>
│   ├── out (created during test run)
│   |   ├── <GeneratedResources>
│   ├── sourcePolices
│   |   ├── <TestName1>.yaml 
│   ├── templates
│   |   ├── <TestName1>.yaml 
|
├── <TestName2>
│   ├── sourcePolices
│   |   ├── <TestName2>.yaml 
│   ├── templates
│   |   ├── <TestName2>.yaml 
```
### Assertions supported in every test:

For the ACM compatible resource generation run:

* Verify that directory for generated ACM compatible resource exists.
* Verify that the expected file for the generated ACM compatible resource exists and is named appropriately.
* Verify that the placementBinding and placementRule files exist and are named appropriately.
* Verify that the generated file contains the source CR in the spec object definition.
* Verify field substitution added using template.

For the direct resource generation run:

* Verify that directory for generated direct resource exists.
* Verify that the expected file for the generated resource exists and is named appropriately.
* Verify that the generated file is identical to the source CRD in case of no changes within the template.
* Verify field substitution added using template.

###   Adding a new Assertion:
Considerations when adding a new Assertion:
* Using/Adding generic getters that leverage the _testMethodName_ to resolve the file/path. 
* Add any new getters to the testing code at `TestGetters.go`. 
* Make sure these getters are generic and can be reused by other tests.
* Using the `utils.go` file within PolicyGen for structs defined to cast yaml files into objects.
* Document the custom assertions added at the start of any new tests added.
* Add new methods for any new assertions within `TestAssertions.go`
* In case of any generic helper methods are needed to compute the assertions, please add these methods under `TestHelpers.go`


###   Run Tests & View Coverage :

To run tests and view coverage run the following commands from within this directory.

- Run single unit test


      $ go test -run <TestMethodName>

- Run all unit tests and verify coverage.


    $  go test -cover -coverpkg ./... -coverprofile cover.out

- View Coverage per function name by using this command.


    $  go tool cover -func cover.out 

- View HTML rendering of source files with coverage information.


    $ go tool cover -html cover.out 
