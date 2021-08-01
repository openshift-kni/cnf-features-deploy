package siteConfig

import (
	"fmt"
	"reflect"
	"strings"
)

const clusterCRsFileName = "cluster-crs.yaml"
const operatorsGroupsPath = "operators-groups"
const operatorGroupsFile = "02-operators-groups.yaml"
const workloadPath = "workload"
const workloadFile = "03-workload-partitioning.yaml"
const workloadCrioFile = "crio.conf"
const workloadKubeletFile = "kubelet.conf"
const cpuset = "$cpuset"
const mountNSPath = "mount-ns"
const mountNSFile = "01-container-mount-ns.yaml"

var Separator = []byte("---\n")

func (sc *SiteConfig) GetSiteConfigFieldValue(path string, clusterId int, nodeId int) (interface{}, error) {
	keys := strings.Split(path, ".")
	v := reflect.ValueOf(sc)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for _, key := range keys[1:] {
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			intf := v.Interface()
			arrClusters, ok := intf.([]Clusters)

			if ok {
				v = reflect.ValueOf(arrClusters[clusterId])
			}
			arrNodes, ok := intf.([]Nodes)

			if ok {
				v = reflect.ValueOf(arrNodes[nodeId])
			}
			v = v.FieldByName(key)
		} else if v.Kind() == reflect.Struct {
			v = v.FieldByName(key)
		} else if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("only accepts structs; got %T", v)
		}
	}

	if !v.IsValid() {
		return "", nil
	}

	return v.Interface(), nil
}

// SiteConfig
type SiteConfig struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

// Metadata
type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

// Spec
type Spec struct {
	PullSecretRef          PullSecretRef          `yaml:"pullSecretRef"`
	ClusterImageSetNameRef string                 `yaml:"clusterImageSetNameRef"`
	SshPublicKey           string                 `yaml:"sshPublicKey"`
	SshPrivateKeySecretRef SshPrivateKeySecretRef `yaml:"sshPrivateKeySecretRef"`
	Clusters               []Clusters             `yaml:"clusters"`
	BaseDomain             string                 `yaml:"baseDomain"`
}

// PullSecretRef
type PullSecretRef struct {
	Name string `yaml:"name"`
}

// SshPrivateKeySecretRef
type SshPrivateKeySecretRef struct {
	Name string `yaml:"name"`
}

// Clusters
type Clusters struct {
	ApiVIP                 string            `yaml:"apiVIP"`
	IngressVIP             string            `yaml:"ingressVIP"`
	ClusterName            string            `yaml:"clusterName"`
	AdditionalNTPSources   []string          `yaml:"additionalNTPSources"`
	Nodes                  []Nodes           `yaml:"nodes"`
	MachineNetwork         []MachineNetwork  `yaml:"machineNetwork"`
	ServiceNetwork         []string          `yaml:"serviceNetwork"`
	ManifestsConfig        ManifestsConfig   `yaml:"manifestsConfig"`
	ClusterType            string            `yaml:"clusterType"`
	ClusterProfile         string            `yaml:"clusterProfile"`
	ClusterLabels          map[string]string `yaml:"clusterLabels"`
	ClusterNetwork         []ClusterNetwork  `yaml:"clusterNetwork"`
	IgnitionConfigOverride string            `yaml:"ignitionConfigOverride"`
}

// Nodes
type Nodes struct {
	BmcAddress         string                 `yaml:"bmcAddress"`
	BootMACAddress     string                 `yaml:"bootMACAddress"`
	RootDeviceHints    map[string]interface{} `yaml:"rootDeviceHints"`
	Cpuset             string                 `yaml:"cpuset"`
	NodeNetwork        NodeNetwork            `yaml:"nodeNetwork"`
	HostName           string                 `yaml:"hostName"`
	BmcCredentialsName BmcCredentialsName     `yaml:"bmcCredentialsName"`
	BootMode           string                 `yaml:"bootMode"`
	UserData           map[string]interface{} `yaml:"userData"`
}

// MachineNetwork
type MachineNetwork struct {
	Cidr string `yaml:"cidr"`
}

// ClusterNetwork
type ClusterNetwork struct {
	Cidr       string `yaml:"cidr"`
	HostPrefix int    `yaml:"hostPrefix"`
}

// ManifestsConfig
type ManifestsConfig struct {
	NtpServer string `yaml:"ntpServer"`
}

// NodeNetwork
type NodeNetwork struct {
	Config     map[string]interface{} `yaml:"config"`
	Interfaces []Interfaces           `yaml:"interfaces"`
}

// BmcCredentialsName
type BmcCredentialsName struct {
	Name string `yaml:"name"`
}

// Interfaces
type Interfaces struct {
	Name       string `yaml:"name"`
	MacAddress string `yaml:"macAddress"`
}
