// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"path/filepath"
	"testing"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	"github.com/spf13/cobra"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var testDir = filepath.Join("test", "unit")

func TestOptions_complete(t *testing.T) {
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		clusterName             string
		values                  map[string]interface{}
	}
	type args struct {
		cmd  *cobra.Command
		args []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Failed, bad valuesPath",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: "bad-values-path.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "Failed, empty values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: filepath.Join(testDir, "values-empty.yaml"),
				},
			},
			wantErr: true,
		},
		{
			name: "Sucess, with values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{
					ValuesPath: filepath.Join(testDir, "values-fake.yaml"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				clusterName:             tt.fields.clusterName,
				values:                  tt.fields.values,
			}
			if err := o.complete(tt.args.cmd, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("Options.complete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptions_validate(t *testing.T) {
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		clusterName             string
		values                  map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Success all info in values",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Failed name missing",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values:                  map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "Failed name empty",
			fields: fields{
				applierScenariosOptions: &applierscenarios.ApplierScenariosOptions{},
				values: map[string]interface{}{
					"managedCluster": map[string]interface{}{
						"name": "",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				clusterName:             tt.fields.clusterName,
				values:                  tt.fields.values,
			}
			if err := o.validate(); (err != nil) != tt.wantErr {
				t.Errorf("Options.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptions_runWithClient(t *testing.T) {
	type fields struct {
		applierScenariosOptions *applierscenarios.ApplierScenariosOptions
		clusterName             string
		values                  map[string]interface{}
	}
	type args struct {
		client crclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				applierScenariosOptions: tt.fields.applierScenariosOptions,
				clusterName:             tt.fields.clusterName,
				values:                  tt.fields.values,
			}
			if err := o.runWithClient(tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("Options.runWithClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
