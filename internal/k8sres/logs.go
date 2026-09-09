package k8sres

import (
	"context"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Containers lists the container names in a pod (init containers first,
// then regular containers), for the container-picker shown before tailing
// logs when a pod has more than one.
func Containers(clientset *kubernetes.Clientset, namespace, pod string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	p, err := clientset.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, c := range p.Spec.InitContainers {
		names = append(names, c.Name)
	}
	for _, c := range p.Spec.Containers {
		names = append(names, c.Name)
	}
	return names, nil
}

// StreamLogs opens a following log stream for one container. The caller
// owns the returned ReadCloser and must Close it (e.g. when the log view is
// dismissed) to stop the underlying watch. tailLines bounds how much
// history is fetched before following.
func StreamLogs(ctx context.Context, clientset *kubernetes.Clientset, namespace, pod, container string, tailLines int64) (io.ReadCloser, error) {
	opts := &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
		TailLines: &tailLines,
	}
	return clientset.CoreV1().Pods(namespace).GetLogs(pod, opts).Stream(ctx)
}
