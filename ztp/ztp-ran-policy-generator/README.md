# ztp-ran-policy-generator
-  to build the plugin

    - $ cd ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/
    - $ go build -o RanPolicyGenerator

-  to execute kustomize

    - $ cd cnf-features-deploy/ztp/ztp-ran-policy-generator/
    - $ XDG_CONFIG_HOME=./ kustomize build --enable-alpha-plugins
