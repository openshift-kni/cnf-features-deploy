package utils

const ExistOper = "Exists"
const InOper = "In"
const CustomResource = "customResource"
const ACMPolicyTemplate = "acm-policy-template.yaml"
const ResourcesDir = "resources"
const FileExt = ".yaml"

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
	BindingRules map[string]string `yaml:"bindingRules,omitempty"`
	Mcp          string            `yaml:"mcp,omitempty"`
	SourceFiles  []SourceFile      `yaml:"sourceFiles,omitempty"`
}

type SourceFile struct {
	FileName   string                 `yaml:"fileName"`
	PolicyName string                 `yaml:"policyName,omitempty"`
	Metadata   MetaData               `yaml:"metadata,omitempty"`
	Spec       map[string]interface{} `yaml:"spec,omitempty"`
	Data       map[string]interface{} `yaml:"data,omitempty"`
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
