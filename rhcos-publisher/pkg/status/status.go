// Copyright 2026 Microsoft Corporation
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

// Package status reads and updates RHCOSRelease resources. The controller
// only ever writes the status subresource; the spec is owned by the Helm
// chart. A dynamic client is used so no generated clientset is needed for
// this single small resource.
package status

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/api/v1alpha1"
)

// CRName is the RHCOSRelease resource name of a branch: dots are not allowed
// in a DNS-1123 label position we use, so "release-4.22" becomes
// "release-4-22". The Helm chart applies the same transformation.
func CRName(branch string) string {
	return strings.ReplaceAll(branch, ".", "-")
}

// Client reads and updates RHCOSRelease resources in one namespace.
type Client struct {
	dynamicClient dynamic.Interface
	namespace     string
}

// NewClient builds a status client bound to the given namespace.
func NewClient(dynamicClient dynamic.Interface, namespace string) *Client {
	return &Client{dynamicClient: dynamicClient, namespace: namespace}
}

// Get fetches the RHCOSRelease of a branch.
func (c *Client) Get(ctx context.Context, branch string) (*v1alpha1.RHCOSRelease, error) {
	obj, err := c.dynamicClient.Resource(v1alpha1.RHCOSReleaseGVR).Namespace(c.namespace).Get(ctx, CRName(branch), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get RHCOSRelease %s: %w", CRName(branch), err)
	}
	release := &v1alpha1.RHCOSRelease{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, release); err != nil {
		return nil, fmt.Errorf("failed to convert RHCOSRelease %s: %w", CRName(branch), err)
	}
	return release, nil
}

// UpdateStatus applies mutate to the branch's RHCOSRelease status and writes
// it back through the status subresource, retrying on conflicts.
func (c *Client) UpdateStatus(ctx context.Context, branch string, mutate func(*v1alpha1.RHCOSReleaseStatus)) error {
	resourceClient := c.dynamicClient.Resource(v1alpha1.RHCOSReleaseGVR).Namespace(c.namespace)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		obj, err := resourceClient.Get(ctx, CRName(branch), metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get RHCOSRelease %s: %w", CRName(branch), err)
		}
		release := &v1alpha1.RHCOSRelease{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, release); err != nil {
			return fmt.Errorf("failed to convert RHCOSRelease %s: %w", CRName(branch), err)
		}

		mutate(&release.Status)

		updated, err := runtime.DefaultUnstructuredConverter.ToUnstructured(release)
		if err != nil {
			return fmt.Errorf("failed to convert RHCOSRelease %s: %w", CRName(branch), err)
		}
		_, err = resourceClient.UpdateStatus(ctx, &unstructured.Unstructured{Object: updated}, metav1.UpdateOptions{})
		return err
	})
}

// UpdateArchStatus applies mutate to one architecture's status entry,
// creating it if absent.
func (c *Client) UpdateArchStatus(ctx context.Context, branch, arch string, mutate func(*v1alpha1.RHCOSReleaseArchStatus)) error {
	return c.UpdateStatus(ctx, branch, func(releaseStatus *v1alpha1.RHCOSReleaseStatus) {
		if releaseStatus.Architectures == nil {
			releaseStatus.Architectures = map[string]v1alpha1.RHCOSReleaseArchStatus{}
		}
		archStatus := releaseStatus.Architectures[arch]
		mutate(&archStatus)
		releaseStatus.Architectures[arch] = archStatus
	})
}
