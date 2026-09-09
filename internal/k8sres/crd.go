package k8sres

import (
	"context"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// CustomResourceDefinition is read the same way as VerticalPodAutoscaler
// (see vpa.go): generically via the dynamic client, rather than adding
// k8s.io/apiextensions-apiserver's typed client (a much heavier dependency)
// just for a list/get/delete view. CRDs are cluster-scoped.
var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

func listCRDs(ctx context.Context, dyn dynamic.Interface) ([]Row, error) {
	if dyn == nil {
		return nil, fmt.Errorf("dynamic client not configured")
	}
	list, err := dyn.Resource(crdGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, item := range list.Items {
		group, _, _ := unstructured.NestedString(item.Object, "spec", "group")
		kind, _, _ := unstructured.NestedString(item.Object, "spec", "names", "kind")
		scope, _, _ := unstructured.NestedString(item.Object, "spec", "scope")

		versions := "-"
		if vs, found, _ := unstructured.NestedSlice(item.Object, "spec", "versions"); found {
			names := make([]string, 0, len(vs))
			for _, v := range vs {
				if vm, ok := v.(map[string]interface{}); ok {
					if n, ok := vm["name"].(string); ok {
						names = append(names, n)
					}
				}
			}
			if len(names) > 0 {
				sort.Strings(names)
				versions = strings.Join(names, ",")
			}
		}

		rows = append(rows, Row{
			Name:  item.GetName(),
			Cells: []string{item.GetName(), group, versions, kind, scope, age(item.GetCreationTimestamp().Time)},
		})
	}
	sortRows(rows)
	return rows, nil
}

func getCRDYAML(ctx context.Context, dyn dynamic.Interface, name string) (interface{}, error) {
	if dyn == nil {
		return nil, fmt.Errorf("dynamic client not configured")
	}
	obj, err := dyn.Resource(crdGVR).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")
	return obj.Object, nil
}

func deleteCRD(ctx context.Context, dyn dynamic.Interface, name string) error {
	if dyn == nil {
		return fmt.Errorf("dynamic client not configured")
	}
	return dyn.Resource(crdGVR).Delete(ctx, name, metav1.DeleteOptions{})
}
