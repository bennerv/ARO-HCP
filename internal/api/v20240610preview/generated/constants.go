//go:build go1.18
// +build go1.18

// Code generated by Microsoft (R) AutoRest Code Generator (autorest: 3.10.4, generator: @autorest/go@4.0.0-preview.63)
// Changes may cause incorrect behavior and will be lost if the code is regenerated.
// Code generated by @autorest/go. DO NOT EDIT.

package generated

const (
	moduleName    = "undefined"
	moduleVersion = "v0.0.1"
)

// ActionType - Enum. Indicates the action type. "Internal" refers to actions that are for internal only APIs.
type ActionType string

const (
	ActionTypeInternal ActionType = "Internal"
)

// PossibleActionTypeValues returns the possible values for the ActionType const type.
func PossibleActionTypeValues() []ActionType {
	return []ActionType{
		ActionTypeInternal,
	}
}

// CreatedByType - The type of identity that created the resource.
type CreatedByType string

const (
	CreatedByTypeApplication     CreatedByType = "Application"
	CreatedByTypeKey             CreatedByType = "Key"
	CreatedByTypeManagedIdentity CreatedByType = "ManagedIdentity"
	CreatedByTypeUser            CreatedByType = "User"
)

// PossibleCreatedByTypeValues returns the possible values for the CreatedByType const type.
func PossibleCreatedByTypeValues() []CreatedByType {
	return []CreatedByType{
		CreatedByTypeApplication,
		CreatedByTypeKey,
		CreatedByTypeManagedIdentity,
		CreatedByTypeUser,
	}
}

// Effect - The taint effect the same as in K8s
type Effect string

const (
	// EffectNoExecute - NoExecute taint effect
	EffectNoExecute Effect = "NoExecute"
	// EffectNoSchedule - NoSchedule taint effect
	EffectNoSchedule Effect = "NoSchedule"
	// EffectPreferNoSchedule - PreferNoSchedule taint effect
	EffectPreferNoSchedule Effect = "PreferNoSchedule"
)

// PossibleEffectValues returns the possible values for the Effect const type.
func PossibleEffectValues() []Effect {
	return []Effect{
		EffectNoExecute,
		EffectNoSchedule,
		EffectPreferNoSchedule,
	}
}

// ManagedServiceIdentityType - Type of managed service identity (where both SystemAssigned and UserAssigned types are allowed).
type ManagedServiceIdentityType string

const (
	ManagedServiceIdentityTypeNone                       ManagedServiceIdentityType = "None"
	ManagedServiceIdentityTypeSystemAssigned             ManagedServiceIdentityType = "SystemAssigned"
	ManagedServiceIdentityTypeSystemAssignedUserAssigned ManagedServiceIdentityType = "SystemAssigned,UserAssigned"
	ManagedServiceIdentityTypeUserAssigned               ManagedServiceIdentityType = "UserAssigned"
)

// PossibleManagedServiceIdentityTypeValues returns the possible values for the ManagedServiceIdentityType const type.
func PossibleManagedServiceIdentityTypeValues() []ManagedServiceIdentityType {
	return []ManagedServiceIdentityType{
		ManagedServiceIdentityTypeNone,
		ManagedServiceIdentityTypeSystemAssigned,
		ManagedServiceIdentityTypeSystemAssignedUserAssigned,
		ManagedServiceIdentityTypeUserAssigned,
	}
}

// NetworkType - The main controller responsible for rendering the core networking components
type NetworkType string

const (
	// NetworkTypeOVNKubernetes - The OVN network plugin for the OpenShift cluster
	NetworkTypeOVNKubernetes NetworkType = "OVNKubernetes"
	// NetworkTypeOther - Other network plugins
	NetworkTypeOther NetworkType = "Other"
)

// PossibleNetworkTypeValues returns the possible values for the NetworkType const type.
func PossibleNetworkTypeValues() []NetworkType {
	return []NetworkType{
		NetworkTypeOVNKubernetes,
		NetworkTypeOther,
	}
}

