module github.com/openshift-kni/cnf-features-deploy

go 1.20

require (
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/coreos/ignition v0.35.0
	github.com/gatekeeper/gatekeeper-operator v0.2.1
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/golang/glog v1.2.0
	github.com/google/go-cmp v0.6.0
	github.com/ishidawataru/sctp v0.0.0-20210707070123-9a39160e9062
	github.com/k8snetworkplumbingwg/multi-networkpolicy v0.0.0-20220908143610-19b7d2ba63f9
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.4.0
	github.com/k8snetworkplumbingwg/sriov-network-operator v0.0.0-00010101000000-000000000000
	github.com/lack/mcmaker v0.0.6
	github.com/lack/yamltrim v0.0.1
	github.com/onsi/ginkgo/v2 v2.15.0
	github.com/onsi/gomega v1.31.1
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20230822235116-f0b62fe1e4c4
	github.com/open-policy-agent/gatekeeper/v3 v3.13.0
	github.com/openshift-kni/k8sreporter v1.0.5
	github.com/openshift-psap/special-resource-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/api v0.0.0-20230807121159-a81c3efc8824
	github.com/openshift/client-go v0.0.0-20230807132528-be5346fb33cb
	github.com/openshift/cluster-nfd-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/cluster-node-tuning-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/machine-config-operator v0.0.1-0.20230807154212-886c5c3fc7a9
	github.com/openshift/ptp-operator v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.8.4
	golang.org/x/sys v0.18.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.28.3
	k8s.io/apiextensions-apiserver v0.28.3
	k8s.io/apimachinery v0.28.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.120.0
	k8s.io/kubelet v0.28.3
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b
	kubevirt.io/qe-tools v0.1.8
	open-cluster-management.io/config-policy-controller v0.10.0
	open-cluster-management.io/governance-policy-propagator v0.12.0
	sigs.k8s.io/controller-runtime v0.16.3
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20200220154158-5ae607161559 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230305170008-8188dc5388df // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go v1.44.302 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/clarketm/json v1.17.1 // indirect
	github.com/coreos/fcct v0.5.0 // indirect
	github.com/coreos/go-json v0.0.0-20230131223807-18775e0fb4fb // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/coreos/ign-converter v0.0.0-20230417193809-cee89ea7d8ff // indirect
	github.com/coreos/ignition/v2 v2.15.0 // indirect
	github.com/coreos/vcontext v0.0.0-20230201181013-d72178a18687 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.7.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/zapr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.20.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/cel-go v0.16.1 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20230705174524-200ffdc848b8 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/openshift/custom-resource-status v1.1.3-0.20220503160415-f2fdb4999d87 // indirect
	github.com/openshift/library-go v0.0.0-20230803043003-e1dfb9bf12bb // indirect
	github.com/operator-framework/api v0.10.7 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.57.0 // indirect
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.57.0 // indirect
	github.com/prometheus/client_golang v1.17.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/cobra v1.7.0 // indirect
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/vincent-petithory/dataurl v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.25.0 // indirect
	go4.org v0.0.0-20200104003542-c7e774b10ea0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/oauth2 v0.13.0 // indirect
	golang.org/x/sync v0.5.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.16.1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230717213848-3f92550aa753 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230717213848-3f92550aa753 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiserver v0.28.3 // indirect
	k8s.io/component-base v0.28.3 // indirect
	k8s.io/kube-aggregator v0.27.4 // indirect
	k8s.io/kube-openapi v0.0.0-20231010175941-2dd684a91f00 // indirect
	open-cluster-management.io/multicloud-operators-subscription v0.11.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kube-storage-version-migrator v0.0.6-0.20230721195810-5c8923c5ff96 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.3.0 // indirect
)

