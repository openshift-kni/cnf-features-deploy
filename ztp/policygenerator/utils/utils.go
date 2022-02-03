package utils

const ExistOper = "Exists"
const InOper = "In"
const DoesNotExistOper = "DoesNotExist"
const NotInOper = "NotIn"
const CustomResource = "customResource"
const SourceCRsPath = "source-crs"
const FileExt = ".yaml"
const UnsetStringValue = "__unset_value__"
const ZtpDeployWaveAnnotation = "ran.openshift.io/ztp-deploy-wave"

// ComplianceType of "mustonlyhave" uses significant CPU to enforce. Default to
// "musthave" so that we realize the CPU reductions unless explicitly told otherwise
const DefaultComplianceType = "musthave"

type KindType struct {
	Kind string `yaml:"kind"`
}

type PolicyGenTemplate struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   MetaData          `yaml:"metadata"`
	Spec       PolicyGenTempSpec `yaml:"spec"`
}

type MetaData struct {
	Annotations map[string]string `yaml:"annotations,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace,omitempty"`
}

type PolicyGenTempSpec struct {
	BindingRules         map[string]string `yaml:"bindingRules,omitempty"`
	BindingExcludedRules map[string]string `yaml:"bindingExcludedRules,omitempty"`
	Mcp                  string            `yaml:"mcp,omitempty"`
	WrapInPolicy         bool              `yaml:"wrapInPolicy,omitempty"`
	RemediationAction    string            `yaml:"remediationAction,omitempty"`
	ComplianceType       string            `yaml:"complianceType,omitempty"`
	SourceFiles          []SourceFile      `yaml:"sourceFiles,omitempty"`
}

func (pgt *PolicyGenTempSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type PolicyGenTemplateSpec PolicyGenTempSpec
	var defaults = PolicyGenTemplateSpec{
		WrapInPolicy:      true,     //Generate ACM wrapped policies by default
		RemediationAction: "inform", // Generate inform policies by default
		ComplianceType:    DefaultComplianceType,
	}

	out := defaults
	err := unmarshal(&out)
	*pgt = PolicyGenTempSpec(out)
	return err
}

type SourceFile struct {
	FileName          string                 `yaml:"fileName"`
	PolicyName        string                 `yaml:"policyName,omitempty"`
	ComplianceType    string                 `yaml:"complianceType,omitempty"`
	RemediationAction string                 `yaml:"remediationAction,omitempty"`
	Metadata          MetaData               `yaml:"metadata,omitempty"`
	Spec              map[string]interface{} `yaml:"spec,omitempty"`
	Data              map[string]interface{} `yaml:"data,omitempty"`
}

// Provide custom YAML unmarshal for SourceFile which provides default values
func (rv *SourceFile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type Defaulted SourceFile
	var defaults = Defaulted{
		ComplianceType:    UnsetStringValue,
		RemediationAction: UnsetStringValue,
	}

	out := defaults
	err := unmarshal(&out)
	*rv = SourceFile(out)
	return err
}

type AcmPolicy struct {
	ApiVersion string        `yaml:"apiVersion"`
	Kind       string        `yaml:"kind"`
	Metadata   MetaData      `yaml:"metadata"`
	Spec       acmPolicySpec `yaml:"spec"`
}

type acmPolicySpec struct {
	RemediationAction string                   `yaml:"remediationAction"`
	Disabled          bool                     `yaml:"disabled"`
	PolicyTemplates   []PolicyObjectDefinition `yaml:"policy-templates"`
}

type PolicyObjectDefinition struct {
	ObjDef AcmConfigurationPolicy `yaml:"objectDefinition"`
}

type AcmConfigurationPolicy struct {
	ApiVersion string              `yaml:"apiVersion"`
	Kind       string              `yaml:"kind"`
	Metadata   MetaData            `yaml:"metadata"`
	Spec       acmConfigPolicySpec `yaml:"spec"`
}

type acmConfigPolicySpec struct {
	RemediationAction string `yaml:"remediationAction"`
	Severity          string `yaml:"severity"`
	NamespaceSelector struct {
		Exclude []string `yaml:"exclude"`
		Include []string `yaml:"include"`
	}
	ObjectTemplates []ObjectTemplates `yaml:"object-templates"`
}

type ObjectTemplates struct {
	ComplianceType   string                 `yaml:"complianceType"`
	ObjectDefinition map[string]interface{} `yaml:"objectDefinition"`
}

type PlacementBinding struct {
	ApiVersion   string    `yaml:"apiVersion"`
	Kind         string    `yaml:"kind"`
	Metadata     MetaData  `yaml:"metadata"`
	PlacementRef Subject   `yaml:"placementRef"`
	Subjects     []Subject `yaml:"subjects"`
}

type Subject struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	ApiGroup string `yaml:"apiGroup"`
}

type PlacementRule struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   MetaData `yaml:"metadata"`
	Spec       struct {
		ClusterSelector ClusterSelector `yaml:"clusterSelector"`
	}
}

type ClusterSelector struct {
	MatchExpressions []map[string]interface{} `yaml:"matchExpressions"`
}
