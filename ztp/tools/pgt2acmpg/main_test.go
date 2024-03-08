package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/acmformat"
	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/pgtformat"
	"gopkg.in/yaml.v3"
)

func Test_processFlags(t *testing.T) {
	inputFileFlag := "dummy.yaml"
	outputDirFlag := "dummy"
	preRenderPatchKindString1 := "kind1,kind2,kind3"
	sourceCRListString1 := "/tmp/source-cr1,/home/source-cr2"
	preRenderPatchKindString2 := "kind1,kind2"
	sourceCRListString2 := "/tmp,/home/source-cr2"

	type args struct {
		inputFile                *string
		outputDir                *string
		preRenderPatchKindString *string
		sourceCRListString       *string
	}
	tests := []struct {
		name                       string
		args                       args
		wantPreRenderPatchKindList []string
		wantPreRenderSourceCRList  []string
	}{
		{
			name:                       "ok",
			args:                       args{inputFile: &inputFileFlag, outputDir: &outputDirFlag, preRenderPatchKindString: &preRenderPatchKindString1, sourceCRListString: &sourceCRListString1},
			wantPreRenderPatchKindList: []string{"kind1", "kind2", "kind3"},
			wantPreRenderSourceCRList:  []string{"/tmp/source-cr1", "/home/source-cr2"},
		},
		{
			name:                       "nok",
			args:                       args{inputFile: &inputFileFlag, outputDir: &outputDirFlag, preRenderPatchKindString: &preRenderPatchKindString2, sourceCRListString: &sourceCRListString2},
			wantPreRenderPatchKindList: []string{"kind1", "kind2"},
			wantPreRenderSourceCRList:  []string{"/tmp", "/home/source-cr2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPreRenderPatchKindList, gotPreRenderSourceCRList := processFlags(tt.args.inputFile, tt.args.outputDir, tt.args.preRenderPatchKindString, tt.args.sourceCRListString)
			if !reflect.DeepEqual(gotPreRenderPatchKindList, tt.wantPreRenderPatchKindList) {
				t.Errorf("processFlags() gotPreRenderPatchKindList = %v, want %v", gotPreRenderPatchKindList, tt.wantPreRenderPatchKindList)
			}
			if !reflect.DeepEqual(gotPreRenderSourceCRList, tt.wantPreRenderSourceCRList) {
				t.Errorf("processFlags() gotPreRenderSourceCRList = %v, want %v", gotPreRenderSourceCRList, tt.wantPreRenderSourceCRList)
			}
		})
	}
}

func loadTestPGT() (policyGenTemp pgtformat.PolicyGenTemplate, err error) {
	policyGenFileContent := `---
    apiVersion: ran.openshift.io/v1
    kind: PolicyGenTemplate
    metadata:
      name: "group-du-standard-latest"
      namespace: "ztp-group"
    spec:
      bindingRules:
        # These policies will correspond to all clusters with this label:
        group-du-standard: ""
        du-profile: "latest"
      mcp: "worker"
      sourceFiles:
        - fileName: PtpOperatorConfig.yaml
          policyName: "config-policy"
        - fileName: PtpConfigSlave.yaml   # Change to PtpConfigSlaveCvl.yaml for ColumbiaVille NIC
          policyName: "config-policy"
          metadata:
            name: "du-ptp-slave"
          spec:
            profile:
            - name: "slave"
              # This interface must match the hardware in this group
              interface: "ens5f0"
              ptp4lOpts: "-2 -s --summary_interval -4"
              phc2sysOpts: "-a -r -n 24"
        - fileName: SriovOperatorConfig.yaml
          policyName: "config-policy"
        - fileName: PerformanceProfile.yaml
          policyName: "config-policy"
          spec:
            cpu:
              # These must be tailored for the specific hardware platform
              isolated: "2-19,22-39"
              reserved: "0-1,20-21"
            hugepages:
              defaultHugepagesSize: 1G
              pages:
                - size: 1G
                  count: 32
        - fileName: TunedPerformancePatch.yaml
          policyName: "config-policy"
        #
        # These CRs are to enable crun on master and worker nodes for 4.13+ only
        #
        # Include these CRs in the group PGT instead of the common PGT to make sure
        # they are applied after the operators have been successfully installed,
        # however, it's strongly recommended to include these CRs as day-0 extra manifests
        # to avoid an extra reboot of the master nodes.
        - fileName: optional-extra-manifest/enable-crun-master.yaml
          policyName: "config-policy"
        - fileName: optional-extra-manifest/enable-crun-worker.yaml
          policyName: "config-policy"
    `
	policyGenTemp = pgtformat.PolicyGenTemplate{}

	err = yaml.Unmarshal([]byte(policyGenFileContent), &policyGenTemp)
	if err != nil {
		return policyGenTemp, fmt.Errorf("could not unmarshal PolicyGenTemplate data: %s", err)
	}
	return policyGenTemp, err
}

