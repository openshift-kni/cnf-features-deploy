module github.com/openshift-kni/cnf-features-deploy

go 1.17

require (
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/ignition v0.35.0
	github.com/gatekeeper/gatekeeper-operator v0.1.1
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/golang/glog v1.0.0
	github.com/ishidawataru/sctp v0.0.0-20191218070446-00ab2ac2db07
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20200626054723-37f83d1996bc
	github.com/k8snetworkplumbingwg/sriov-network-operator v1.0.1-0.20211126031536-11faae79733e
	github.com/kennygrant/sanitize v1.2.4
	github.com/lack/mcmaker v0.0.6
	github.com/metallb/metallb-operator v0.0.0-20211202081249-1b0df396f552
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20211123155909-217139c4a6bd
	github.com/open-policy-agent/gatekeeper v0.0.0-20211201075931-d7de2a075a41
	github.com/openshift-kni/performance-addon-operators v0.0.0-20210722194338-183a9c3da026
	github.com/openshift-psap/special-resource-operator v0.0.0-20210726202540-2fdec192a48e
	github.com/openshift/api v3.9.1-0.20191213091414-3fbf6bcf78e8+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/openshift/cluster-nfd-operator v0.0.0-20210727033955-e8e9697b5ffc
	github.com/openshift/cluster-node-tuning-operator v0.0.0-20200914165052-a39511828cf0
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/openshift/ptp-operator v0.0.0-20210714172658-472d32e04af5
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20211029165221-6e7872819dc8
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.23.3
	k8s.io/apiextensions-apiserver v0.23.3
	k8s.io/apimachinery v0.23.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kubelet v0.23.3
	k8s.io/kubernetes v1.21.1
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	kubevirt.io/qe-tools v0.1.6
	sigs.k8s.io/controller-runtime v0.11.0
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20210826220005-b48c857c3a0e // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/clarketm/json v1.14.1 // indirect
	github.com/coreos/fcct v0.5.0 // indirect
	github.com/coreos/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/coreos/ign-converter v0.0.0-20201123214124-8dac862888aa // indirect
	github.com/coreos/ignition/v2 v2.11.0 // indirect
	github.com/coreos/vcontext v0.0.0-20210407161507-4ee6c745c8bd // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/cel-go v0.9.0 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lack/yamltrim v0.0.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mikioh/ipaddr v0.0.0-20190404000644-d465c8ab6721 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/openshift/custom-resource-status v0.0.0-20210221154447-420d9ecf2a00 // indirect
	github.com/operator-framework/operator-lifecycle-manager v3.11.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.1 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/vincent-petithory/dataurl v0.0.0-20191104211930-d1553a71de50 // indirect
	go4.org v0.0.0-20201209231011-d4a079459e60 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/net v0.0.0-20211209124913-491a49abca63 // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/tools v0.1.6-0.20210820212750-d4cc65f0b2ff // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210831024726-fe130286e0e2 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/component-base v0.23.3 // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65 // indirect
	sigs.k8s.io/json v0.0.0-20211020170558-c049b76a60c6 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

// Pinned to kubernetes-1.21.2
replace (
	k8s.io/api => k8s.io/api v0.23.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.2
	k8s.io/apiserver => k8s.io/apiserver v0.23.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.2
	k8s.io/client-go => k8s.io/client-go v0.23.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.2
	k8s.io/code-generator => k8s.io/code-generator v0.23.2
	k8s.io/component-base => k8s.io/component-base v0.23.2
	k8s.io/component-helpers => k8s.io/component-helpers v0.23.2
	k8s.io/controller-manager => k8s.io/controller-manager v0.23.2
	k8s.io/cri-api => k8s.io/cri-api v0.23.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.2
	k8s.io/kubectl => k8s.io/kubectl v0.23.2
	k8s.io/kubelet => k8s.io/kubelet v0.23.2
	k8s.io/kubernetes => k8s.io/kubernetes v1.23.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.2
	k8s.io/metrics => k8s.io/metrics v0.23.2
	k8s.io/mount-utils => k8s.io/mount-utils v0.23.2
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.23.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.2
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
	github.com/openshift/cluster-node-tuning-operator => github.com/openshift/cluster-node-tuning-operator v0.0.0-20210303185751-cbeeb4d9f3cc // release-4.9
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20210706120254-6f1208ffd780 // release-4.8
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20210701174259-29813c845a4a // release-4.8
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.11.0
)

// Test deps
replace (
	github.com/k8snetworkplumbingwg/sriov-network-operator => github.com/openshift/sriov-network-operator v0.0.0-20211207043958-2bfa00ead503 // release-4.10
	github.com/metallb/metallb-operator => github.com/openshift/metallb-operator v0.0.0-20220209163201-dfea3133085c //release-4.10
	github.com/openshift-kni/performance-addon-operators => github.com/openshift-kni/performance-addon-operators v0.0.41002-0.20220309141158-eddd042858ef // release-4.11
	github.com/openshift-psap/special-resource-operator => github.com/openshift/special-resource-operator v0.0.0-20211202035230-4c86f99c426b // release-4.10
	github.com/openshift/cluster-nfd-operator => github.com/openshift/cluster-nfd-operator v0.0.0-20210727033955-e8e9697b5ffc // release-4.9
	github.com/openshift/ptp-operator => github.com/openshift/ptp-operator v0.0.0-20211201021143-27df2443c98f //release-4.10
)
