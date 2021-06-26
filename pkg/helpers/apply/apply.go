// Copyright Contributors to the Open Cluster Management project
package apply

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/asset"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"

	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
)

const (
	ErrorEmptyAssetAfterTemplating = "ERROR_EMPTY_ASSET_AFTER_TEMPLATING"
)

var (
	genericScheme = runtime.NewScheme()
	genericCodecs = serializer.NewCodecFactory(genericScheme)
	genericCodec  = genericCodecs.UniversalDeserializer()
)

//ApplyDeployments applies a appsv1.Deployment template
func ApplyDeployments(
	kubeClient kubernetes.Interface,
	reader asset.ScenarioReader,
	customFuncMap template.FuncMap,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	genericScheme.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	output := make([]string, 0)
	//Render each file
	for _, name := range files {
		deployment, err := ApplyDeployment(kubeClient, reader, customFuncMap, values, dryRun, headerFile, name)
		if err != nil {
			if IsEmptyAsset(err) {
				continue
			}
			return output, err
		}
		output = append(output, deployment)
	}
	return output, nil
}

//ApplyDeployment apply a deployment
func ApplyDeployment(kubeClient kubernetes.Interface,
	reader asset.ScenarioReader,
	customFuncMap template.FuncMap,
	values interface{},
	dryRun bool,
	headerFile string,
	name string) (string, error) {
	recorder := events.NewInMemoryRecorder(helpers.GetExampleHeader())
	deploymentBytes, err := MustTempalteAsset(reader, customFuncMap, values, headerFile, name)
	if err != nil {
		return string(deploymentBytes), err
	}
	output := string(deploymentBytes)
	if dryRun {
		return output, nil
	}
	deployment, sch, err := genericCodec.Decode(deploymentBytes, nil, nil)
	if err != nil {
		return output, fmt.Errorf("%q: %v %v", name, sch, err)
	}
	_, _, err = resourceapply.ApplyDeployment(
		kubeClient.AppsV1(),
		recorder,
		deployment.(*appsv1.Deployment), 0)
	if err != nil {
		return output, fmt.Errorf("%q (%T): %v", name, deployment, err)
	}
	return output, nil
}

//ApplyDirectly applies standard kubernetes resources.
func ApplyDirectly(clients *resourceapply.ClientHolder,
	reader asset.ScenarioReader,
	customFuncMap template.FuncMap,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	if dryRun {
		return MustTemplateAssets(reader, customFuncMap, values, headerFile, files...)
	}
	recorder := events.NewInMemoryRecorder(helpers.GetExampleHeader())
	output := make([]string, 0)
	//Apply resources
	resourceResults := resourceapply.ApplyDirectly(clients, recorder, func(name string) ([]byte, error) {
		out, err := MustTempalteAsset(reader, customFuncMap, values, headerFile, name)
		if err != nil {
			return nil, err
		}
		output = append(output, string(out))
		return out, nil
	}, files...)
	//Check errors
	for _, result := range resourceResults {
		if result.Error != nil && !IsEmptyAsset(result.Error) {
			return output, fmt.Errorf("%q (%T): %v", result.File, result.Type, result.Error)
		}
	}
	return output, nil
}

//ApplyCustomResouces applies custom resources
func ApplyCustomResouces(dynamicClient dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
	reader asset.ScenarioReader,
	customFuncMap template.FuncMap,
	values interface{},
	dryRun bool,
	headerFile string,
	files ...string) ([]string, error) {
	output := make([]string, 0)
	for _, name := range files {
		asset, err := ApplyCustomResouce(dynamicClient, discoveryClient, reader, customFuncMap, values, dryRun, headerFile, name)
		if err != nil {
			if IsEmptyAsset(err) {
				continue
			}
			return output, err
		}
		output = append(output, string(asset))
	}
	return output, nil
}

