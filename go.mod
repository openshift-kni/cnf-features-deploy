module github.com/openshift-kni/cnf-features-deploy

go 1.13

require (
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/spec v0.19.7 // indirect
	github.com/go-openapi/swag v0.19.9 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.4.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/ishidawataru/sctp v0.0.0-20191218070446-00ab2ac2db07
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/openshift-kni/performance-addon-operators v0.0.0-20200514145830-82baa38e9b10
	github.com/openshift/api v3.9.1-0.20191213091414-3fbf6bcf78e8+incompatible
	github.com/openshift/client-go v0.0.0-20191205152420-9faca5198b4f
	github.com/openshift/cluster-node-tuning-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/openshift/ptp-operator v0.0.0-20200511111616-3d72fdb1c731
	github.com/openshift/sriov-network-operator v0.0.0-20200512234214-8079cf03e552
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.6 // indirect
	go.uber.org/atomic v1.5.0 // indirect
	go4.org v0.0.0-20200411211856-f5505b9728dd // indirect
	golang.org/x/crypto v0.0.0-20200420104511-884d27f42877 // indirect
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	k8s.io/api v0.18.2
	k8s.io/apiextensions-apiserver v0.17.3
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubelet v0.18.2
	k8s.io/kubernetes v1.17.0
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66
	kubevirt.io/qe-tools v0.1.6
	sigs.k8s.io/controller-runtime v0.5.2
	sigs.k8s.io/yaml v1.2.0 // indirect
)

// Pinned to kubernetes-1.17.0
replace (
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.0
	k8s.io/apiserver => k8s.io/apiserver v0.17.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.0
	k8s.io/client-go => k8s.io/client-go v0.17.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.0
	k8s.io/code-generator => k8s.io/code-generator v0.17.0
	k8s.io/component-base => k8s.io/component-base v0.17.0
	k8s.io/cri-api => k8s.io/cri-api v0.17.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.0
	k8s.io/kubectl => k8s.io/kubectl v0.17.0
	k8s.io/kubelet => k8s.io/kubelet v0.17.0
	k8s.io/kubernetes => k8s.io/kubernetes v1.17.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.0
	k8s.io/metrics => k8s.io/metrics v0.17.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.0
)

// Other pinned deps
replace (
	github.com/cri-o/cri-o => github.com/cri-o/cri-o v1.16.1
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
	github.com/go-log/log => github.com/go-log/log v0.1.0
	github.com/mtrmac/gpgme => github.com/mtrmac/gpgme v0.1.1
	github.com/openshift/api => github.com/openshift/api v0.0.0-20191220175332-378bec237e34 // release-4.4
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20191205152420-9faca5198b4f // release-4.4
	github.com/openshift/cluster-node-tuning-operator => github.com/openshift/cluster-node-tuning-operator v0.0.0-20191217222311-500135cb8754 // release-4.4
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.0-20200123151440-ca3e3e1921f3 // release-4.4
	golang.org/x/tools => golang.org/x/tools v0.0.0-20191206213732-070c9d21b343
)

// Test deps
replace (
	github.com/openshift-kni/performance-addon-operators => github.com/openshift-kni/performance-addon-operators v0.0.0-20200514145830-82baa38e9b10 // release-4.5
	github.com/openshift/ptp-operator => github.com/openshift/ptp-operator v0.0.0-20200511111616-3d72fdb1c731 // release-4.5
	github.com/openshift/sriov-network-operator => github.com/openshift/sriov-network-operator v0.0.0-20200512234214-8079cf03e552 // release-4.5
)
