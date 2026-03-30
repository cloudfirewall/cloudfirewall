package model

type ObjectScope string

const (
	ObjectScopeWorkspace   ObjectScope = "WORKSPACE"
	ObjectScopeEnvironment ObjectScope = "ENVIRONMENT"
)

type NetworkObjectKind string

const (
	NetworkObjectIP             NetworkObjectKind = "IP"
	NetworkObjectCIDR           NetworkObjectKind = "CIDR"
	NetworkObjectList           NetworkObjectKind = "LIST"
	NetworkObjectPseudoExpanded NetworkObjectKind = "PSEUDO_EXPANDED"
)

type IPFamily string

const (
	IPFamilyV4    IPFamily = "IPV4"
	IPFamilyV6    IPFamily = "IPV6"
	IPFamilyMixed IPFamily = "MIXED"
)

type NormalizedNetworkValue struct {
	Family IPFamily `json:"family"`
	Value  string   `json:"value"`
}

type ResolvedNetworkObject struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Kind        NetworkObjectKind        `json:"kind"`
	Values      []NormalizedNetworkValue `json:"values"`
	SourceScope ObjectScope              `json:"sourceScope"`
}

type ServiceEntry struct {
	Protocol Protocol `json:"protocol"`
	Port     int      `json:"port"`
}

type ResolvedServiceObject struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Entries     []ServiceEntry `json:"entries"`
	SourceScope ObjectScope    `json:"sourceScope"`
}

type ResolvedObjectCatalog struct {
	Networks map[string]ResolvedNetworkObject `json:"networks"`
	Services map[string]ResolvedServiceObject `json:"services"`
}
