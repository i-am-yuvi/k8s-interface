package k8sinterface

import (
	"fmt"
	"strings"

	wlidpkg "github.com/armosec/utils-k8s-go/wlid"
	"github.com/kubescape/k8s-interface/workloadinterface"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

type IWorkload workloadinterface.IWorkload

func (k8sAPI *KubernetesApi) ListAllWorkload() ([]IWorkload, error) {
	workloads := []IWorkload{}
	var errs error
	for resource := range GetResourceGroupMapping() {
		groupVersionResource, err := GetGroupVersionResource(resource)
		if err != nil {
			errs = fmt.Errorf("%v\n%s", errs, err.Error())
			continue
		}
		w, err := k8sAPI.ListWorkloads(&groupVersionResource, "", nil, nil)
		if err != nil {
			errs = fmt.Errorf("%v\n%s", errs, err.Error())
			continue
		}
		if len(w) == 0 {
			continue
		}
		workloads = append(workloads, w...)
	}
	return workloads, errs
}

func (k8sAPI *KubernetesApi) GetWorkloadByWlid(wlid string) (IWorkload, error) {
	return k8sAPI.GetWorkload(wlidpkg.GetNamespaceFromWlid(wlid), wlidpkg.GetKindFromWlid(wlid), wlidpkg.GetNameFromWlid(wlid))
}

func (k8sAPI *KubernetesApi) GetWorkload(namespace, kind, name string) (IWorkload, error) {
	groupVersionResource, err := GetGroupVersionResource(kind)
	if err != nil {
		return nil, err
	}

	w, err := k8sAPI.ResourceInterface(&groupVersionResource, namespace).Get(k8sAPI.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to GET resource, kind: '%s', namespace: '%s', name: '%s', reason: %s", kind, namespace, name, err.Error())
	}
	return workloadinterface.NewWorkloadObj(w.Object), nil
}

func (k8sAPI *KubernetesApi) ListWorkloads2(namespace, kind string) ([]IWorkload, error) {
	groupVersionResource, err := GetGroupVersionResource(kind)
	if err != nil {
		return nil, err
	}

	uList, err := k8sAPI.ResourceInterface(&groupVersionResource, namespace).List(k8sAPI.Context, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to LIST resources, reason: %s", err.Error())
	}
	workloads := make([]IWorkload, len(uList.Items))
	for i := range uList.Items {
		workloads[i] = workloadinterface.NewWorkloadObj(uList.Items[i].Object)
	}
	return workloads, nil
}

func (k8sAPI *KubernetesApi) ListWorkloads(groupVersionResource *schema.GroupVersionResource, namespace string, podLabels, fieldSelector map[string]string) ([]IWorkload, error) {
	listOptions := metav1.ListOptions{}
	if len(podLabels) > 0 {
		set := labels.Set(podLabels)
		listOptions.LabelSelector = SelectorToString(set)
	}
	if len(fieldSelector) > 0 {
		set := labels.Set(fieldSelector)
		listOptions.FieldSelector = SelectorToString(set)
	}
	uList, err := k8sAPI.ResourceInterface(groupVersionResource, namespace).List(k8sAPI.Context, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to LIST resources, reason: %s", err.Error())
	}
	workloads := make([]IWorkload, len(uList.Items))
	for i := range uList.Items {
		workloads[i] = workloadinterface.NewWorkloadObj(uList.Items[i].Object)
	}
	return workloads, nil
}

func (k8sAPI *KubernetesApi) DeleteWorkloadByWlid(wlid string) error {
	groupVersionResource, err := GetGroupVersionResource(wlidpkg.GetKindFromWlid(wlid))
	if err != nil {
		return err
	}
	err = k8sAPI.ResourceInterface(&groupVersionResource, wlidpkg.GetNamespaceFromWlid(wlid)).Delete(k8sAPI.Context, wlidpkg.GetNameFromWlid(wlid), metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to DELETE resource, workloadID: '%s', reason: %s", wlid, err.Error())
	}
	return nil
}

func (k8sAPI *KubernetesApi) CreateWorkload(workload IWorkload) (IWorkload, error) {
	groupVersionResource, err := GetGroupVersionResource(workload.GetKind())
	if err != nil {
		return nil, err
	}
	obj, err := workload.ToUnstructured()
	if err != nil {
		return nil, err
	}
	w, err := k8sAPI.ResourceInterface(&groupVersionResource, workload.GetNamespace()).Create(k8sAPI.Context, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to CREATE resource, workload: '%s', reason: %s", workload.ToString(), err.Error())
	}
	return workloadinterface.NewWorkloadObj(w.Object), nil
}

func (k8sAPI *KubernetesApi) UpdateWorkload(workload IWorkload) (IWorkload, error) {
	groupVersionResource, err := GetGroupVersionResource(workload.GetKind())
	if err != nil {
		return nil, err
	}

	obj, err := workload.ToUnstructured()
	if err != nil {
		return nil, err
	}

	w, err := k8sAPI.ResourceInterface(&groupVersionResource, workload.GetNamespace()).Update(k8sAPI.Context, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to UPDATE resource, workload: '%s', reason: %s", workload.ToString(), err.Error())
	}
	return workloadinterface.NewWorkloadObj(w.Object), nil
}

func (k8sAPI *KubernetesApi) GetNamespace(ns string) (IWorkload, error) {
	groupVersionResource, err := GetGroupVersionResource("namespace")
	if err != nil {
		return nil, err
	}
	w, err := k8sAPI.DynamicClient.Resource(groupVersionResource).Get(k8sAPI.Context, ns, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: '%s', reason: %s", ns, err.Error())
	}
	return workloadinterface.NewWorkloadObj(w.Object), nil
}

func (k8sAPI *KubernetesApi) ResourceInterface(resource *schema.GroupVersionResource, namespace string) dynamic.ResourceInterface {
	if IsNamespaceScope(resource) {
		return k8sAPI.DynamicClient.Resource(*resource).Namespace(namespace)
	}
	return k8sAPI.DynamicClient.Resource(*resource)
}

func (k8sAPI *KubernetesApi) CalculateWorkloadParentRecursive(workload IWorkload) (string, string, error) {
	ownerReferences, err := workload.GetOwnerReferences() // OwnerReferences in workload
	if err != nil {
		return workload.GetKind(), workload.GetName(), err
	}
	if len(ownerReferences) == 0 {
		return workload.GetKind(), workload.GetName(), nil // parent found
	}
	ownerReference := ownerReferences[0]

	parentWorkload, err := k8sAPI.GetWorkload(workload.GetNamespace(), ownerReference.Kind, ownerReference.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found in resourceMap") { // if parent is RCD
			return workload.GetKind(), workload.GetName(), nil // parent found
		}
		return workload.GetKind(), workload.GetName(), err
	}
	return k8sAPI.CalculateWorkloadParentRecursive(parentWorkload)
}
