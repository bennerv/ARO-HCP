// Copyright 2025 Microsoft Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/yaml"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/Azure/ARO-HCP/test/util/framework"
	"github.com/Azure/ARO-HCP/test/util/labels"
)

var (
	acmPolicyGVR = schema.GroupVersionResource{
		Group:    "policy.open-cluster-management.io",
		Version:  "v1",
		Resource: "policies",
	}
	acmPlacementBindingGVR = schema.GroupVersionResource{
		Group:    "policy.open-cluster-management.io",
		Version:  "v1",
		Resource: "placementbindings",
	}
)

var _ = Describe("Ingress Policy Validation", func() {
	It("should dump ingress and ACM policy state for policy removal validation",
		labels.RequireNothing,
		labels.Medium,
		labels.Positive,
		labels.DevelopmentOnly,
		labels.AroRpApiCompatible,
		func(ctx context.Context) {
			const (
				resourceGroupName        = "policy-removal-test"
				managedResourceGroupName = "policy-removal-test-managed"
				customerClusterName      = "pol-rem-cl"
			)

			tc := framework.NewTestContext()

			hcpClient := tc.Get20240610ClientFactoryOrDie(ctx).NewHcpOpenShiftClustersClient()

			By("checking if cluster already exists")
			_, err := framework.GetHCPCluster20240610(ctx, hcpClient, resourceGroupName, customerClusterName)
			clusterExists := err == nil

			if clusterExists {
				GinkgoLogr.Info("cluster already exists, skipping creation", "cluster", customerClusterName, "resourceGroup", resourceGroupName)
			} else {
				GinkgoLogr.Info("cluster does not exist, creating resources", "cluster", customerClusterName, "resourceGroup", resourceGroupName)

				if tc.UsePooledIdentities() {
					err := tc.AssignIdentityContainers(ctx, 1, framework.IdentityContainerAssignmentRetryInterval)
					Expect(err).NotTo(HaveOccurred(), "failed to assign pooled identity containers")
				}

				By("creating resource group (no auto-cleanup)")
				rgClient := tc.GetARMResourcesClientFactoryOrDie(ctx).NewResourceGroupsClient()
				_, err = framework.CreateResourceGroup(ctx, rgClient, resourceGroupName, tc.Location(), framework.StandardResourceGroupExpiration, 20*time.Minute)
				Expect(err).NotTo(HaveOccurred(), "failed to create resource group %q", resourceGroupName)

				By("creating cluster parameters")
				clusterParams := framework.NewDefaultClusterParams20240610()
				clusterParams.ClusterName = customerClusterName
				clusterParams.ManagedResourceGroupName = managedResourceGroupName

				By("creating customer resources")
				resourceGroup := &armresources.ResourceGroup{
					Name:     to.Ptr(resourceGroupName),
					Location: to.Ptr(tc.Location()),
				}
				clusterParams, err = tc.CreateClusterCustomerResources20240610(ctx,
					resourceGroup,
					clusterParams,
					map[string]any{},
					TestArtifactsFS,
					framework.RBACScopeResourceGroup,
				)
				Expect(err).NotTo(HaveOccurred(), "failed to create customer resources")

				By("creating HCP cluster")
				err = tc.CreateHCPClusterFromParam20240610(
					ctx,
					GinkgoLogr,
					resourceGroupName,
					clusterParams,
					framework.ClusterCreationTimeout,
				)
				Expect(err).NotTo(HaveOccurred(), "failed to create HCP cluster %q", customerClusterName)
			}

			By("getting admin REST config for hosted cluster (registers for oc adm inspect)")
			_, err = tc.GetAdminRESTConfigForHCPCluster20240610(
				ctx,
				hcpClient,
				resourceGroupName,
				customerClusterName,
				framework.GetAdminRESTConfigTimeout,
			)
			Expect(err).NotTo(HaveOccurred(), "failed to get admin REST config for cluster %q", customerClusterName)

			By("dumping management cluster ACM policy state")
			dumpManagementClusterState(ctx, tc.LogDirPath)
		},
	)
})

func dumpManagementClusterState(ctx context.Context, artifactDir string) {
	mgmtKubeconfig := os.Getenv("MGMT_KUBECONFIG")
	Expect(mgmtKubeconfig).NotTo(BeEmpty(), "MGMT_KUBECONFIG env var must be set to the management cluster kubeconfig path")
	GinkgoLogr.Info("using management cluster kubeconfig", "path", mgmtKubeconfig)

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: mgmtKubeconfig}
	mgmtRESTConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	Expect(err).NotTo(HaveOccurred(), "failed to load management cluster kubeconfig from %q", mgmtKubeconfig)

	dynamicClient, err := dynamic.NewForConfig(mgmtRESTConfig)
	Expect(err).NotTo(HaveOccurred(), "failed to create dynamic client for management cluster")

	timestamp := time.Now().UTC().Format("20060102-150405")
	acmDir := filepath.Join(artifactDir, fmt.Sprintf("acm-policy-state-%s", timestamp))
	Expect(os.MkdirAll(acmDir, 0755)).To(Succeed(), "failed to create artifact directory %q", acmDir)
	GinkgoLogr.Info("writing ACM policy artifacts", "dir", acmDir)

	By("listing all ACM Policies across all namespaces")
	policyList, err := dynamicClient.Resource(acmPolicyGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		GinkgoLogr.Info("failed to list ACM Policies (CRD may not exist)", "error", err.Error())
	} else if len(policyList.Items) == 0 {
		GinkgoLogr.Info("no ACM Policies found across any namespace")
	} else {
		for i := range policyList.Items {
			p := &policyList.Items[i]
			pYAML := mustMarshalYAML(p.Object)
			writeArtifact(acmDir, fmt.Sprintf("policy-%s-%s.yaml", p.GetNamespace(), p.GetName()), pYAML)
		}
	}

	By("listing all ACM PlacementBindings across all namespaces")
	pbList, err := dynamicClient.Resource(acmPlacementBindingGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		GinkgoLogr.Info("failed to list ACM PlacementBindings (CRD may not exist)", "error", err.Error())
	} else if len(pbList.Items) == 0 {
		GinkgoLogr.Info("no ACM PlacementBindings found across any namespace")
	} else {
		for i := range pbList.Items {
			pb := &pbList.Items[i]
			pbYAML := mustMarshalYAML(pb.Object)
			writeArtifact(acmDir, fmt.Sprintf("placementbinding-%s-%s.yaml", pb.GetNamespace(), pb.GetName()), pbYAML)
		}
	}
}

func mustMarshalYAML(obj any) string {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("<marshal error: %v>", err)
	}
	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return fmt.Sprintf("<yaml conversion error: %v>", err)
	}
	return string(yamlBytes)
}

func writeArtifact(dir, filename, content string) {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		GinkgoLogr.Error(err, "failed to write artifact", "path", path)
	}
}
