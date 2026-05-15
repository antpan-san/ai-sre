package go_runtime

import "testing"

func TestParseNamespacedTarget(t *testing.T) {
	ns, name, err := ParseNamespacedTarget("memleak-demo/memleak-demo-75744877fb-6cwcx")
	if err != nil || ns != "memleak-demo" || name != "memleak-demo-75744877fb-6cwcx" {
		t.Fatalf("got %q %q err=%v", ns, name, err)
	}
	_, name2, err := ParseNamespacedTarget("api")
	if err != nil || name2 != "api" {
		t.Fatalf("single name: %q err=%v", name2, err)
	}
}

func TestNewK8sResourceRef(t *testing.T) {
	ref, err := NewK8sResourceRef(K8sKindDeployment, "default/api")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Kind != K8sKindDeployment || ref.Namespace != "default" || ref.Name != "api" {
		t.Fatalf("%+v", ref)
	}
	if ref.String() != "deployment/default/api" {
		t.Fatalf("string %q", ref.String())
	}
}

func TestK8sResourceRefHasWorkloadPods(t *testing.T) {
	dep := K8sResourceRef{Kind: K8sKindDeployment}
	if !dep.hasWorkloadPods() {
		t.Fatal("deployment should have pods")
	}
	ing := K8sResourceRef{Kind: K8sKindIngress}
	if ing.hasWorkloadPods() {
		t.Fatal("ingress should not use proc path")
	}
}

func TestPodRank(t *testing.T) {
	if podRank(podListItem{Phase: "Running", Ready: true}) >= podRank(podListItem{Phase: "Pending", Ready: false, WaitingReason: "CrashLoopBackOff"}) {
		t.Fatal("unhealthy pod should rank higher")
	}
}
