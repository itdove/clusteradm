// Copyright Contributors to the Open Cluster Management project
package init

import (
	"fmt"
	"time"

	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"

	"github.com/spf13/cobra"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	o.values = Values{
		Hub: Hub{
			TokenID:     helpers.RandStringRunes_az09(6),
			TokenSecret: helpers.RandStringRunes_az09(16),
		},
	}
	return nil
}

func (o *Options) validate() error {
	if o.force {
		return nil
	}
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	apiExtensionsClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	installed, err := helpers.IsClusterManagerInstalled(apiExtensionsClient)
	if err != nil {
		return err
	}
	if installed {
		return fmt.Errorf("hub already initialized")
	}
	return nil
}

func (o *Options) run() error {
	token := fmt.Sprintf("%s.%s", o.values.Hub.TokenID, o.values.Hub.TokenSecret)
	output := make([]string, 0)
	reader := scenario.GetScenarioResourcesReader()

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	applierBuilder := &apply.ApplierBuilder{}
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	files := []string{
		"init/namespace.yaml",
	}
	if o.useBootstrapToken {
		files = append(files,
			"init/bootstrap-token-secret.yaml",
			"init/bootstrap_cluster_role.yaml",
			"init/bootstrap_cluster_role_binding.yaml",
		)
	} else {
		files = append(files,
			"init/bootstrap_sa.yaml",
			"init/bootstrap_cluster_role.yaml",
			"init/bootstrap_sa_cluster_role_binding.yaml",
		)
	}

	files = append(files,
		"init/clustermanager_cluster_role.yaml",
		"init/clustermanager_cluster_role_binding.yaml",
		"init/clustermanagers.crd.yaml",
		"init/clustermanager_sa.yaml",
	)

	out, err := applier.ApplyDirectly(reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return err
	}
	output = append(output, out...)

	//if service-account wait for the sa secret
	if !o.useBootstrapToken && !o.ClusteradmFlags.DryRun {
		err = wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
			return waitForBootstrapToken(kubeClient)
		})
		if err != nil {
			return err
		}
		token, err = helpers.GetBootstrapTokenFromSA(kubeClient)
		if err != nil {
			return err
		}
	}

	out, err = applier.ApplyDeployments(reader, o.values, o.ClusteradmFlags.DryRun, "", "init/operator.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	if !o.ClusteradmFlags.DryRun {
		b := retry.DefaultBackoff
		b.Duration = 200 * time.Millisecond
		err = helpers.WaitCRDToBeReady(apiExtensionsClient, "clustermanagers.operator.open-cluster-management.io", b)
		if err != nil {
			return err
		}
	}

	out, err = applier.ApplyCustomResouces(reader, o.values, o.ClusteradmFlags.DryRun, "", "init/clustermanager.cr.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return nil
	}

	fmt.Printf("please log on spoke and run:\n%s join --hub-token %s --hub-apiserver %s --cluster-name <cluster_name>\n",
		helpers.GetExampleHeader(),
		token,
		restConfig.Host,
	)

	return apply.WriteOutput(o.outputFile, output)
}

func waitForBootstrapToken(kubeClient kubernetes.Interface) (bool, error) {
	_, err := helpers.GetBootstrapTokenFromSA(kubeClient)
	switch {
	case errors.IsNotFound(err):
		return false, err
	case err != nil:
		return false, err
	}
	return true, nil
}
