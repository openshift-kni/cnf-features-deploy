package main

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

const im = "override-image"

const expected = `
apiVersion: batch/v1
kind: Job
metadata:
  name: ztp-profile-install-accelerator
  namespace: ztp-profile
spec:
  backoffLimit: 2
  template:
    spec:
      serviceAccountName: ztp-profile-accelerator-sa
      terminationGracePeriodSeconds: 3
      nodeSelector:
        node-role.kubernetes.io/master: ""
      restartPolicy: OnFailure
      containers:
        - name: ztp-accelerator
          securityContext:
            allowPrivilegeEscalation: false
            seccompProfile:
              type: RuntimeDefault
            capabilities:
              drop:
              - ALL
          image: override-image
          imagePullPolicy: Always
          command:
          - /bin/bash
          - -c
          - --
          args:
          - accelerator # -override=true
          # - "sleep inf"
          env:
          - name: CONFIGMAP_NAME
            value: "ztp-post-provision"
          - name: CONFIGMAP_NAMESPACE
            value: "ztp-profile"
          - name: END_CONDITION_EXTENSION_TIME
            value: 60saaa
`

func TestJob_renderYamlTemplates(t *testing.T) {
	var data templateData
	data.ZtpImage = im
	buf, err := renderYamlTemplate("job", job, data)
	if err != nil {
		t.Errorf("error rendering yaml template: %v", err)
	}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	renderedObject, _, err := dec.Decode(buf.Bytes(), nil, obj)
	if err != nil {
		t.Errorf("error serializing yaml template: %v", err)
	}
	obj1 := &unstructured.Unstructured{}
	desiredObject, _, err := dec.Decode([]byte(expected), nil, obj1)
	if err != nil {
		t.Errorf("error serializing yaml string: %v", err)
	}
	l := log.New(os.Stderr, "", 0)

	if !reflect.DeepEqual(renderedObject, desiredObject) {
		//TODO replace duplication by loop
		renderedJSON, err := json.Marshal(renderedObject)
		if err != nil {
			l.Println(renderedObject)
		} else {
			l.Println(string(renderedJSON))
		}

		desiredJSON, err := json.Marshal(desiredObject)
		if err != nil {
			l.Println(desiredObject)
		} else {
			l.Println(string(desiredJSON))
		}
		t.Error("rendered object does not match expected object")
	}

}

func Test_renderAztpTemplates(t *testing.T) {
	var data templateData
	data.ZtpImage = "my image"
	objects, err := renderAztpTemplates(data)
	if err != nil {
		t.Error("failed to render aztp templates", err)
	}
	if len(objects) != len(templates) {
		t.Error("object rendering failed")
	}
	// l := log.New(os.Stderr, "", 0)
	// for _, o := range objects {
	// 	l.Println(o)
	// }
}
