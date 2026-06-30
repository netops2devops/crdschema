package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var outputDir string
	var kubeconfig string
	var crdName string
	var all bool

	rootCmd := &cobra.Command{
		Use:   "getschema",
		Short: "Download JSON schemas for Kubernetes CRDs",
		Long: `getschema fetches Custom Resource Definitions from the active Kubernetes cluster
and writes their OpenAPI v3 JSON schemas to disk under:

  <output-dir>/<group>/<version>/<Kind>.json

Use --all to download every CRD, or --crd to target a specific one.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("all") && !cmd.Flags().Changed("crd") {
				return cmd.Help()
			}
			return run(outputDir, kubeconfig, crdName, all)
		},
	}

	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "./schemas", "Directory where schemas will be stored")
	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "Path to kubeconfig file")
	rootCmd.Flags().StringVar(&crdName, "crd", "", "Download schema for a specific CRD (full name like 'widgets.example.com' or kind like 'Widget')")
	rootCmd.Flags().BoolVar(&all, "all", false, "Download schemas for all CRDs (overrides --crd when both are set)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(outputDir, kubeconfig, crdName string, all bool) error {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("config acquisition failed: %w", err)
	}

	crdClient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("API client generation failed: %w", err)
	}

	crdList, err := crdClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("unable to fetch CRDs from cluster: %w", err)
	}

	// Filter to a specific CRD only when --crd is set and --all is not
	if crdName != "" && !all {
		filtered := crdList.Items[:0]
		for _, crd := range crdList.Items {
			if crd.Name == crdName || strings.EqualFold(crd.Spec.Names.Kind, crdName) {
				filtered = append(filtered, crd)
			}
		}
		crdList.Items = filtered
	}

	for _, crd := range crdList.Items {
		if len(crd.Spec.Versions) == 0 {
			continue
		}

		targetVersion := crd.Spec.Versions[0]
		if targetVersion.Schema == nil || targetVersion.Schema.OpenAPIV3Schema == nil {
			continue
		}

		// Build GROUP/VERSION/ subdirectory under the output root
		schemaDir := filepath.Join(outputDir, crd.Spec.Group, targetVersion.Name)
		if err := os.MkdirAll(schemaDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", schemaDir, err)
		}

		jsonSchemaStruct := map[string]interface{}{
			"$schema": "http://json-schema.org",
			"title":   crd.Spec.Names.Kind,
			"type":    "object",
			"properties": map[string]interface{}{
				"apiVersion": map[string]string{"type": "string"},
				"kind":       map[string]string{"type": "string"},
				"metadata":   map[string]string{"type": "object"},
				"spec":       targetVersion.Schema.OpenAPIV3Schema.Properties["spec"],
				"status":     targetVersion.Schema.OpenAPIV3Schema.Properties["status"],
			},
			"required": []string{"apiVersion", "kind", "metadata", "spec"},
		}

		fileName := crd.Spec.Names.Kind + ".json"
		fileData, _ := json.MarshalIndent(jsonSchemaStruct, "", "  ")

		filePath := filepath.Join(schemaDir, fileName)
		if err = os.WriteFile(filePath, fileData, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filePath, err)
		}
		fmt.Printf("Extracted: %s\n", filePath)
	}

	return nil
}
