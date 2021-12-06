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
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var ptpconfiglog = logf.Log.WithName("ptpconfig-resource")

func (r *PtpConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

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
				section.options[line[:split]] = line[split+1:]
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

	// When validating, add ptp4lopts to conf for fields we check
	opts := strings.Split(*ptp4lopts, " ")
	for index, opt := range opts {
		if opt == "--summary_interval" && index < len(opts)-1 {
			output.sections["[global]"].options["summary_interval"] = opts[index+1]
		}
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

		// Validate that summary_interval matches logSyncInterval
		summary_interval := "0"
		logSyncInterval := "0"
		for option, value := range conf.sections["[global]"].options {
			if option == "summary_interval" {
				summary_interval = value
			}
			if option == "logSyncInterval" {
				logSyncInterval = value
			}
		}
		if summary_interval != logSyncInterval {
			return errors.New("summary_interval " + summary_interval + " must match logSyncInterval " + logSyncInterval)
		}

		if profile.PtpSchedulingPolicy != nil && *profile.PtpSchedulingPolicy == "SCHED_FIFO" {
			if profile.PtpSchedulingPriority == nil {
				return errors.New("PtpSchedulingPriority must be set for SCHED_FIFO PtpSchedulingPolicy")
			}
		}
	}
	return nil
}

var _ webhook.Validator = &PtpConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PtpConfig) ValidateCreate() error {
	ptpconfiglog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PtpConfig) ValidateUpdate(old runtime.Object) error {
	ptpconfiglog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PtpConfig) ValidateDelete() error {
	ptpconfiglog.Info("validate delete", "name", r.Name)
	return nil
}
