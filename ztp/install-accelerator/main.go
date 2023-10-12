package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	v1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/yaml"
)

type StatusEvent struct {
	err        error
	status     string
	apiVersion string
	kind       string
	name       string
	namespace  string
}

func clusterVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "config.openshift.io",
		Version:  "v1",
		Resource: "clusterversions",
	}
}

func clusterOperatorResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "config.openshift.io",
		Version:  "v1",
		Resource: "clusteroperators",
	}
}

// IsStatusConditionPresentAndTrue checks for a specific status condition on a resource.
func IsStatusConditionPresentAndTrue(client *dynamic.DynamicClient, gvr schema.GroupVersionResource,
	name string, conditionType string) (found bool, positive bool, err error) {

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	obj, err := client.Resource(gvr).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return false, false, err
	}
	return isObjectStatusConditionPresentAndTrue(obj, conditionType)
}

func isObjectStatusConditionPresentAndTrue(obj *unstructured.Unstructured, conditionType string) (found bool, positive bool, err error) {
	var conditions []v1.Condition
	var stat map[string]interface{}
	var data []byte

	stat = obj.Object["status"].(map[string]interface{})

	data, err = json.Marshal(stat["conditions"])
	if err != nil {
		return false, false, err
	}

	err = json.Unmarshal(data, &conditions)
	if err != nil {
		return false, false, err
	}
	condition := meta.FindStatusCondition(conditions, conditionType)
	if condition == nil {
		return false, false, nil
	} else {
		found = true
	}
	if condition.Status == v1.ConditionTrue {
		positive = true
	}
	return
}

// waitForStartCondition blocks until either start or end condition occurs.
// Start condition - OLM is available and version is not progressing
// End condition - clusterversion is available and not progressing
// Error is returned upon the end condition.
func waitForStartCondition(client *dynamic.DynamicClient) error {
	for {
		var versionFound, versionProgressing bool
		var olmAvailable bool
		var err error
		versionFound, versionProgressing, err = IsStatusConditionPresentAndTrue(
			client, clusterVersionResource(), "version", "Progressing")
		if err != nil {
			log.Println(err)
			goto continueWaitingForStart
		}
		_, olmAvailable, err = IsStatusConditionPresentAndTrue(
			client, clusterOperatorResource(),
			"operator-lifecycle-manager-packageserver", "Available")
		if err != nil {
			log.Println(err)
			goto continueWaitingForStart
		}
		if versionFound && !versionProgressing {
			return fmt.Errorf("cluster version is no longer progressing - exiting")
		}
		if versionProgressing && olmAvailable {
			return nil
		}

	continueWaitingForStart:
		waitTime := 30 * time.Second
		log.Print("start condition is not reached, wait another ", waitTime)
		time.Sleep(waitTime)
	}
}

// extracts a list of manifests from configmap and returns them as a slice of unstructured
func extractManifests(ctx context.Context, config *rest.Config) ([]unstructured.Unstructured, error) {
	retryTime := 30 * time.Second
	name := os.Getenv("CONFIGMAP_NAME")
	if name == "" {
		name = "ztp-post-provision"
	}
	namespace := os.Getenv("CONFIGMAP_NAMESPACE")
	if namespace == "" {
		namespace = "ztp-profile"
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	var manifests []unstructured.Unstructured
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, v1.GetOptions{})
			if err != nil {
				goto waitForCm
			}
			err = cmToManifests(cm, &manifests)
			if err != nil {
				return nil, err
			}
			goto done
		waitForCm:
			log.Printf("waiting %s for configmap to appear", retryTime)
			time.Sleep(retryTime)
		}
	}
done:
	return manifests, nil
}

// cmToManifests extracts manifests from configmap
func cmToManifests(cm *corev1.ConfigMap, manifests *[]unstructured.Unstructured) error {
	for _, v := range cm.Data {
		jData, err := yaml.YAMLToJSON([]byte(v))
		if err != nil {
			return err
		}

		var data unstructured.Unstructured
		err = data.UnmarshalJSON([]byte(jData))
		if err != nil {
			return err
		}
		*manifests = append(*manifests, data)
	}
	return nil
}

// applyManifest applies arbitrary manifests
func applyManifest(ctx context.Context, wg *sync.WaitGroup, channel chan StatusEvent, config *rest.Config, obj unstructured.Unstructured) {
	defer wg.Done()
	retryTime := 30 * time.Second

	ns := obj.GetNamespace()
	name := obj.GetName()
	apiVersion := obj.GetAPIVersion()
	kind := obj.GetKind()

	ev := StatusEvent{
		err:        nil,
		status:     "starting",
		apiVersion: apiVersion,
		kind:       kind,
		name:       name,
		namespace:  ns,
	}
	channel <- ev

	disClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		ev.err = fmt.Errorf("error creating discovery client, %v", err)
		ev.status = "fail"
		channel <- ev
		return

	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		ev.err = fmt.Errorf("error creating dynamic client, %v", err)
		ev.status = "fail"
		channel <- ev
		return
	}
	gv := strings.Split(apiVersion, "/")
	var mapper *restmapper.DeferredDiscoveryRESTMapper
	var mapping *meta.RESTMapping
	for {
		select {
		case <-ctx.Done():
			log.Printf("cancelled application of %s %s %s %s", apiVersion, kind, name, ns)
			return
		default:
			mapper = restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disClient))
			mapping, err = mapper.RESTMapping(schema.GroupKind{
				Group: gv[0],
				Kind:  kind,
			}, gv[1])
			if err != nil {
				log.Printf("can't find GVR, will retry in %s, %v", retryTime, err) //this will fail until CRD is applied
				time.Sleep(retryTime)
				continue
			}

			resource := schema.GroupVersionResource{
				Group:    mapping.Resource.Group,
				Version:  mapping.Resource.Version,
				Resource: mapping.Resource.Resource,
			}

			if ns != "" {
				_, err = dynamicClient.Resource(resource).Namespace(ns).Create(ctx, &obj, v1.CreateOptions{})
			} else {
				_, err = dynamicClient.Resource(resource).Create(ctx, &obj, v1.CreateOptions{})
			}
			if err != nil && !errors.IsAlreadyExists(err) {
				log.Printf("failed to apply resource, will retry in %s, %v", retryTime, err)
				time.Sleep(retryTime)
				continue
			}
			ev.err = nil
			ev.status = "success"
			channel <- ev
			return
		}
	}
}

