package k8sres

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// VerticalPodAutoscaler is a CRD from the separate autoscaler/vertical-pod-
// autoscaler project, not core Kubernetes — there's no typed client for it
// in client-go, so it's read generically via the dynamic client instead of
// pulling in that project's client as a dependency. A cluster without the
// VPA CRD installed simply fails this List/Get/Delete like any other
// missing API, which the dashboard already surfaces as a normal error.
var vpaGVR = schema.GroupVersionResource{
	Group:    "autoscaling.k8s.io",
	Version:  "v1",
	Resource: "verticalpodautoscalers",
}

func listVPAs(ctx context.Context, dyn dynamic.Interface, ns string) ([]Row, error) {
	if dyn == nil {
		return nil, fmt.Errorf("dynamic client not configured")
	}
	list, err := dyn.Resource(vpaGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, item := range list.Items {
		mode, _, _ := unstructured.NestedString(item.Object, "spec", "updatePolicy", "updateMode")
		if mode == "" {
			mode = "Auto" // the VPA API defaults updateMode to Auto when unset
		}
		target, _, _ := unstructured.NestedString(item.Object, "spec", "targetRef", "name")
		if target == "" {
			target = "-"
		}
		cpu, mem := vpaRecommendation(item.Object)

		rows = append(rows, Row{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
			Cells:     []string{item.GetName(), mode, target, cpu, mem, age(item.GetCreationTimestamp().Time)},
		})
	}
	sortRows(rows)
	return rows, nil
}

// vpaRecommendation pulls the target CPU/memory for the first container
// recommendation, if the VPA's recommender has produced one yet.
func vpaRecommendation(obj map[string]interface{}) (cpu, mem string) {
	cpu, mem = "-", "-"
	recs, found, _ := unstructured.NestedSlice(obj, "status", "recommendation", "containerRecommendations")
	if !found || len(recs) == 0 {
		return
	}
	rec, ok := recs[0].(map[string]interface{})
	if !ok {
		return
	}
	target, found, _ := unstructured.NestedStringMap(rec, "target")
	if !found {
		return
	}
	if v, ok := target["cpu"]; ok {
		cpu = v
	}
	if v, ok := target["memory"]; ok {
		mem = v
	}
	return
}

func getVPAYAML(ctx context.Context, dyn dynamic.Interface, ns, name string) (interface{}, error) {
	if dyn == nil {
		return nil, fmt.Errorf("dynamic client not configured")
	}
	obj, err := dyn.Resource(vpaGVR).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")
	return obj.Object, nil
}

func deleteVPA(ctx context.Context, dyn dynamic.Interface, ns, name string) error {
	if dyn == nil {
		return fmt.Errorf("dynamic client not configured")
	}
	return dyn.Resource(vpaGVR).Namespace(ns).Delete(ctx, name, metav1.DeleteOptions{})
}
