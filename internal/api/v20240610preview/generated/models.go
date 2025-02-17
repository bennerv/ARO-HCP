//go:build go1.18
// +build go1.18

// Code generated by Microsoft (R) AutoRest Code Generator (autorest: 3.10.3, generator: @autorest/go@4.0.0-preview.63)
// Changes may cause incorrect behavior and will be lost if the code is regenerated.
// Code generated by @autorest/go. DO NOT EDIT.

package generated

import "time"

// APIProfile - Information about the API of a cluster.
type APIProfile struct {
	// REQUIRED; should the API server be accessible from the internet
	Visibility *Visibility

	// READ-ONLY; URL endpoint for the API server
	URL *string
}

// ClaimProfile - External auth claim profile
type ClaimProfile struct {
	// REQUIRED; Claim name of the external profile
	Claim *string

	// REQUIRED; Prefix for the claim external profile
	Prefix *string

	// REQUIRED; Prefix policy
	PrefixPolicy *string
}

type ComponentsQjfoe3SchemasManagedserviceidentityupdatePropertiesUserassignedidentitiesAdditionalproperties struct {
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

// ExternalAuthClaimProfile - External auth claim profile
type ExternalAuthClaimProfile struct {
	// REQUIRED; The claim mappings
	Mappings *TokenClaimMappingsProfile

	// REQUIRED; The claim validation rules
	ValidationRules []*TokenClaimValidationRuleProfile
}

// ExternalAuthClientComponentProfile - External auth component profile
type ExternalAuthClientComponentProfile struct {
	// REQUIRED; The namespace of the external auth client
	AuthClientNamespace *string

	// REQUIRED; The name of the external auth client
	Name *string
}

// ExternalAuthClientProfile - External auth client profile
type ExternalAuthClientProfile struct {
	// REQUIRED; External auth client component
	Component *ExternalAuthClientComponentProfile

	// REQUIRED; external auth client scopes
	ExtraScopes []*string

	// REQUIRED; external auth client id
	ID *string

	// REQUIRED; external auth client secret
	Secret *string
}

// ExternalAuthConfigProfile - External authentication configuration profile
type ExternalAuthConfigProfile struct {
	// READ-ONLY; This can only be set as a day-2 resource on a separate endpoint to provide a self-managed auth service
	ExternalAuths []*ExternalAuthProfile

	// This can be set during cluster creation only to ensure there is no openshift-oauth-apiserver in cluster
	Enabled *bool
}

// ExternalAuthProfile - External authentication profile
type ExternalAuthProfile struct {
	// REQUIRED; External auth claim
	Claim *ExternalAuthClaimProfile

	// REQUIRED; External auth clients
	Clients []*ExternalAuthClientProfile

	// REQUIRED; Token Issuer profile
	Issuer *TokenIssuerProfile
}

// HcpOpenShiftClusterCredentials - HCP cluster credentials
type HcpOpenShiftClusterCredentials struct {
	// READ-ONLY; kube admin password
	KubeadminPassword *string

	// READ-ONLY; kubeadmin user name
	KubeadminUsername *string
}

// HcpOpenShiftClusterKubeconfig - HCP cluster admin kubeconfig
type HcpOpenShiftClusterKubeconfig struct {
	// READ-ONLY; The kubeconfig file
	Kubeconfig *string
}

// HcpOpenShiftClusterNodePoolPatch - The template for adding optional properties.
type HcpOpenShiftClusterNodePoolPatch struct {
	// Managed Service Identity
	Identity *ManagedServiceIdentityUpdate

	// Represents the patchable node pool properties
	Properties *NodePoolPatchProperties

	// Resource tags.
	Tags map[string]*string
}

// HcpOpenShiftClusterNodePoolResource - Concrete tracked resource types can be created by aliasing this type using a specific
// property type.
type HcpOpenShiftClusterNodePoolResource struct {
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

// HcpOpenShiftClusterNodePoolResourceListResult - The response of a HcpOpenShiftClusterNodePoolResource list operation.
type HcpOpenShiftClusterNodePoolResourceListResult struct {
	// REQUIRED; The HcpOpenShiftClusterNodePoolResource items on this page
	Value []*HcpOpenShiftClusterNodePoolResource

	// The link to the next page of items
	NextLink *string
}

// HcpOpenShiftClusterPatch - The template for adding optional properties.
type HcpOpenShiftClusterPatch struct {
	// Managed service identity
	Identity *ManagedServiceIdentityUpdate

	// HCP patchable cluster properties
	Properties *HcpOpenShiftClusterPatchProperties

	// Resource tags.
	Tags map[string]*string
}

// HcpOpenShiftClusterPatchProperties - HCP patchable cluster properties
type HcpOpenShiftClusterPatchProperties struct {
	// Disable user workload monitoring
	DisableUserWorkloadMonitoring *bool

	// Openshift cluster proxy configuration
	Proxy *ProxyProfile

	// READ-ONLY; The status of the last operation.
	ProvisioningState *ProvisioningState
}

// HcpOpenShiftClusterProperties - HCP cluster properties
type HcpOpenShiftClusterProperties struct {
	// REQUIRED; Version of the control plane components
	Version *VersionProfile

	// READ-ONLY; URL for the OIDC provider to be used for authentication to authenticate against user Azure cloud account
	IssuerURL *string

	// Cluster DNS configuration
	DNS *DNSProfile

	// Disable user workload monitoring
	DisableUserWorkloadMonitoring *bool

	// Enables customer ETCD encryption, set during creation When set to true, platform.etcdEncryptionSetId must be set
	EtcdEncryption *bool

	// Configuration to override the openshift-oauth-apiserver inside cluster This changes user login into the cluster to external
// provider
	ExternalAuth *ExternalAuthConfigProfile

	// Enable FIPS mode for the cluster When set to true, etcdEncryption must be set to true
	Fips *bool

	// Cluster network configuration
	Network *NetworkProfile

	// Azure platform configuration
	Platform *PlatformProfile

	// Openshift cluster proxy configuration
	Proxy *ProxyProfile

	// READ-ONLY; Shows the cluster API server profile
	API *APIProfile

	// READ-ONLY; Shows the cluster web console information
	Console *ConsoleProfile

	// READ-ONLY; The status of the last operation.
	ProvisioningState *ProvisioningState
}

// HcpOpenShiftClusterResource - HCP cluster resource
type HcpOpenShiftClusterResource struct {
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

// HcpOpenShiftClusterResourceListResult - The response of a HcpOpenShiftClusterResource list operation.
type HcpOpenShiftClusterResourceListResult struct {
	// REQUIRED; The HcpOpenShiftClusterResource items on this page
	Value []*HcpOpenShiftClusterResource

	// The link to the next page of items
	NextLink *string
}

// HcpOpenShiftVersionResource - HcpOpenShiftVersions represents a location based available HCP cluster versions
type HcpOpenShiftVersionResource struct {
	// The resource-specific properties for this resource.
	Properties *HcpOpenShiftVersionsProperties

	// READ-ONLY; Fully qualified resource ID for the resource. E.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// HcpOpenShiftVersionResourceListResult - The response of a HcpOpenShiftVersionResource list operation.
type HcpOpenShiftVersionResourceListResult struct {
	// REQUIRED; The HcpOpenShiftVersionResource items on this page
	Value []*HcpOpenShiftVersionResource

	// The link to the next page of items
	NextLink *string
}

// HcpOpenShiftVersionsProperties is the installable cluster version
type HcpOpenShiftVersionsProperties struct {
	// READ-ONLY; The cluster version
	ClusterVersion *string

	// READ-ONLY; The provisioning state of the resource.
	ProvisioningState *ResourceProvisioningState
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

// ManagedServiceIdentityUpdate - The template for adding optional properties.
type ManagedServiceIdentityUpdate struct {
	// The type of managed identity assigned to this resource.
	Type *ManagedServiceIdentityType

	// The identities assigned to this resource by the user.
	UserAssignedIdentities map[string]*ComponentsQjfoe3SchemasManagedserviceidentityupdatePropertiesUserassignedidentitiesAdditionalproperties
}

// NetworkProfile - Network profile of the cluster
type NetworkProfile struct {
	// REQUIRED; from which to assign machine IP addresses, example: 10.0.0.0/16
	MachineCidr *string

	// REQUIRED; The CIDR of the pod IP addresses example: 10.128.0.0/14
	PodCidr *string

	// REQUIRED; The CIDR block for assigned service IPs, example: 172.30.0.0/16
	ServiceCidr *string

	// Network host prefix which is defaulted to 23 if not specified.
	HostPrefix *int32

	// The main controller responsible for rendering the core networking components
	NetworkType *NetworkType
}

// NodePoolAutoScaling - Node pool autoscaling
type NodePoolAutoScaling struct {
	// The maximum number of nodes in the node pool
	Max *int32

	// The minimum number of nodes in the node pool
	Min *int32
}

// NodePoolPatchProperties - Represents the patchable node pool properties
type NodePoolPatchProperties struct {
	// Representation of a autoscaling in a node pool.
	AutoScaling *NodePoolAutoScaling

	// K8s labels to propagate to the NodePool Nodes The good example of the label is node-role.kubernetes.io/master: ""
	Labels []*Label

	// The number of worker nodes, it cannot be used together with autoscaling
	Replicas *int32

	// Taints for the nodes
	Taints []*Taint

	// Tuning configs, TODO provide meaningful explanation TuningConfig is a list of references to ConfigMaps containing serialized
// Tuned resources to define the tuning configuration to be applied to nodes
// in the NodePool. Each ConfigMap must have a single key named "tuned" whose value is the JSON or YAML of a serialized Tuned
// or PerformanceProfile.
	TuningConfigs []*string

	// READ-ONLY; Provisioning state
	ProvisioningState *ResourceProvisioningState
}

// NodePoolPlatformProfile - Azure node pool platform configuration
type NodePoolPlatformProfile struct {
	// REQUIRED; The VM size according to the documentation:
// * https://learn.microsoft.com/en-us/azure/virtual-machines/sizes
	VMSize *string

	// The availability zone for the node pool. Please read the documentation to see which regions support availability zones
// * https://learn.microsoft.com/en-us/azure/availability-zones/az-overview
	AvailabilityZone *string

	// Disk Encryption Set ID that will be used for encryption the Nodes disks
// * https://learn.microsoft.com/en-us/azure/virtual-machines/disk-encryption-overview
// * https://learn.microsoft.com/en-us/azure/virtual-machines/disk-encryption
	DiskEncryptionSetID *string

	// The OS disk size in GiB
	DiskSizeGiB *int32

	// The type of the disk storage account
// * https://learn.microsoft.com/en-us/azure/virtual-machines/disks-types
	DiskStorageAccountType *string

	// Whether the worker machines should be encrypted at host
	EncryptionAtHost *bool

	// Is the disk ephemeral
	EphemeralOsDisk *bool

	// The resourceId for the subnet used by the workers
	SubnetID *string
}

// NodePoolProperties - Represents the node pool properties
type NodePoolProperties struct {
	// REQUIRED; Azure node pool platform configuration
	Platform *NodePoolPlatformProfile

	// REQUIRED; OpenShift version for the nodepool
	Version *VersionProfile

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

	// Tuning configs, TODO provide meaningful explanation TuningConfig is a list of references to ConfigMaps containing serialized
// Tuned resources to define the tuning configuration to be applied to nodes
// in the NodePool. Each ConfigMap must have a single key named "tuned" whose value is the JSON or YAML of a serialized Tuned
// or PerformanceProfile.
	TuningConfigs []*string

	// READ-ONLY; Provisioning state
	ProvisioningState *ProvisioningState
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
	// REQUIRED; The configuration that the operators of the cluster have to authenticate to Azure
	OperatorsAuthentication *OperatorsAuthenticationProfile

	// REQUIRED; ResourceId for the subnet used by the control plane
	SubnetID *string

	// The id of the disk encryption set to be used for etcd. Configure this when etcdEncryption is set to true Is used the
// https://learn.microsoft.com/en-us/azure/storage/common/customer-managed-keys-overview
	EtcdEncryptionSetID *string

	// Resource group to put cluster resources
	ManagedResourceGroup *string

	// ResourceId for the network security group attached to the cluster subnet
	NetworkSecurityGroupID *string

	// The core outgoing configuration
	OutboundType *OutboundType
}

// ProxyProfile - OpenShift cluster proxy configuration
type ProxyProfile struct {
	// http proxy config
	HTTPProxy *string

	// https proxy config
	HTTPSProxy *string

	// no proxy config
	NoProxy *string

	// The trusted CA for the proxy
	TrustedCa *string
}

// ProxyResource - The resource model definition for a Azure Resource Manager proxy resource. It will not have tags and a
// location
type ProxyResource struct {
	// READ-ONLY; Fully qualified resource ID for the resource. E.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
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
	// The effect of the taint The good example of the taint effect is NoSchedule
	Effect *Effect

	// The key of the taint The good example of the taint key is node-role.kubernetes.io/master
	Key *string

	// The value of the taint The good example of the taint value is NoSchedule
	Value *string
}

// TokenClaimMappingsProfile - External auth claim mappings profile
type TokenClaimMappingsProfile struct {
	// REQUIRED; The claim mappings groups
	Groups *ClaimProfile

	// REQUIRED; The claim mappings username
	Username *ClaimProfile
}

// TokenClaimValidationRuleProfile - External auth claim validation rule
type TokenClaimValidationRuleProfile struct {
	// REQUIRED; Claim name for the validation profile
	Claim *string

	// REQUIRED; Required value
	RequiredValue *string
}

// TokenIssuerProfile - Token issuer profile
type TokenIssuerProfile struct {
	// REQUIRED; The audience of the token issuer
	Audiences []*string

	// REQUIRED; The issuer of the token
	Ca *string

	// REQUIRED; The URL of the token issuer
	URL *string
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
	// REQUIRED; ChannelGroup is the name of the set to which this version belongs. Each version belongs to only a single set.
	ChannelGroup *string

	// REQUIRED; ID is the unique identifier of the version.
	ID *string

	// READ-ONLY; AvailableUpgrades is a list of version names the current version can be upgraded to.
	AvailableUpgrades []*string
}

