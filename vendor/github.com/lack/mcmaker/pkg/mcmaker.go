package mcmaker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/lack/yamltrim"
	"github.com/vincent-petithory/dataurl"

	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const roleKey = "machineconfiguration.openshift.io/role"

// McMaker represents a MachineConfig that is being constructed from constituent parts
type McMaker struct {
	name string
	mc   *machineconfigv1.MachineConfig
	i    *ign3types.Config
}

// New creates a new McMaker with the given name
func New(name string) McMaker {
	return McMaker{
		name: name,
		mc: &machineconfigv1.MachineConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: machineconfigv1.GroupVersion.String(),
				Kind:       "MachineConfig",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: make(map[string]string),
			},
			Spec: machineconfigv1.MachineConfigSpec{},
		},
		i: &ign3types.Config{
			Ignition: ign3types.Ignition{
				Version: ign3types.MaxVersion.String(),
			},
		},
	}
}

// SetRole sets the MachineConfig object name and MCP-role selection label to the given role
func (m *McMaker) SetRole(role string) {
	m.mc.ObjectMeta.Name = fmt.Sprintf("%s-%s", m.name, role)
	m.mc.ObjectMeta.Labels[roleKey] = role
}

// AddFile adds a file to the MachineConfig object from the given local file
func (m *McMaker) AddFile(fname, path string, mode int) error {
	fdata, err := os.Open(fname)
	if err != nil {
		return err
	}
	return m.AddFileFromStream(fdata, path, mode)
}

// AddFileFromStream adds a file to the MachineConfig object from the given io.Reader
func (m *McMaker) AddFileFromStream(fdata io.Reader, path string, mode int) error {
	if path == "" {
		return fmt.Errorf("file entries require a path")
	}

	var fbytes bytes.Buffer
	io.Copy(&fbytes, fdata)
	encodedContent := dataurl.EncodeBytes(fbytes.Bytes())

	f := ign3types.File{
		Node: ign3types.Node{
			Path: path,
		},
		FileEmbedded1: ign3types.FileEmbedded1{
			Contents: ign3types.Resource{
				Source: &encodedContent,
			},
			Mode: &mode,
		},
	}
	m.i.Storage.Files = append(m.i.Storage.Files, f)
	return nil
}

// AddUnit adds a systemd unit to the MachineConfig object from the given local file
func (m *McMaker) AddUnit(fname, name string, enable bool) error {
	s, err := os.Open(fname)
	if err != nil {
		return err
	}
	if name == "" {
		name = filepath.Base(fname)
	}
	return m.AddUnitFromStream(s, name, enable)
}

// AddUnitFromStream adds a systemd unit to the MachineConfig object from the given io.Reader
func (m *McMaker) AddUnitFromStream(source io.Reader, name string, enable bool) error {
	if name == "" {
		return fmt.Errorf("unit entries require a name")
	}

	var contents bytes.Buffer
	_, err := io.Copy(&contents, source)
	if err != nil {
		return err
	}

	contentString := contents.String()

	u := ign3types.Unit{
		Name:     name,
		Contents: &contentString,
		Enabled:  &enable,
	}
	m.i.Systemd.Units, err = mergeSystemdUnits(m.i.Systemd.Units, u)
	return err
}

// AddDropin adds a systemd unit to the MachineConfig object from the given local file
func (m *McMaker) AddDropin(fname, service, name string) error {
	s, err := os.Open(fname)
	if err != nil {
		return err
	}
	if name == "" {
		name = filepath.Base(fname)
	}
	return m.AddDropinFromStream(s, service, name)
}

// AddDropinFromStream adds a systemd drop-in to the MachineConfig object from the given io.Reader
func (m *McMaker) AddDropinFromStream(source io.Reader, service, name string) error {
	if service == "" {
		return fmt.Errorf("dropin entries require a service")
	}
	if name == "" {
		return fmt.Errorf("dropin entries require a name")
	}

	var contents bytes.Buffer
	_, err := io.Copy(&contents, source)
	if err != nil {
		return err
	}

	contentString := contents.String()

	u := ign3types.Unit{
		Name:     service,
		Contents: nil,
		Dropins: []ign3types.Dropin{{
			Contents: &contentString,
			Name:     name,
		}},
	}
	m.i.Systemd.Units, err = mergeSystemdUnits(m.i.Systemd.Units, u)
	return err
}

func mergeSystemdUnits(units []ign3types.Unit, newUnit ign3types.Unit) ([]ign3types.Unit, error) {
	// If there's already a unit with this same name, we may need to append a drop-in or content to combine into one entry
	for i := range units {
		u := &units[i]
		if u.Name == newUnit.Name {
			if newUnit.Contents != nil {
				if u.Contents != nil {
					return units, fmt.Errorf("unit '%s' already has 'Contents' defined", u.Name)
				}
				u.Contents = newUnit.Contents
			}
			if newUnit.Enabled != nil {
				if u.Enabled != nil {
					return units, fmt.Errorf("unit '%s' already has 'Enabled' defined", u.Name)
				}
				u.Enabled = newUnit.Enabled
			}
			if len(newUnit.Dropins) > 0 {
				for _, d := range u.Dropins {
					for _, nd := range newUnit.Dropins {
						if d.Name == nd.Name {
							return units, fmt.Errorf("unit '%s' already has drop-in '%s' defined", u.Name, d.Name)
						}
					}
				}
				u.Dropins = append(u.Dropins, newUnit.Dropins...)
			}
			return units, nil
		}
	}
	return append(units, newUnit), nil
}

// WriteTo writes the fully rendered MachineConfig object as yaml to the given io.Writer, after stripping empty fields
func (m *McMaker) WriteTo(output io.Writer) (int64, error) {
	//Combine the ingition struct into the mc struct
	rawIgnition, err := json.Marshal(m.i)
	if err != nil {
		return 0, err
	}
	m.mc.Spec.Config = runtime.RawExtension{Raw: rawIgnition}

	// Marshal to json to do 1st-order stripping (omitempty)
	b, err := json.Marshal(m.mc)
	if err != nil {
		return 0, err
	}

	//convert to raw map for 2nd-order stripping
	var c map[string]interface{}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return 0, err
	}

	//trim out any zero values recursively
	d := yamltrim.YamlTrim(c)
	if d == nil {
		return 0, fmt.Errorf("empty machineconfig")
	}

	// Finally marshal to yaml and write it out
	yamlBytes, err := yaml.Marshal(d)
	if err != nil {
		return 0, err
	}
	n, err := output.Write(yamlBytes)
	return int64(n), err
}
