module github.com/openshift-kni/cnf-features-deploy

go 1.19

require (
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/coreos/ignition v0.35.0
	github.com/gatekeeper/gatekeeper-operator v0.2.1
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/golang/glog v1.0.0
	github.com/google/go-cmp v0.5.9
	github.com/ishidawataru/sctp v0.0.0-20210707070123-9a39160e9062
	github.com/k8snetworkplumbingwg/multi-networkpolicy v0.0.0-20220908143610-19b7d2ba63f9
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.3.0
	github.com/k8snetworkplumbingwg/sriov-network-operator v0.0.0-00010101000000-000000000000
	github.com/lack/mcmaker v0.0.6
	github.com/lack/yamltrim v0.0.1
	github.com/metallb/metallb-operator v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo/v2 v2.8.4
	github.com/onsi/gomega v1.27.1
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20211123155909-217139c4a6bd
	github.com/open-policy-agent/gatekeeper v0.0.0-20211201075931-d7de2a075a41
	github.com/openshift-kni/k8sreporter v1.0.2
	github.com/openshift-psap/special-resource-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/openshift/cluster-nfd-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/cluster-node-tuning-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/machine-config-operator v0.0.1-0.20230118083703-fc27a2bdaa85
	github.com/openshift/ptp-operator v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify v1.8.2
	golang.org/x/sys v0.5.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.26.1
	k8s.io/apiextensions-apiserver v0.26.1
	k8s.io/apimachinery v0.26.1
	k8s.io/client-go v1.5.2
	k8s.io/klog v1.0.0
	k8s.io/kubelet v0.26.1
	k8s.io/kubernetes v1.26.1
	k8s.io/utils v0.0.0-20230115233650-391b47cb4029
	kubevirt.io/qe-tools v0.1.8
	sigs.k8s.io/controller-runtime v0.14.1
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.0 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/clarketm/json v1.14.1 // indirect
	github.com/coreos/fcct v0.5.0 // indirect
	github.com/coreos/go-json v0.0.0-20220810161552-7cce03887f34 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/coreos/ign-converter v0.0.0-20201123214124-8dac862888aa // indirect
	github.com/coreos/ignition/v2 v2.14.0 // indirect
	github.com/coreos/vcontext v0.0.0-20220810162454-88bd546c634c // indirect
	github.com/creasty/defaults v1.6.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.10.1 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/cel-go v0.12.6 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/openshift/custom-resource-status v0.0.0-20210221154447-420d9ecf2a00 // indirect
	github.com/operator-framework/api v0.10.7 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/test-network-function/graphsolver-exports v0.0.1 // indirect
	github.com/test-network-function/graphsolver-lib v0.0.2 // indirect
	github.com/test-network-function/l2discovery-exports v0.0.1 // indirect
	github.com/test-network-function/l2discovery-lib v0.0.5 // indirect
	github.com/test-network-function/privileged-daemonset v0.0.5 // indirect
	github.com/vincent-petithory/dataurl v1.0.0 // indirect
	github.com/yourbasic/graph v0.0.0-20210606180040-8ecfec1c2869 // indirect
	go4.org v0.0.0-20200104003542-c7e774b10ea0 // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/oauth2 v0.4.0 // indirect
	golang.org/x/term v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20221227171554-f9683d7f8bef // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiserver v0.26.1 // indirect
	k8s.io/component-base v0.26.1 // indirect
	k8s.io/klog/v2 v2.90.0 // indirect
	k8s.io/kube-openapi v0.0.0-20230123231816-1cb3ae25d79a // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

// Pinned to kubernetes-1.26.1
replace (
	k8s.io/api => k8s.io/api v0.26.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.26.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.26.1
	k8s.io/apiserver => k8s.io/apiserver v0.26.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.26.1
	k8s.io/client-go => k8s.io/client-go v0.26.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.26.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.26.1
	k8s.io/code-generator => k8s.io/code-generator v0.26.1
	k8s.io/component-base => k8s.io/component-base v0.26.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.26.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.26.1
	k8s.io/cri-api => k8s.io/cri-api v0.26.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.26.1
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.26.1
	k8s.io/kms => k8s.io/kms v0.26.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.26.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.26.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.26.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.26.1
	k8s.io/kubectl => k8s.io/kubectl v0.26.1
	k8s.io/kubelet => k8s.io/kubelet v0.26.1
	k8s.io/kubernetes => k8s.io/kubernetes v1.26.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.26.1
	k8s.io/metrics => k8s.io/metrics v0.26.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.26.1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.26.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.26.1
)

// Other pinned deps
replace (
	github.com/apache/thrift => github.com/apache/thrift v0.14.0
	github.com/cri-o/cri-o => github.com/cri-o/cri-o v1.18.1
	github.com/go-log/log => github.com/go-log/log v0.1.0
	github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2
	github.com/mtrmac/gpgme => github.com/mtrmac/gpgme v0.1.1
	github.com/openshift/api => github.com/openshift/api v0.0.0-20230414143018-3367bc7e6ac7 // release-4.13
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210521082421-73d9475a9142 // release-4.8
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20230130232623-47904dd9ff5a // release-4.13
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20230419202402-70aa0a560c0b // release-4.13
	github.com/test-network-function/l2discovery-lib => github.com/test-network-function/l2discovery-lib v0.0.5
)

// Test deps
replace (
	github.com/k8snetworkplumbingwg/sriov-network-operator => github.com/openshift/sriov-network-operator v0.0.0-20230330150324-84715294edb9 // release-4.13
	github.com/k8stopologyawareschedwg/resource-topology-exporter => github.com/k8stopologyawareschedwg/resource-topology-exporter v0.8.0
	github.com/metallb/metallb-operator => github.com/openshift/metallb-operator v0.0.0-20230807120428-6267b32eaa61 // release-4.13
	github.com/openshift-psap/special-resource-operator => github.com/openshift/special-resource-operator v0.0.0-20211202035230-4c86f99c426b // release-4.10
	github.com/openshift/cluster-nfd-operator => github.com/openshift/cluster-nfd-operator v0.0.0-20210727033955-e8e9697b5ffc // release-4.9
	github.com/openshift/cluster-node-tuning-operator => github.com/openshift/cluster-node-tuning-operator v0.0.0-20230711003646-5b4af42e9cfb // release-4.13
	github.com/openshift/ptp-operator => github.com/openshift/ptp-operator v0.0.0-20230206122400-e0231ea64d3a // release-4.13
)
