package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ocmClient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	cluster "open-cluster-management.io/api/cluster/v1"
	policyv1 "open-cluster-management.io/config-policy-controller/api/v1"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	"sigs.k8s.io/yaml"
)

type zapExtractor struct {
	retryTime     time.Duration
	ocmClientset  *ocmClient.Clientset
	kubeClientSet *kubernetes.Clientset
	dynamicClient *dynamic.DynamicClient
	ctx           context.Context
}

func policyResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "policy.open-cluster-management.io",
		Version:  "v1",
		Resource: "policies",
	}
}

func (z *zapExtractor) convertObjects(mc *cluster.ManagedCluster) error {
	clusterName := mc.ObjectMeta.Name
	configMapName := fmt.Sprintf("%s-zap", clusterName)
	// Cluster name is the configmap namespace
	_, err := z.kubeClientSet.CoreV1().ConfigMaps(clusterName).Get(
		z.ctx, configMapName, metav1.GetOptions{})
	if err == nil {
		log.Printf("configmap %s already exists in %s namespace, skip policy extraction", configMapName, clusterName)
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	pl, err := z.dynamicClient.Resource(policyResource()).Namespace(clusterName).List(z.ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	var childPolicies []policiesv1.Policy
	for _, item := range pl.Items {
		var pol policiesv1.Policy
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &pol)
		if err != nil {
			return err
		}
		childPolicies = append(childPolicies, pol)
	}
	var objects []runtime.RawExtension
	for _, pol := range childPolicies {
		for _, template := range pol.Spec.PolicyTemplates {
			o := *template.ObjectDefinition.DeepCopy()
			objects = append(objects, o)
		}
	}
	var directlyAppliedObjects []unstructured.Unstructured
	var wrappedObjects []unstructured.Unstructured
	var objectsToConvert []unstructured.Unstructured
	for _, ob := range objects {
		var pol policyv1.ConfigurationPolicy
		err := json.Unmarshal(ob.DeepCopy().Raw, &pol)
		if err != nil {
			log.Print(err)
			return err
		}
		for _, ot := range pol.Spec.ObjectTemplates {
			var object unstructured.Unstructured
			err = object.UnmarshalJSON(ot.ObjectDefinition.DeepCopy().Raw)
			if err != nil {
				return err
			}
			object.Object["status"] = map[string]interface{}{} // remove status, we can't apply it
			switch object.GetKind() {
			case "PerformanceProfile", "Tuned":
				objectsToConvert = append(objectsToConvert, object)
			case "Namespace", "OperatorGroup", "Subscription", "CatalogSource":
				directlyAppliedObjects = append(directlyAppliedObjects, object)
			default:
				wrappedObjects = append(wrappedObjects, object)

			}
		}
	}

	innerCm := initInnerCm()
	err = wrapObjects(innerCm, wrappedObjects)
	if err != nil {
		return err
	}

	innerCmObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(innerCm)
	if err != nil {
		return err
	}
	var innerCmUns unstructured.Unstructured
	innerCmUns.SetUnstructuredContent(innerCmObj)
	directlyAppliedObjects = append(directlyAppliedObjects, innerCmUns)

	var out *[]unstructured.Unstructured
	if os.Getenv("CONVERT_PERFORMANCE") != "" {
		out, err = convertPerformance(objectsToConvert)
		if err != nil {
			return err
		}
	} else {
		out = &objectsToConvert
	}

	directlyAppliedObjects = append(directlyAppliedObjects, *out...)

	cm := corev1.ConfigMap{}
	cm.ObjectMeta.Name = configMapName
	cm.ObjectMeta.Namespace = clusterName
	cm.Data = map[string]string{}
	err = wrapObjects(&cm, directlyAppliedObjects)
	if err != nil {
		return err
	}
	_, err = z.kubeClientSet.CoreV1().ConfigMaps(clusterName).Create(z.ctx, &cm, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func convertPerformance(in []unstructured.Unstructured) (*[]unstructured.Unstructured, error) {
	var out []unstructured.Unstructured
	return &out, nil
}

func initInnerCm() *corev1.ConfigMap {
	name := os.Getenv("INNER_CONFIGMAP_NAME")
	if name == "" {
		name = "ztp-post-provision"
	}
	namespace := os.Getenv("INNER_CONFIGMAP_NAMESPACE")
	if namespace == "" {
		namespace = "ztp-profile"
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{},
	}
}

func wrapObjects(c *corev1.ConfigMap, objects []unstructured.Unstructured) error {
	for _, item := range objects {
		key := item.GetName()
		out, err := yaml.Marshal(item.Object)
		if err != nil {
			return err
		}
		c.Data[key] = string(out)
	}
	return nil
}

func (z *zapExtractor) handleAdd(e watch.Event) error {
	mc := e.Object.(*cluster.ManagedCluster)
	val, found := mc.ObjectMeta.Labels["ztp-accelerated-provisioning"]
	log.Println("found: ", found, " val: ", val)
	if found && (val == "full" || val == "policies") {
		return z.convertObjects(mc)
	}
	return nil
}

func (z *zapExtractor) watchManagedClusters() error {

	watcher, err := z.ocmClientset.ClusterV1().ManagedClusters().Watch(z.ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Added:
			log.Printf("event %s", event.Type)
			err = z.handleAdd(event)
			if err != nil {
				log.Print(err)
			}
		case watch.Error:
			return fmt.Errorf("watcher error: %+v", event)

		}
	}
	return nil
}

// main
func main() {
	var zap zapExtractor

	zap.retryTime = 30 * time.Second
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalln(err)
	}

	zap.ocmClientset = ocmClient.NewForConfigOrDie(config)
	zap.kubeClientSet = kubernetes.NewForConfigOrDie(config)
	zap.dynamicClient = dynamic.NewForConfigOrDie(config)
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	zap.ctx = ctx
	log.Println("watching ManagedClusters")
	for {
		err := zap.watchManagedClusters()
		if err != nil {
			log.Printf("managed cluster watcher exited: %s. will retry in %v", err, zap.retryTime)
			time.Sleep(zap.retryTime)
		}
	}

}
