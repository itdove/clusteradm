// Copyright Contributors to the Open Cluster Management project
package apply

import (
	"text/template"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Applier struct {
	kubeClient          kubernetes.Interface
	apiExtensionsClient apiextensionsclient.Interface
	dynamicClient       dynamic.Interface
	templateFuncMap     template.FuncMap
}

//ApplierBuilder a builder to build the applier
type ApplierBuilder struct {
	Applier
}

type iApplierBuilder interface {
	Build() Applier
	WithClient(
		kubeClient kubernetes.Interface,
		apiExtensionsClient apiextensionsclient.Interface,
		dynamicClient dynamic.Interface) *ApplierBuilder
	WithTemplateFuncMap(fm template.FuncMap) *ApplierBuilder
}

var _ iApplierBuilder = &ApplierBuilder{}

func (a *ApplierBuilder) Build() Applier {
	return a.Applier
}

//WithClient adds the several clients to the applier
func (a *ApplierBuilder) WithClient(
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface) *ApplierBuilder {
	a.kubeClient = kubeClient
	a.apiExtensionsClient = apiExtensionsClient
	a.dynamicClient = dynamicClient
	return a
}

//WithTemplateFuncMap add template.FuncMap to the applier.
func (a *ApplierBuilder) WithTemplateFuncMap(fm template.FuncMap) *ApplierBuilder {
	a.templateFuncMap = fm
	return a
}
