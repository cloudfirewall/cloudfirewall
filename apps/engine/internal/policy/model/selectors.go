package model

type NetworkSelector struct {
	RefType       RefType                  `json:"refType"`
	ObjectID      string                   `json:"objectId,omitempty"`
	ObjectName    string                   `json:"objectName,omitempty"`
	LiteralValues []NormalizedNetworkValue `json:"literalValues,omitempty"`
	PseudoType    PseudoNetworkType        `json:"pseudoType,omitempty"`
}

type ServiceSelector struct {
	RefType    RefType        `json:"refType"`
	ObjectID   string         `json:"objectId,omitempty"`
	ObjectName string         `json:"objectName,omitempty"`
	Entries    []ServiceEntry `json:"entries,omitempty"`
}