// Approves InstallPlans on the all namespaces, until cancelled
func approveInstallPlans(ctx context.Context, wg *sync.WaitGroup, channel chan StatusEvent, config *rest.Config) {

	defer wg.Done()
	retryTime := 30 * time.Second
	ev := StatusEvent{
		err: nil,
	}

	client, err := operatorsv1alpha1.NewForConfig(config)
	if err != nil {
		ev.err = fmt.Errorf("error creating operators client, %v", err)
		ev.status = "fail"
		channel <- ev
		return
	}
	for {
		watcher, err := client.InstallPlans("").Watch(ctx, v1.ListOptions{})
		if err != nil {
			log.Print(context.Canceled, err)
			if err == context.Canceled {
				return
			}
			log.Printf("watch installplans error, will retry in %s, %v", retryTime, err)
			time.Sleep(retryTime)
			continue
		}
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				item := event.Object.(*v1alpha1.InstallPlan)
				log.Print("installplan watch: ", item.Name, " ", item.Namespace, " ", event.Type)
				if !item.Spec.Approved {
					log.Printf("approving installplan %s in namespace %s", item.Name, item.Namespace)
					_, err = client.InstallPlans(item.Namespace).Patch(ctx, item.Name, types.MergePatchType,
						[]byte("{\"spec\":{\"approved\":true}}"), v1.PatchOptions{})
					if err != nil {
						log.Printf("update installplans error, will retry, %v", err)
						continue
					}
				}
			case watch.Error:
				log.Print(context.Canceled, err)
				if ctx.Err() == context.Canceled {
					log.Println("stopping installplan watcher")
					return
				}
				log.Print(ctx.Err())
				goto cont
			}
		}
	cont:
	}
}

// applyManifests applies extracted manifests
func applyManifests(ctx context.Context, wg *sync.WaitGroup, channel chan StatusEvent, config *rest.Config) {
	defer wg.Done()
	manifests, err := extractManifests(ctx, config)
	if err != nil {
		channel <- StatusEvent{
			err:    err,
			status: "fatal",
		}
	}
	for _, manifest := range manifests {
		wg.Add(1)
		go applyManifest(ctx, wg, channel, config, manifest)
	}

}

// checkDelayExit checks if exit delay is configured and waits configured amount
func checkDelayExit() {
	extension, err := time.ParseDuration(os.Getenv("END_CONDITION_EXTENSION_TIME"))
	if err == nil && extension != 0 {
		log.Printf("delaying exit by %v", extension)
		time.Sleep(extension)
	}
}

// main
func main() {
	checkStartCondition := flag.Bool("override", false, "Block until start condition occurs")
	flag.Parse()
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Panic(err)
	}
	// Dynamic client - for applying and monitoring arbitrary manifests.
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Panic(err)
	}
	if !*checkStartCondition {
		err = waitForStartCondition(dynamicClient)
		if err != nil {
			log.Println("end condition determined when waiting for start condition - exiting")
			os.Exit(1)
		}
	}
	eventChannel := make(chan StatusEvent, 1)
	log.Println("starting installation of custom resources")
	ctx, ctxCancel := context.WithCancel(context.Background())
	tickerAbortCheck := time.NewTicker(time.Second * 30)
	defer tickerAbortCheck.Stop()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go applyManifests(ctx, &wg, eventChannel, config)
	go approveInstallPlans(ctx, &wg, eventChannel, config)
	const maxRetries = 20
	var retries int
	var countNotDone int
	allDone := false
	status := map[string]string{}
	for {
		select {
		case notification := <-eventChannel:
			key := strings.Join([]string{notification.apiVersion, notification.kind, notification.name, notification.namespace}, " ")
			log.Println(notification.status, notification.apiVersion, notification.kind, notification.err)
			switch notification.status {
			case "fatal":
				log.Fatal(notification.err)
			case "starting":
				status[key] = "not done"
				countNotDone += 1
			case "success":
				status[key] = "done"
				countNotDone -= 1
			}
			if countNotDone == 0 {
				allDone = true
				tickerAbortCheck.Stop()
				checkDelayExit()
				ctxCancel()
			}

		case <-ctx.Done():
			wg.Wait()
			log.Println("all done ", allDone, " status ", status)
			if !allDone {
				os.Exit(1)

			}
			os.Exit(0)

		case <-tickerAbortCheck.C:
			versionFound, versionProgressing, err := IsStatusConditionPresentAndTrue(
				dynamicClient, clusterVersionResource(), "version", "Progressing")
			if err != nil {
				log.Println(err, "will retry")
				retries += 1
				if retries >= maxRetries {
					log.Printf("can't read clusterversion status, exiting after %d retries", retries)
					ctxCancel()
				}
				continue
			}
			retries = 0
			if versionFound && !versionProgressing {
				checkDelayExit()
				log.Printf("stop condition - cancelling all jobs and exiting")
				ctxCancel()
			}
			continue
		}
	}
}
