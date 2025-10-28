module github.com/openshift-kni/cnf-features-deploy

// This go version is the highest level source of truth in this project.
// This should be matching the current branches OCP release golang version.
// This should also be matched within this project at:
//   - cnf-tests/Dockerfile*
//   - openshift-ci/Dockerfile*
//   - ztp/resource-generator/Containerfile
//   - ztp/tools/pgt2acmpg/go.mod
go 1.23

require (
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/coreos/ignition v0.35.0
	github.com/gatekeeper/gatekeeper-operator v0.2.1
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v1.4.2
	github.com/go-logr/stdr v1.2.2
	github.com/google/go-cmp v0.6.0
	github.com/ishidawataru/sctp v0.0.0-20210707070123-9a39160e9062
	github.com/jaypipes/ghw v0.14.0
	github.com/k8snetworkplumbingwg/multi-networkpolicy v1.0.1
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.4.0
	github.com/k8snetworkplumbingwg/sriov-network-operator v0.0.0-00010101000000-000000000000
	github.com/lack/mcmaker v0.0.7
	github.com/lack/yamltrim v0.0.1
	github.com/nmstate/kubernetes-nmstate/api v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo/v2 v2.22.1
	github.com/onsi/gomega v1.36.2
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20230822235116-f0b62fe1e4c4
	github.com/open-policy-agent/gatekeeper/v3 v3.13.0
	github.com/openshift-kni/k8sreporter v1.0.7
	github.com/openshift/api v0.0.0-20240530231226-9d1c2e5ff5a8
	github.com/openshift/client-go v0.0.0-20240415214935-be70f772f157
	github.com/openshift/cluster-nfd-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/cluster-node-tuning-operator v0.0.0-00010101000000-000000000000
	github.com/openshift/machine-config-operator v0.0.1-0.20231024085435-7e1fb719c1ba
	github.com/openshift/ptp-operator v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.10.0
	golang.org/x/sys v0.30.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.31.1
	k8s.io/apiextensions-apiserver v0.31.1
	k8s.io/apimachinery v0.31.1
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.130.1
	k8s.io/kubelet v0.31.1
	k8s.io/utils v0.0.0-20240711033017-18e509b52bc8
	kubevirt.io/qe-tools v0.1.8
	open-cluster-management.io/config-policy-controller v0.10.0
	open-cluster-management.io/governance-policy-propagator v0.12.0
	sigs.k8s.io/controller-runtime v0.19.0
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20200220154158-5ae607161559 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go v1.50.25 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clarketm/json v1.17.1 // indirect
	github.com/coreos/fcct v0.5.0 // indirect
	github.com/coreos/go-json v0.0.0-20230131223807-18775e0fb4fb // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/coreos/ign-converter v0.0.0-20230417193809-cee89ea7d8ff // indirect
	github.com/coreos/ignition/v2 v2.18.0 // indirect
	github.com/coreos/vcontext v0.0.0-20231102161604-685dc7299dc5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.12.0 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.2.5 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.20.1 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20241210010833-40e02aabc2ad // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.4.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/openshift/custom-resource-status v1.1.3-0.20220503160415-f2fdb4999d87 // indirect
	github.com/openshift/hypershift/api v0.0.0-20240604072534-cd2d5291e2b7 // indirect
	github.com/openshift/library-go v0.0.0-20240419113445-f1541d628746 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.68.0 // indirect
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.68.0 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/cobra v1.8.1 // indirect
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/vincent-petithory/dataurl v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go4.org v0.0.0-20230225012048-214862532bf5 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/term v0.27.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.28.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiserver v0.31.1 // indirect
	k8s.io/component-base v0.31.1 // indirect
	k8s.io/kube-aggregator v0.31.1 // indirect
	k8s.io/kube-openapi v0.0.0-20240411171206-dc4e619f62f3 // indirect
	open-cluster-management.io/multicloud-operators-subscription v0.11.0 // indirect
	sigs.k8s.io/cluster-api v1.7.2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kube-storage-version-migrator v0.0.6-0.20230721195810-5c8923c5ff96 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

// Pinned to kubernetes-1.28.3
replace (
	k8s.io/api => k8s.io/api v0.31.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.31.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.31.1
	k8s.io/apiserver => k8s.io/apiserver v0.31.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.31.1
	k8s.io/client-go => k8s.io/client-go v0.31.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.31.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.31.1
	k8s.io/code-generator => k8s.io/code-generator v0.31.1
	k8s.io/component-base => k8s.io/component-base v0.31.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.31.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.31.1
	k8s.io/cri-api => k8s.io/cri-api v0.31.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.31.1
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.31.1
	k8s.io/kms => k8s.io/kms v0.31.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.31.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.31.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.31.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.31.1
	k8s.io/kubectl => k8s.io/kubectl v0.31.1
	k8s.io/kubelet => k8s.io/kubelet v0.31.1
	k8s.io/kubernetes => k8s.io/kubernetes v0.31.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.31.1
	k8s.io/metrics => k8s.io/metrics v0.31.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.31.1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.31.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.31.1
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
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.19.0
)

// Test deps
replace (
	github.com/k8snetworkplumbingwg/sriov-network-operator => github.com/openshift/sriov-network-operator v0.0.0-20250305152347-bbd92d9a0c64 // release-4.16
	github.com/nmstate/kubernetes-nmstate/api => github.com/openshift/kubernetes-nmstate/api v0.0.0-20240726065608-fbf9eb6f75e6
	github.com/openshift/cluster-nfd-operator => github.com/openshift/cluster-nfd-operator v0.0.0-20240125121050-830c889e311e // release-4.9
	github.com/openshift/cluster-node-tuning-operator => github.com/openshift/cluster-node-tuning-operator v0.0.0-20251024081946-eb5caaf6e854 // release-4.18
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