// Pinned to kubernetes-1.28.3
replace (
	k8s.io/api => k8s.io/api v0.28.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.28.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.28.3
	k8s.io/apiserver => k8s.io/apiserver v0.28.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.28.3
	k8s.io/client-go => k8s.io/client-go v0.28.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.28.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.28.3
	k8s.io/code-generator => k8s.io/code-generator v0.28.3
	k8s.io/component-base => k8s.io/component-base v0.28.3
	k8s.io/component-helpers => k8s.io/component-helpers v0.28.3
	k8s.io/controller-manager => k8s.io/controller-manager v0.28.3
	k8s.io/cri-api => k8s.io/cri-api v0.28.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.28.3
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.28.3
	k8s.io/kms => k8s.io/kms v0.28.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.28.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.28.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.28.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.28.3
	k8s.io/kubectl => k8s.io/kubectl v0.28.3
	k8s.io/kubelet => k8s.io/kubelet v0.28.3
	k8s.io/kubernetes => k8s.io/kubernetes v1.28.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.28.3
	k8s.io/metrics => k8s.io/metrics v0.28.3
	k8s.io/mount-utils => k8s.io/mount-utils v0.28.3
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.28.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.28.3
)

// Other pinned deps
replace (
	github.com/apache/thrift => github.com/apache/thrift v0.14.0
	github.com/cri-o/cri-o => github.com/cri-o/cri-o v1.18.1
	github.com/go-log/log => github.com/go-log/log v0.1.0
	github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2
	github.com/mtrmac/gpgme => github.com/mtrmac/gpgme v0.1.1
	github.com/open-policy-agent/gatekeeper/v3 => github.com/open-policy-agent/gatekeeper/v3 v3.13.0
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210521082421-73d9475a9142 // release-4.8
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.46.0
	github.com/test-network-function/l2discovery-lib => github.com/test-network-function/l2discovery-lib v0.0.5
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.15.2
)

// Test deps
replace (
	github.com/k8snetworkplumbingwg/sriov-network-operator => github.com/openshift/sriov-network-operator v0.0.0-20240125124104-58986501f2b4 // release-4.16
	github.com/openshift-psap/special-resource-operator => github.com/openshift/special-resource-operator v0.0.0-20211202035230-4c86f99c426b // release-4.10
	github.com/openshift/cluster-nfd-operator => github.com/openshift/cluster-nfd-operator v0.0.0-20240125121050-830c889e311e // release-4.9
	github.com/openshift/cluster-node-tuning-operator => github.com/openshift/cluster-node-tuning-operator v0.0.0-20231204115124-e9fa8996e6b2 // release-4.14
	github.com/openshift/ptp-operator => github.com/openshift/ptp-operator v0.0.0-20230831212656-4b8be2662cfe // release-4.14
)

// ZTP must produce MachineConfig resources with ignition version v3.2.0
// Since https://github.com/openshift/machine-config-operator/pull/3814, MCO writes v3.4.0 ignition configuration.
// https://github.com/openshift/machine-config-operator/commit/63d7be1ef18b86826b47c61172c7a9dc7c2b6de1 is the commit just before the culprit one.
replace (
	// This bump is required to avoid the `go mod tidy` error:
	// 	go: github.com/openshift/machine-config-operator@v0.0.1-0.20230807154212-886c5c3fc7a9 requires
	//    	github.com/containers/common@v0.50.1 requires
	//    	github.com/containerd/containerd@v1.6.8 requires
	//    	github.com/containerd/aufs@v1.0.0 requires
	//    	github.com/containerd/containerd@v1.5.0-beta.3 requires
	//    	github.com/Microsoft/hcsshim@v0.8.15 requires
	//    	github.com/containerd/containerd@v1.5.0-beta.1 requires
	//    	github.com/Microsoft/hcsshim/test@v0.0.0-20201218223536-d3e5debf77da requires
	//    	github.com/Microsoft/hcsshim@v0.8.7 requires
	//    	k8s.io/kubernetes@v1.13.0 requires
	//    	k8s.io/endpointslice@v0.0.0: reading k8s.io/endpointslice/go.mod at revision v0.0.0: unknown revision v0.0.0
	//
	// See https://github.com/microsoft/hcsshim/pull/783
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.8

	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20230811181556-63d7be1ef18b
)
