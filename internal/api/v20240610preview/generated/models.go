//go:build go1.18
// +build go1.18

// Code generated by Microsoft (R) AutoRest Code Generator (autorest: 3.10.4, generator: @autorest/go@4.0.0-preview.63)
// Changes may cause incorrect behavior and will be lost if the code is regenerated.
// Code generated by @autorest/go. DO NOT EDIT.

package generated

import "time"

// APIProfile - Information about the API of a cluster.
type APIProfile struct {
	// The internet visibility of the OpenShift API server
	Visibility *Visibility

	// READ-ONLY; URL endpoint for the API server
	URL *string
}

// AzureResourceManagerCommonTypesManagedServiceIdentityUpdate - Managed service identity (system assigned and/or user assigned
// identities)
type AzureResourceManagerCommonTypesManagedServiceIdentityUpdate struct {
	// The type of managed identity assigned to this resource.
	Type *ManagedServiceIdentityType

	// The identities assigned to this resource by the user.
	UserAssignedIdentities map[string]*Components19Kgb1NSchemasAzureResourcemanagerCommontypesManagedserviceidentityupdatePropertiesUserassignedidentitiesAdditionalproperties
}

// AzureResourceManagerCommonTypesTrackedResourceUpdate - The resource model definition for an Azure Resource Manager tracked
// top level resource which has 'tags' and a 'location'
type AzureResourceManagerCommonTypesTrackedResourceUpdate struct {
	// Resource tags.
	Tags map[string]*string

	// READ-ONLY; Fully qualified resource ID for the resource. E.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// ClusterCapabilitiesProfile - Cluster capabilities configuration.
type ClusterCapabilitiesProfile struct {
	// Immutable list of disabled capabilities. May only contain "ImageRegistry" at this time. Additional capabilities may be
	// available in the future. Clients should expect to handle additional values.
	Disabled []*OptionalClusterCapability
}

type Components19Kgb1NSchemasAzureResourcemanagerCommontypesManagedserviceidentityupdatePropertiesUserassignedidentitiesAdditionalproperties struct {
	// READ-ONLY; The client ID of the assigned identity.
	ClientID *string

	// READ-ONLY; The principal ID of the assigned identity.
	PrincipalID *string
}

// ConsoleProfile - Configuration of the cluster web console
type ConsoleProfile struct {
	// READ-ONLY; The cluster web console URL endpoint
	URL *string
}

// DNSProfile - DNS contains the DNS settings of the cluster
type DNSProfile struct {
	// BaseDomainPrefix is the unique name of the cluster representing the OpenShift's cluster name. BaseDomainPrefix is the name
	// that will appear in the cluster's DNS, provisioned cloud providers resources
	BaseDomainPrefix *string

	// READ-ONLY; BaseDomain is the base DNS domain of the cluster.
	BaseDomain *string
}

// ErrorAdditionalInfo - The resource management error additional info.
type ErrorAdditionalInfo struct {
	// READ-ONLY; The additional info.
	Info any

	// READ-ONLY; The additional info type.
	Type *string
}

// ErrorDetail - The error detail.
type ErrorDetail struct {
	// READ-ONLY; The error additional info.
	AdditionalInfo []*ErrorAdditionalInfo

	// READ-ONLY; The error code.
	Code *string

	// READ-ONLY; The error details.
	Details []*ErrorDetail

	// READ-ONLY; The error message.
	Message *string

	// READ-ONLY; The error target.
	Target *string
}

// ErrorResponse - Common error response for all Azure Resource Manager APIs to return error details for failed operations.
// (This also follows the OData error response format.).
type ErrorResponse struct {
	// The error object.
	Error *ErrorDetail
}

// HcpOpenShiftCluster - HCP cluster resource
type HcpOpenShiftCluster struct {
	// REQUIRED; The geo-location where the resource lives
	Location *string

	// The managed service identities assigned to this resource.
	Identity *ManagedServiceIdentity

	// The resource-specific properties for this resource.
	Properties *HcpOpenShiftClusterProperties

	// Resource tags.
	Tags map[string]*string

	// READ-ONLY; Fully qualified resource ID for the resource. E.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// HcpOpenShiftClusterAdminCredential - HCP cluster admin credential
type HcpOpenShiftClusterAdminCredential struct {
	// READ-ONLY; Expiration timestamp for the kubeconfig's client certificate
	ExpirationTimestamp *time.Time

	// READ-ONLY; Admin kubeconfig with a temporary client certificate
	Kubeconfig *string
}

// HcpOpenShiftClusterListResult - The response of a HcpOpenShiftCluster list operation.
type HcpOpenShiftClusterListResult struct {
	// REQUIRED; The HcpOpenShiftCluster items on this page
	Value []*HcpOpenShiftCluster

	// The link to the next page of items
	NextLink *string
}

// HcpOpenShiftClusterProperties - HCP cluster properties
type HcpOpenShiftClusterProperties struct {
	// REQUIRED; Version of the control plane components
	Version *VersionProfile

	// Cluster DNS configuration
	DNS *DNSProfile

	// READ-ONLY; Shows the cluster web console information
	Console *ConsoleProfile

	// Configure cluter capabilities.
	Capabilities *ClusterCapabilitiesProfile

	// Disable user workload monitoring
	DisableUserWorkloadMonitoring *bool

	// Cluster network configuration
	Network *NetworkProfile

	// Azure platform configuration
	Platform *PlatformProfile

	// READ-ONLY; Shows the cluster API server profile
	API *APIProfile

	// READ-ONLY; The status of the last operation.
	ProvisioningState *ProvisioningState
}

// HcpOpenShiftClusterPropertiesUpdate - HCP cluster properties
type HcpOpenShiftClusterPropertiesUpdate struct {
	// Cluster DNS configuration
	DNS *DNSProfile

	// Disable user workload monitoring
	DisableUserWorkloadMonitoring *bool
}

// HcpOpenShiftClusterUpdate - HCP cluster resource
type HcpOpenShiftClusterUpdate struct {
	// The managed service identities assigned to this resource.
	Identity *AzureResourceManagerCommonTypesManagedServiceIdentityUpdate

	// The resource-specific properties for this resource.
	Properties *HcpOpenShiftClusterPropertiesUpdate

	// Resource tags.
	Tags map[string]*string

	// READ-ONLY; Fully qualified resource ID for the resource. E.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// Label represents the k8s label
type Label struct {
	// The key of the label
	Key *string

	// The value of the label
	Value *string
}

// ManagedServiceIdentity - Managed service identity (system assigned and/or user assigned identities)
type ManagedServiceIdentity struct {
	// REQUIRED; Type of managed service identity (where both SystemAssigned and UserAssigned types are allowed).
	Type *ManagedServiceIdentityType

	// The set of user assigned identities associated with the resource. The userAssignedIdentities dictionary keys will be ARM
	// resource ids in the form:
	// '/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ManagedIdentity/userAssignedIdentities/{identityName}.
	// The dictionary values can be empty objects ({}) in
	// requests.
	UserAssignedIdentities map[string]*UserAssignedIdentity

	// READ-ONLY; The service principal ID of the system assigned identity. This property will only be provided for a system assigned
	// identity.
	PrincipalID *string

	// READ-ONLY; The tenant ID of the system assigned identity. This property will only be provided for a system assigned identity.
	TenantID *string
}

// NetworkProfile - OpenShift networking configuration
type NetworkProfile struct {
	// Network host prefix
	HostPrefix *int32

	// The CIDR block from which to assign machine IP addresses
	MachineCidr *string

	// The main controller responsible for rendering the core networking components
	NetworkType *NetworkType

	// The CIDR of the pod IP addresses
	PodCidr *string

	// The CIDR block for assigned service IPs
	ServiceCidr *string
}

// NodePool - Concrete tracked resource types can be created by aliasing this type using a specific property type.
type NodePool struct {
	// REQUIRED; The geo-location where the resource lives
	Location *string

	// The managed service identities assigned to this resource.
	Identity *ManagedServiceIdentity

	// The resource-specific properties for this resource.
	Properties *NodePoolProperties

	// Resource tags.
	Tags map[string]*string

	// READ-ONLY; Fully qualified resource ID for the resource. E.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// NodePoolAutoScaling - Node pool autoscaling
type NodePoolAutoScaling struct {
	// The maximum number of nodes in the node pool
	Max *int32

	// The minimum number of nodes in the node pool
	Min *int32
}

// NodePoolListResult - The response of a NodePool list operation.
type NodePoolListResult struct {
	// REQUIRED; The NodePool items on this page
	Value []*NodePool

	// The link to the next page of items
	NextLink *string
}

// NodePoolPlatformProfile - Azure node pool platform configuration
type NodePoolPlatformProfile struct {
	// REQUIRED; The VM size according to the documentation:
	// * https://learn.microsoft.com/en-us/azure/virtual-machines/sizes
	VMSize *string

	// The availability zone for the node pool. Please read the documentation to see which regions support availability zones
	// * https://learn.microsoft.com/en-us/azure/availability-zones/az-overview
	AvailabilityZone *string

	// The OS disk size in GiB
	DiskSizeGiB *int32

	// The type of the disk storage account
	// * https://learn.microsoft.com/en-us/azure/virtual-machines/disks-types
	DiskStorageAccountType *DiskStorageAccountType

	// The Azure resource ID of the worker subnet
	SubnetID *string
}

// NodePoolProperties - Represents the node pool properties
type NodePoolProperties struct {
	// REQUIRED; Azure node pool platform configuration
	Platform *NodePoolPlatformProfile

	// REQUIRED; OpenShift version for the nodepool
	Version *NodePoolVersionProfile

	// Auto-repair
	AutoRepair *bool

	// Representation of a autoscaling in a node pool.
	AutoScaling *NodePoolAutoScaling

	// K8s labels to propagate to the NodePool Nodes The good example of the label is node-role.kubernetes.io/master: ""
	Labels []*Label

	// The number of worker nodes, it cannot be used together with autoscaling
	Replicas *int32

	// Taints for the nodes
	Taints []*Taint

	// READ-ONLY; Provisioning state
	ProvisioningState *ProvisioningState
}

// NodePoolPropertiesUpdate - Represents the node pool properties
type NodePoolPropertiesUpdate struct {
	// Representation of a autoscaling in a node pool.
	AutoScaling *NodePoolAutoScaling

	// K8s labels to propagate to the NodePool Nodes The good example of the label is node-role.kubernetes.io/master: ""
	Labels []*Label

	// The number of worker nodes, it cannot be used together with autoscaling
	Replicas *int32

	// Taints for the nodes
	Taints []*Taint
}

// NodePoolUpdate - Concrete tracked resource types can be created by aliasing this type using a specific property type.
type NodePoolUpdate struct {
	// The managed service identities assigned to this resource.
	Identity *AzureResourceManagerCommonTypesManagedServiceIdentityUpdate

	// The resource-specific properties for this resource.
	Properties *NodePoolPropertiesUpdate

	// Resource tags.
	Tags map[string]*string

	// READ-ONLY; Fully qualified resource ID for the resource. E.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// NodePoolVersionProfile - Versions represents an OpenShift version.
type NodePoolVersionProfile struct {
	// ChannelGroup is the name of the set to which this version belongs. Each version belongs to only a single set.
	ChannelGroup *string

	// ID is the unique identifier of the version.
	ID *string

	// READ-ONLY; AvailableUpgrades is a list of version names the current version can be upgraded to.
	AvailableUpgrades []*string
}

// Operation - Details of a REST API operation, returned from the Resource Provider Operations API
type Operation struct {
	// Localized display information for this particular operation.
	Display *OperationDisplay

	// READ-ONLY; Enum. Indicates the action type. "Internal" refers to actions that are for internal only APIs.
	ActionType *ActionType

	// READ-ONLY; Whether the operation applies to data-plane. This is "true" for data-plane operations and "false" for ARM/control-plane
	// operations.
	IsDataAction *bool

	// READ-ONLY; The name of the operation, as per Resource-Based Access Control (RBAC). Examples: "Microsoft.Compute/virtualMachines/write",
	// "Microsoft.Compute/virtualMachines/capture/action"
	Name *string

	// READ-ONLY; The intended executor of the operation; as in Resource Based Access Control (RBAC) and audit logs UX. Default
	// value is "user,system"
	Origin *Origin
}

// OperationDisplay - Localized display information for this particular operation.
type OperationDisplay struct {
	// READ-ONLY; The short, localized friendly description of the operation; suitable for tool tips and detailed views.
	Description *string

	// READ-ONLY; The concise, localized friendly name for the operation; suitable for dropdowns. E.g. "Create or Update Virtual
	// Machine", "Restart Virtual Machine".
	Operation *string

	// READ-ONLY; The localized friendly form of the resource provider name, e.g. "Microsoft Monitoring Insights" or "Microsoft
	// Compute".
	Provider *string

	// READ-ONLY; The localized friendly name of the resource type related to this operation. E.g. "Virtual Machines" or "Job
	// Schedule Collections".
	Resource *string
}

// OperationListResult - A list of REST API operations supported by an Azure Resource Provider. It contains an URL link to
// get the next set of results.
type OperationListResult struct {
	// READ-ONLY; URL to get the next set of operation list results (if there are any).
	NextLink *string

	// READ-ONLY; List of operations supported by the resource provider
	Value []*Operation
}

// OperatorsAuthenticationProfile - The configuration that the operators of the cluster have to authenticate to Azure.
type OperatorsAuthenticationProfile struct {
	// REQUIRED; Represents the information related to Azure User-Assigned managed identities needed to perform Operators authentication
	// based on Azure User-Assigned Managed Identities
	UserAssignedIdentities *UserAssignedIdentitiesProfile
}

// PlatformProfile - Azure specific configuration
type PlatformProfile struct {
	// REQUIRED; ResourceId for the network security group attached to the cluster subnet
	NetworkSecurityGroupID *string

	// REQUIRED; The configuration that the operators of the cluster have to authenticate to Azure
	OperatorsAuthentication *OperatorsAuthenticationProfile

	// REQUIRED; The Azure resource ID of the worker subnet
	SubnetID *string

	// Resource group to put cluster resources
	ManagedResourceGroup *string

	// The core outgoing configuration
	OutboundType *OutboundType

	// READ-ONLY; URL for the OIDC provider to be used for authentication to authenticate against user Azure cloud account
	IssuerURL *string
}

// Resource - Common fields that are returned in the response for all Azure Resource Manager resources
type Resource struct {
	// READ-ONLY; Fully qualified resource ID for the resource. E.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// SystemData - Metadata pertaining to creation and last modification of the resource.
type SystemData struct {
	// The timestamp of resource creation (UTC).
	CreatedAt *time.Time

	// The identity that created the resource.
	CreatedBy *string

	// The type of identity that created the resource.
	CreatedByType *CreatedByType

	// The timestamp of resource last modification (UTC)
	LastModifiedAt *time.Time

	// The identity that last modified the resource.
	LastModifiedBy *string

	// The type of identity that last modified the resource.
	LastModifiedByType *CreatedByType
}

// Taint is controlling the node taint and its effects
type Taint struct {
	// The effect of the taint
	Effect *Effect

	// The key of the taint
	Key *string

	// The value of the taint
	Value *string
}

// TrackedResource - The resource model definition for an Azure Resource Manager tracked top level resource which has 'tags'
// and a 'location'
type TrackedResource struct {
	// REQUIRED; The geo-location where the resource lives
	Location *string

	// Resource tags.
	Tags map[string]*string

	// READ-ONLY; Fully qualified resource ID for the resource. E.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// UserAssignedIdentitiesProfile - Represents the information related to Azure User-Assigned managed identities needed to
// perform Operators authentication based on Azure User-Assigned Managed Identities
type UserAssignedIdentitiesProfile struct {
	// REQUIRED; The set of Azure User-Assigned Managed Identities leveraged for the Control Plane operators of the cluster. The
	// set of required managed identities is dependent on the Cluster's OpenShift version.
	ControlPlaneOperators map[string]*string

	// REQUIRED; The set of Azure User-Assigned Managed Identities leveraged for the Data Plane operators of the cluster. The
	// set of required managed identities is dependent on the Cluster's OpenShift version.
	DataPlaneOperators map[string]*string

	// REQUIRED; Represents the information associated to an Azure User-Assigned Managed Identity whose purpose is to perform
	// service level actions.
	ServiceManagedIdentity *string
}

// UserAssignedIdentity - User assigned identity properties
type UserAssignedIdentity struct {
	// READ-ONLY; The client ID of the assigned identity.
	ClientID *string

	// READ-ONLY; The principal ID of the assigned identity.
	PrincipalID *string
}

// VersionProfile - Versions represents an OpenShift version.
type VersionProfile struct {
	// ChannelGroup is the name of the set to which this version belongs. Each version belongs to only a single set.
	ChannelGroup *string

	// ID is the unique identifier of the version.
	ID *string

	// READ-ONLY; AvailableUpgrades is a list of version names the current version can be upgraded to.
	AvailableUpgrades []*string
}
