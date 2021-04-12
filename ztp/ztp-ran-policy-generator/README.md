# ztp-ran-policy-generator
-  to build the plugin

    - $ cd ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/
    - $ go build -o RanPolicyGenerator

-  to execute kustomize

    - $ cd cnf-features-deploy/
    - $ XDG_CONFIG_HOME=./ztp/ztp-ran-policy-generator/ kustomize build --enable-alpha-plugins