func Test_convertPGTPolicyToACMPGPolicy(t *testing.T) {
	pgt, err := loadTestPGT()
	if err != nil {
		t.Fatalf("failed to load PGT, err: %s", err)
		return
	}
	acmPolicy := acmformat.PolicyConfig{
		Name: "-config-policy",
		PolicyOptions: acmformat.PolicyOptions{
			PolicyAnnotations: map[string]string{
				"ran.openshift.io/ztp-deploy-wave": "1",
			},
		},
		Manifests: []acmformat.Manifest{
			{
				Path: "source-crs/PtpOperatorConfig.yaml",
			},
			{
				Path: "source-crs/PtpConfigSlave.yaml",
				Patches: []map[string]interface{}{
					{
						"metadata": map[string]interface{}{
							"name": "du-ptp-slave",
						},
						"spec": map[string]interface{}{
							"profile": []interface{}{
								map[string]interface{}{
									"interface":   "ens5f0",
									"name":        "slave",
									"phc2sysOpts": "-a -r -n 24",
									"ptp4lOpts":   "-2 -s --summary_interval -4",
								},
							},
						},
					},
				},
				ExtraDependencies: nil,
				IgnorePending:     false,
			},
			{
				Path: "source-crs/SriovOperatorConfig.yaml",
			},
			{
				Path: "source-crs/PerformanceProfile.yaml",
				Patches: []map[string]interface{}{
					{
						"spec": map[string]interface{}{
							"cpu": map[string]interface{}{
								"isolated": "2-19,22-39",
								"reserved": "0-1,20-21",
							},
							"hugepages": map[string]interface{}{
								"defaultHugepagesSize": "1G",
								"pages": []interface{}{
									map[string]interface{}{
										"count": 32,
										"size":  "1G",
									},
								},
							},
						},
					},
				},
				ExtraDependencies: nil,
				IgnorePending:     false,
			},
			{
				Path: "source-crs/TunedPerformancePatch.yaml",
			},
			{
				Path: "source-crs/optional-extra-manifest/enable-crun-master.yaml",
			},
			{
				Path: "source-crs/optional-extra-manifest/enable-crun-worker.yaml",
			},
		},
	}

	type args struct {
		policyGenTemp *pgtformat.PolicyGenTemplate
		rootName      string
		policyName    string
		outputDir     string
		forceWave     string
	}
	tests := []struct {
		name          string
		args          args
		wantNewPolicy acmformat.PolicyConfig
		wantErr       bool
	}{
		{
			name:          "ok",
			args:          args{policyGenTemp: &pgt, rootName: "", policyName: "config-policy", outputDir: "tmp", forceWave: "1"},
			wantNewPolicy: acmPolicy,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewPolicy, err := convertPGTPolicyToACMPGPolicy(tt.args.policyGenTemp, "", "", tt.args.rootName, tt.args.policyName, tt.args.outputDir, tt.args.forceWave)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertPGTPolicyToACMPGPolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotNewPolicy, tt.wantNewPolicy) {
				t.Errorf("convertPGTPolicyToACMPGPolicy() = %v, want %v", gotNewPolicy, tt.wantNewPolicy)
			}
		})
	}
}
