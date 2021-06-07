// Copyright Contributors to the Open Cluster Management project

package apply

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	"k8s.io/client-go/discovery/cached/memory"
	fakekubernetes "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/restmapper"
	"k8s.io/kubectl/pkg/scheme"
)

func Test_Discory(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(clusterv1.SchemeGroupVersion, &clusterv1.ManagedCluster{})
	err := fakekubernetes.AddToScheme(s)
	if err != nil {
		t.Error(err)
	}
	kubeClient := fakekubernetes.NewSimpleClientset()
	fmt.Printf("Types: %v\n", s.AllKnownTypes())
	discoveryClient := kubeClient.Discovery()
	groups, err := discoveryClient.ServerGroups()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Groups: %v\n", groups)
	gvk := schema.GroupVersionKind{
		Group:   "cluster.open-cluster-management.io",
		Version: "v1",
		Kind:    "ManagedCluster",
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	t.Log(mapper)
	_, err = mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		t.Error(err)
	}

}
