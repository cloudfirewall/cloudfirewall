package model

type AuthoringPolicyModel struct {
	PolicyID              string            `json:"policyId,omitempty"`
	VersionNumber         int               `json:"versionNumber,omitempty"`
	EnvironmentID         string            `json:"environmentId"`
	Name                  string            `json:"name"`
	Description           string            `json:"description,omitempty"`
	DefaultInboundAction  Verdict           `json:"defaultInboundAction"`
	DefaultOutboundAction Verdict           `json:"defaultOutboundAction"`
	SystemRules           SystemRuleOptions `json:"systemRules"`
	Rules                 []AuthoringRule   `json:"rules"`
}

type AuthoringRuleRef struct {
	Type       RefType           `json:"type"`
	ObjectID   string            `json:"objectId,omitempty"`
	ObjectName string            `json:"objectName,omitempty"`
	PseudoType PseudoNetworkType `json:"pseudoType,omitempty"`
	Literal    string            `json:"literal,omitempty"`
	Literals   []string          `json:"literals,omitempty"`
	Protocol   Protocol          `json:"protocol,omitempty"`
	Ports      []int             `json:"ports,omitempty"`
}

type AuthoringRule struct {
	ID          string           `json:"id"`
	Direction   Direction        `json:"direction"`
	Action      Verdict          `json:"action"`
	Source      AuthoringRuleRef `json:"source"`
	Destination AuthoringRuleRef `json:"destination"`
	Service     AuthoringRuleRef `json:"service"`
	LogEnabled  bool             `json:"logEnabled"`
	Enabled     bool             `json:"enabled"`
	OrderIndex  int              `json:"orderIndex"`
	Description string           `json:"description,omitempty"`
}
