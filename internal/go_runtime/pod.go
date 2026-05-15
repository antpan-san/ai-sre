package go_runtime

import "fmt"

type PodPIDResolver interface {
	Resolve(namespace, pod, container string) (int, error)
}

type UnimplementedPodPIDResolver struct{}

func (UnimplementedPodPIDResolver) Resolve(namespace, pod, container string) (int, error) {
	return 0, fmt.Errorf("pod to host PID resolver is not implemented yet: namespace=%q pod=%q container=%q", namespace, pod, container)
}
