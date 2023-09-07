/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PtpRole int

const (
	Master PtpRole = 1
	Slave  PtpRole = 0
)

// log is for logging in this package.
var ptpconfiglog = logf.Log.WithName("ptpconfig-resource")

func (r *PtpConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-ptp-openshift-io-v1-ptpconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=ptp.openshift.io,resources=ptpconfigs,verbs=create;update,versions=v1,name=vptpconfig.kb.io,admissionReviewVersions=v1

type ptp4lConfSection struct {
	options map[string]string
}

type ptp4lConf struct {
	sections map[string]ptp4lConfSection
}

func (output *ptp4lConf) populatePtp4lConf(config *string, ptp4lopts *string) error {
	var string_config string
	if config != nil {
		string_config = *config
	}
	lines := strings.Split(string_config, "\n")
	var currentSection string
	output.sections = make(map[string]ptp4lConfSection)

	for _, line := range lines {
		if strings.HasPrefix(line, "[") {
			currentSection = line
			currentLine := strings.Split(line, "]")

			if len(currentLine) < 2 {
				return errors.New("Section missing closing ']'")
			}

			currentSection = fmt.Sprintf("%s]", currentLine[0])
			section := ptp4lConfSection{options: map[string]string{}}
			output.sections[currentSection] = section
		} else if currentSection != "" {
			split := strings.IndexByte(line, ' ')
			if split > 0 {
				section := output.sections[currentSection]
				section.options[line[:split]] = strings.TrimSpace(line[split+1:])
				output.sections[currentSection] = section
			}
		} else {
			return errors.New("Config option not in section")
		}
	}
	_, exist := output.sections["[global]"]
	if !exist {
		output.sections["[global]"] = ptp4lConfSection{options: map[string]string{}}
	}

	return nil
}

func (r *PtpConfig) validate() error {
	profiles := r.Spec.Profile
	for _, profile := range profiles {
		conf := &ptp4lConf{}
		conf.populatePtp4lConf(profile.Ptp4lConf, profile.Ptp4lOpts)

		// Validate that interface field only set in ordinary clock
		if profile.Interface != nil && *profile.Interface != "" {
			for section := range conf.sections {
				if section != "[global]" {
					if section != ("[" + *profile.Interface + "]") {
						return errors.New("interface section " + section + " not allowed when specifying interface section")
					}
				}
			}
		}

		if profile.PtpSchedulingPolicy != nil && *profile.PtpSchedulingPolicy == "SCHED_FIFO" {
			if profile.PtpSchedulingPriority == nil {
				return errors.New("PtpSchedulingPriority must be set for SCHED_FIFO PtpSchedulingPolicy")
			}
		}

		if profile.PtpSettings != nil {
			for k, v := range profile.PtpSettings {
				switch {
				case k == "stdoutFilter":
					_, err := regexp.Compile(v)
					if err != nil {
						return errors.New("stdoutFilter='" + v + "' is invalid; " + err.Error())
					}
				case k == "logReduce":
					v = strings.ToLower(v)
					if v != "true" && v != "false" {
						return errors.New("logReduce='" + v + "' is invalid; must be in 'true' or 'false'")
					}
				default:
					return errors.New("profile.PtpSettings '" + k + "' is not a configurable setting")
				}
			}
		}
	}
	return nil
}

var _ webhook.Validator = &PtpConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PtpConfig) ValidateCreate() (admission.Warnings, error) {
	ptpconfiglog.Info("validate create", "name", r.Name)
	if err := r.validate(); err != nil {
		return admission.Warnings{}, err
	}

	return admission.Warnings{}, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PtpConfig) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	ptpconfiglog.Info("validate update", "name", r.Name)
	if err := r.validate(); err != nil {
		return admission.Warnings{}, err
	}

	return admission.Warnings{}, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PtpConfig) ValidateDelete() (admission.Warnings, error) {
	ptpconfiglog.Info("validate delete", "name", r.Name)
	return admission.Warnings{}, nil
}

func getInterfaces(input *ptp4lConf, mode PtpRole) (interfaces []string) {

	for index, section := range input.sections {
		sectionName := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(index, "[", ""), "]", ""))
		if strings.TrimSpace(section.options["masterOnly"]) == strconv.Itoa(int(mode)) {
			interfaces = append(interfaces, strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(sectionName, "[", ""), "]", "")))
		}
	}
	return interfaces
}

func GetInterfaces(config PtpConfig, mode PtpRole) (interfaces []string) {

	if len(config.Spec.Profile) > 1 {
		logrus.Warnf("More than one profile detected for ptpconfig %s", config.ObjectMeta.Name)
	}
	if len(config.Spec.Profile) == 0 {
		logrus.Warnf("No profile detected for ptpconfig %s", config.ObjectMeta.Name)
		return interfaces
	}
	conf := &ptp4lConf{}
	var dummy *string
	err := conf.populatePtp4lConf(config.Spec.Profile[0].Ptp4lConf, dummy)
	if err != nil {
		logrus.Warnf("ptp4l conf parsing failed, err=%s", err)
	}

	interfaces = getInterfaces(conf, mode)
	var finalInterfaces []string
	for _, aIf := range interfaces {
		if aIf == "global" {
			if config.Spec.Profile[0].Interface != nil {
				finalInterfaces = append(finalInterfaces, *config.Spec.Profile[0].Interface)
			}
		} else {
			finalInterfaces = append(finalInterfaces, aIf)
		}
	}
	if len(interfaces) == 0 && mode == Slave && config.Spec.Profile[0].Interface != nil {
		finalInterfaces = append(finalInterfaces, *config.Spec.Profile[0].Interface)
	}
	return finalInterfaces
}