//ApplyCustomResouces applies custom resources
func ApplyCustomResouce(dynamicClient dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
	reader asset.ScenarioReader,
	customFuncMap template.FuncMap,
	values interface{},
	dryRun bool,
	headerFile string,
	name string) (string, error) {
	asset, err := MustTempalteAsset(reader, customFuncMap, values, headerFile, name)
	output := string(asset)
	if err != nil {
		return output, err
	}
	if dryRun {
		return output, nil
	}
	u, err := bytesToUnstructured(reader, asset)
	if err != nil {
		return output, err
	}
	gvks, _, err := genericScheme.ObjectKinds(u)
	if err != nil {
		return output, err
	}
	gvk := gvks[0]
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return output, err
	}
	dr := dynamicClient.Resource(mapping.Resource)
	ug, err := dr.Namespace(u.GetNamespace()).Get(context.TODO(), u.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = dr.Namespace(u.GetNamespace()).
				Create(context.TODO(), u, metav1.CreateOptions{})
		}
	} else {
		u.SetResourceVersion(ug.GetResourceVersion())
		_, err = dr.Namespace(u.GetNamespace()).
			Update(context.TODO(), u, metav1.UpdateOptions{})
	}
	if err != nil {
		return output, err
	}
	return output, nil
}

//bytesToUnstructured converts an asset to unstructured.
func bytesToUnstructured(reader asset.ScenarioReader, asset []byte) (*unstructured.Unstructured, error) {
	j, err := reader.ToJSON(asset)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(j, nil, u)
	if err != nil {
		//In case it is not a kube yaml
		if !runtime.IsMissingKind(err) {
			return nil, err
		}
	}
	return u, nil
}

//getTemplate generate the template for rendering.
func getTemplate(templateName string, customFuncMap template.FuncMap) *template.Template {
	tmpl := template.New(templateName).
		Option("missingkey=zero").
		Funcs(FuncMap())
	tmpl = tmpl.Funcs(TemplateFuncMap(tmpl)).
		Funcs(sprig.TxtFuncMap())
	if customFuncMap != nil {
		tmpl = tmpl.Funcs(customFuncMap)
	}
	return tmpl
}

//MustTemplateAssets render list of files
func MustTemplateAssets(reader asset.ScenarioReader, customFuncMap template.FuncMap, values interface{}, headerFile string, files ...string) ([]string, error) {
	output := make([]string, 0)
	for _, name := range files {
		deploymentBytes, err := MustTempalteAsset(reader, customFuncMap, values, headerFile, name)
		if err != nil {
			if IsEmptyAsset(err) {
				continue
			}
			return output, err
		}
		output = append(output, string(deploymentBytes))
	}
	return output, nil
}

//MustTempalteAsset generates textual output for a template file name.
//The headerfile will be added to each file.
//Usually it contains nested template definitions as described https://golang.org/pkg/text/template/#hdr-Nested_template_definitions
//This allows to add functions which can be use in each file.
//The values object will be used to render the template
func MustTempalteAsset(reader asset.ScenarioReader, customFuncMap template.FuncMap, values interface{}, headerFile, name string) ([]byte, error) {
	tmpl := getTemplate(name, customFuncMap)
	h := []byte{}
	var err error
	if headerFile != "" {
		h, err = reader.Asset(headerFile)
		if err != nil {
			return nil, err
		}
	}
	b, err := reader.Asset(name)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	tmplParsed, err := tmpl.Parse(string(b))
	if err != nil {
		return nil, err
	}
	tmplParsed, err = tmplParsed.Parse(string(h))
	if err != nil {
		return nil, err
	}

	err = tmplParsed.Execute(&buf, values)
	if err != nil {
		return nil, err
	}

	//If the content is empty after rendering then returns an ErrorEmptyAssetAfterTemplating error.
	if isEmpty(buf.Bytes()) {
		return nil, fmt.Errorf("asset %s becomes %s", name, ErrorEmptyAssetAfterTemplating)
	}

	return buf.Bytes(), nil
}

//isEmpty check if a content is empty after removing comments and blank lines.
func isEmpty(body []byte) bool {
	//Remove comments
	re := regexp.MustCompile("#.*")
	bodyNoComment := re.ReplaceAll(body, nil)
	//Remove blank lines
	trim := strings.TrimSuffix(string(bodyNoComment), "\n")
	trim = strings.TrimSpace(trim)

	return len(trim) == 0
}

//IsEmptyAsset returns true if the error is ErrorEmptyAssetAfterTemplating
func IsEmptyAsset(err error) bool {
	return strings.Contains(err.Error(), ErrorEmptyAssetAfterTemplating)
}

func WriteOutput(fileName string, output []string) (err error) {
	if fileName == "" {
		return nil
	}
	f, err := os.OpenFile(filepath.Clean(fileName), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, s := range output {
		_, err := f.WriteString(fmt.Sprintf("%s\n---\n", s))
		if err != nil {
			return err
		}
	}
	return err
}
