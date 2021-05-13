// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"fmt"
	"path/filepath"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/detach/cluster/scenario"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var example = `
# Detach a cluster
%[1]s detach cluster --values values.yaml

# Detach a cluster with overwritting the cluster name
%[1]s detach cluster --values values.yaml --name mycluster
`

const (
	scenarioDirectory = "scale"
)

var valuesTemplatePath = filepath.Join(scenarioDirectory, "values-template.yaml")

// NewCmd ...
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "scale a cluster",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}
			if err := o.run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.SetUsageTemplate(applierscenarios.UsageTempate(cmd, scenario.GetApplierScenarioResourcesReader(), valuesTemplatePath))
	cmd.Flags().StringVar(&o.clusterName, "name", "", "Name of the cluster to scale")
	cmd.Flags().IntVar(&o.replicas, "replicas", -1, "Number of replicas")

	o.applierScenariosOptions.AddFlags(cmd.Flags())
	o.applierScenariosOptions.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}