// Origin - The intended executor of the operation; as in Resource Based Access Control (RBAC) and audit logs UX. Default
// value is "user,system"
type Origin string

const (
	OriginSystem     Origin = "system"
	OriginUser       Origin = "user"
	OriginUserSystem Origin = "user,system"
)

// PossibleOriginValues returns the possible values for the Origin const type.
func PossibleOriginValues() []Origin {
	return []Origin{
		OriginSystem,
		OriginUser,
		OriginUserSystem,
	}
}

// OutboundType - The core outgoing configuration
type OutboundType string

const (
	// OutboundTypeLoadBalancer - The load balancer configuration
	OutboundTypeLoadBalancer OutboundType = "loadBalancer"
)

// PossibleOutboundTypeValues returns the possible values for the OutboundType const type.
func PossibleOutboundTypeValues() []OutboundType {
	return []OutboundType{
		OutboundTypeLoadBalancer,
	}
}

// ProvisioningState - The resource provisioning state.
type ProvisioningState string

const (
	// ProvisioningStateAccepted - Non-terminal state indicating the resource has been accepted
	ProvisioningStateAccepted ProvisioningState = "Accepted"
	// ProvisioningStateCanceled - Resource creation was canceled.
	ProvisioningStateCanceled ProvisioningState = "Canceled"
	// ProvisioningStateDeleting - Non-terminal state indicating the resource is deleting
	ProvisioningStateDeleting ProvisioningState = "Deleting"
	// ProvisioningStateFailed - Resource creation failed.
	ProvisioningStateFailed ProvisioningState = "Failed"
	// ProvisioningStateProvisioning - Non-terminal state indicating the resource is provisioning
	ProvisioningStateProvisioning ProvisioningState = "Provisioning"
	// ProvisioningStateSucceeded - Resource has been created.
	ProvisioningStateSucceeded ProvisioningState = "Succeeded"
	// ProvisioningStateUpdating - Non-terminal state indicating the resource is updating
	ProvisioningStateUpdating ProvisioningState = "Updating"
)

// PossibleProvisioningStateValues returns the possible values for the ProvisioningState const type.
func PossibleProvisioningStateValues() []ProvisioningState {
	return []ProvisioningState{
		ProvisioningStateAccepted,
		ProvisioningStateCanceled,
		ProvisioningStateDeleting,
		ProvisioningStateFailed,
		ProvisioningStateProvisioning,
		ProvisioningStateSucceeded,
		ProvisioningStateUpdating,
	}
}

// ResourceProvisioningState - The provisioning state of a resource type.
type ResourceProvisioningState string

const (
	// ResourceProvisioningStateCanceled - Resource creation was canceled.
	ResourceProvisioningStateCanceled ResourceProvisioningState = "Canceled"
	// ResourceProvisioningStateFailed - Resource creation failed.
	ResourceProvisioningStateFailed ResourceProvisioningState = "Failed"
	// ResourceProvisioningStateSucceeded - Resource has been created.
	ResourceProvisioningStateSucceeded ResourceProvisioningState = "Succeeded"
)

// PossibleResourceProvisioningStateValues returns the possible values for the ResourceProvisioningState const type.
func PossibleResourceProvisioningStateValues() []ResourceProvisioningState {
	return []ResourceProvisioningState{
		ResourceProvisioningStateCanceled,
		ResourceProvisioningStateFailed,
		ResourceProvisioningStateSucceeded,
	}
}

// Visibility - The visibility of the API server
type Visibility string

const (
	// VisibilityPrivate - The API server is not visible from the internet.
	VisibilityPrivate Visibility = "private"
	// VisibilityPublic - The API server is visible from the internet.
	VisibilityPublic Visibility = "public"
)

// PossibleVisibilityValues returns the possible values for the Visibility const type.
func PossibleVisibilityValues() []Visibility {
	return []Visibility{
		VisibilityPrivate,
		VisibilityPublic,
	}
}
