module github.com/openshift-kni/cnf-features-deploy

go 1.13

require (
	github.com/ghodss/yaml v1.0.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.7.1
	github.com/openshift/api v3.9.1-0.20191213091414-3fbf6bcf78e8+incompatible // indirect
	github.com/openshift/client-go v0.0.0-20191205152420-9faca5198b4f
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/vincent-petithory/dataurl v0.0.0-20191104211930-d1553a71de50 // indirect
	golang.org/x/crypto v0.0.0-20200109152110-61a87790db17 // indirect
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kubelet v0.17.0
	k8s.io/utils v0.0.0-20200109141947-94aeca20bf09
)

// Pinned to kubernetes-1.17.0
replace (
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.0
	k8s.io/apiserver => k8s.io/apiserver v0.17.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.0
	k8s.io/client-go => k8s.io/client-go v0.17.0
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
	github.com/go-log/log => github.com/go-log/log v0.1.0
	github.com/openshift/api => github.com/openshift/api v0.0.0-20191220175332-378bec237e34 // release-4.4
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20191205152420-9faca5198b4f // release-4.4
	github.com/openshift/cluster-node-tuning-operator => github.com/openshift/cluster-node-tuning-operator v0.0.0-20191217222311-500135cb8754 // release-4.4
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.0-20191220033234-347a7a09e869 // release-4.4
	golang.org/x/tools => golang.org/x/tools v0.0.0-20191206213732-070c9d21b343
)
