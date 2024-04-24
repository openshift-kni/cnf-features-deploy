// The content of this file mainly comes from the following source
// at https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/policygenerator/utils/utils.go

package pgtformat

const UnsetStringValue = "__unset_value__"
const DefaultComplianceType = "musthave"
const DefaultCompliantEvaluationInterval = "10m"
const DefaultNonCompliantEvaluationInterval = "10s"
const Inform = "inform"

type PolicyGenTemplate struct {
	//nolint:revive,stylecheck // keep same name as original lib
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
	BindingRules         map[string]string  `yaml:"bindingRules,omitempty"`
	BindingExcludedRules map[string]string  `yaml:"bindingExcludedRules,omitempty"`
	Mcp                  string             `yaml:"mcp,omitempty"`
	WrapInPolicy         bool               `yaml:"wrapInPolicy,omitempty"`
	RemediationAction    string             `yaml:"remediationAction,omitempty"`
	ComplianceType       string             `yaml:"complianceType,omitempty"`
	EvaluationInterval   EvaluationInterval `yaml:"evaluationInterval,omitempty"`
	SourceFiles          []SourceFile       `yaml:"sourceFiles,omitempty"`
}

// UnmarshalYAML Unmarshal YAML file as a PGT spec
func (pgt *PolicyGenTempSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type PolicyGenTemplateSpec PolicyGenTempSpec
	var defaults = PolicyGenTemplateSpec{
		WrapInPolicy:       true,   // Generate ACM wrapped policies by default
		RemediationAction:  Inform, // Generate inform policies by default
		ComplianceType:     DefaultComplianceType,
		EvaluationInterval: EvaluationInterval{DefaultCompliantEvaluationInterval, DefaultNonCompliantEvaluationInterval},
	}

	out := defaults
	err := unmarshal(&out)
	*pgt = PolicyGenTempSpec(out)
	return err
}

type EvaluationInterval struct {
	Compliant    string `yaml:"compliant,omitempty"`
	NonCompliant string `yaml:"noncompliant,omitempty"`
}

type SourceFile struct {
	FileName           string                 `yaml:"fileName"`
	PolicyName         string                 `yaml:"policyName,omitempty"`
	ComplianceType     string                 `yaml:"complianceType,omitempty"`
	RemediationAction  string                 `yaml:"remediationAction,omitempty"`
	Metadata           map[string]interface{} `yaml:"metadata,omitempty"`
	Spec               map[string]interface{} `yaml:"spec,omitempty"`
	Data               map[string]interface{} `yaml:"data,omitempty"`
	Status             map[string]interface{} `yaml:"status,omitempty"`
	BinaryData         map[string]interface{} `yaml:"binaryData,omitempty"`
	EvaluationInterval EvaluationInterval     `yaml:"evaluationInterval,omitempty"`
}

// UnmarshalYAML Provide custom YAML unmarshal for SourceFile which provides default values
func (rv *SourceFile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type Defaulted SourceFile
	var defaults = Defaulted{
		ComplianceType:     UnsetStringValue,
		RemediationAction:  UnsetStringValue,
		EvaluationInterval: EvaluationInterval{UnsetStringValue, UnsetStringValue},
	}

	out := defaults
	err := unmarshal(&out)
	*rv = SourceFile(out)
	return err
}
