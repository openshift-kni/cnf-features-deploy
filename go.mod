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
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.3.0
	github.com/k8snetworkplumbingwg/sriov-network-operator v0.0.0-00010101000000-000000000000
	github.com/kennygrant/sanitize v1.2.4
	github.com/lack/mcmaker v0.0.6
	github.com/lack/yamltrim v0.0.1
	github.com/metallb/metallb-operator v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.24.1
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20211123155909-217139c4a6bd
	github.com/open-policy-agent/gatekeeper v0.0.0-20211201075931-d7de2a075a41
	github.com/openshift-kni/numaresources-operator v0.0.0-00010101000000-000000000000
	github.com/openshift-psap/special-resource-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/openshift/cluster-nfd-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/cluster-node-tuning-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/machine-config-operator v0.0.1-0.20220706180257-35d79621a587
	github.com/openshift/ptp-operator v0.0.0-20221109224215-0229788f84f5
	github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify v1.7.1
	golang.org/x/sys v0.13.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.26.0
	k8s.io/apiextensions-apiserver v0.26.0
	k8s.io/apimachinery v0.26.0
	k8s.io/client-go v1.5.2
	k8s.io/klog v1.0.0
	k8s.io/kubelet v0.24.1
	k8s.io/kubernetes v1.25.4
	k8s.io/utils v0.0.0-20221128185143-99ec85e7a448
	kubevirt.io/qe-tools v0.1.6
	sigs.k8s.io/controller-runtime v0.13.0
)

require github.com/k8snetworkplumbingwg/multi-networkpolicy v0.0.0-20220419111628-220baf5d60c1

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.0 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20210826220005-b48c857c3a0e // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/clarketm/json v1.14.1 // indirect
	github.com/coreos/fcct v0.5.0 // indirect
	github.com/coreos/go-json e49c8866685a // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/coreos/ign-converter v0.0.0-20201123214124-8dac862888aa // indirect
	github.com/coreos/ignition/v2 v2.13.0 // indirect
	github.com/coreos/vcontext v0.0.0-20211021162308-f1dbbca7bef4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/cel-go v0.10.1 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/huandu/xstrings v1.3.3 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jaypipes/ghw v0.9.0 // indirect
	github.com/jaypipes/pcidb v1.0.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/k8stopologyawareschedwg/deployer v0.6.0 // indirect
	github.com/k8stopologyawareschedwg/noderesourcetopology-api v0.0.12 // indirect
	github.com/k8stopologyawareschedwg/resource-topology-exporter v0.4.3-0.20220407072408-d8e355742882 // indirect
	github.com/kubevirt/device-plugin-manager v1.18.8 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/openshift/custom-resource-status v0.0.0-20210221154447-420d9ecf2a00 // indirect
	github.com/operator-framework/api v0.10.7 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/test-network-function/graphsolver-exports v0.0.1 // indirect
	github.com/test-network-function/graphsolver-lib v0.0.2 // indirect
	github.com/test-network-function/l2discovery-exports v0.0.1 // indirect
	github.com/test-network-function/l2discovery-lib v0.0.2 // indirect
	github.com/test-network-function/privileged-daemonset v0.0.4 // indirect
	github.com/vincent-petithory/dataurl v1.0.0 // indirect
	github.com/yourbasic/graph v0.0.0-20210606180040-8ecfec1c2869 // indirect
	go4.org v0.0.0-20200104003542-c7e774b10ea0 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/oauth2 v0.0.0-20220524215830-622c5d57e401 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220502173005-c8bf987b8c21 // indirect
	google.golang.org/grpc v1.49.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiserver v0.26.0 // indirect
	k8s.io/component-base v0.26.0 // indirect
	k8s.io/klog/v2 v2.80.1 // indirect
	k8s.io/kube-openapi v0.0.0-20221012153701-172d655c2280 // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

// Pinned to kubernetes-1.24.1
replace (
	k8s.io/api => k8s.io/api v0.24.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.24.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.24.1
	k8s.io/apiserver => k8s.io/apiserver v0.24.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.24.1
	k8s.io/client-go => k8s.io/client-go v0.24.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.24.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.24.1
	k8s.io/code-generator => k8s.io/code-generator v0.24.1
	k8s.io/component-base => k8s.io/component-base v0.24.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.24.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.24.1
	k8s.io/cri-api => k8s.io/cri-api v0.24.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.24.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.24.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.24.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.24.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.24.1
	k8s.io/kubectl => k8s.io/kubectl v0.24.1
	k8s.io/kubelet => k8s.io/kubelet v0.24.1
	k8s.io/kubernetes => k8s.io/kubernetes v1.24.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.24.1
	k8s.io/metrics => k8s.io/metrics v0.24.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.24.1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.24.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.24.1
)

// Other pinned deps
replace (
	github.com/apache/thrift => github.com/apache/thrift v0.14.0
	github.com/cri-o/cri-o => github.com/cri-o/cri-o v1.18.1
	github.com/go-log/log => github.com/go-log/log v0.1.0
	github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2
	github.com/mtrmac/gpgme => github.com/mtrmac/gpgme v0.1.1
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210713130143-be21c6cb1bea // release-4.8
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210521082421-73d9475a9142 // release-4.8
	github.com/openshift/cluster-node-tuning-operator => github.com/openshift/cluster-node-tuning-operator v0.0.0-20230807104110-54538a21a40e // release-4.12
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20221012165547-f859132ee700 // release-4.8
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20210701174259-29813c845a4a // release-4.8
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.11.0
	sigs.k8s.io/json => sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2
)

// Test deps
replace (
	github.com/k8snetworkplumbingwg/sriov-network-operator => github.com/openshift/sriov-network-operator v0.0.0-20221118104808-b97559f25392 // release-4.12
	github.com/metallb/metallb-operator => github.com/openshift/metallb-operator v0.0.0-20231031121015-1489c3b8abf2 //release-4.12
	github.com/openshift-kni/numaresources-operator => github.com/openshift-kni/numaresources-operator v0.4.10-3.2022042201
	github.com/openshift-psap/special-resource-operator => github.com/openshift/special-resource-operator v0.0.0-20211202035230-4c86f99c426b // release-4.10
	github.com/openshift/cluster-nfd-operator => github.com/openshift/cluster-nfd-operator v0.0.0-20210727033955-e8e9697b5ffc // release-4.9
	github.com/openshift/ptp-operator => github.com/openshift/ptp-operator v0.0.0-20221109224215-0229788f84f5 // release-4.12
)
