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

// Package registry contains the manually vendored type definitions from
// the operator-framework/operator-registry repository.
package registry

// AnnotationsFile holds annotation information about a bundle.
type AnnotationsFile struct {
	// annotations is a list of annotations for a given bundle
	Annotations Annotations `json:"annotations" yaml:"annotations"`
}

// Annotations is a list of annotations for a given bundle.
type Annotations struct {
	// PackageName is the name of the overall package, ala `etcd`.
	PackageName string `json:"operators.operatorframework.io.bundle.package.v1" yaml:"operators.operatorframework.io.bundle.package.v1"`

	// Channels are a comma separated list of the declared channels for the bundle, ala `stable` or `alpha`.
	Channels string `json:"operators.operatorframework.io.bundle.channels.v1" yaml:"operators.operatorframework.io.bundle.channels.v1"`

	// DefaultChannelName is, if specified, the name of the default channel for the package. The
	// default channel will be installed if no other channel is explicitly given. If the package
	// has a single channel, then that channel is implicitly the default.
	DefaultChannelName string `json:"operators.operatorframework.io.bundle.channel.default.v1" yaml:"operators.operatorframework.io.bundle.channel.default.v1"`
}
