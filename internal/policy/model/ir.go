package model

type PolicyMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	CreatedBy   string            `json:"createdBy,omitempty"`
	CreatedAt   string            `json:"createdAt"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type PolicyMode struct {
	Type   PolicyModeType `json:"type"`
	Family Family         `json:"family"`
}

type AssignmentContext struct {
	NodeIDs  []string `json:"nodeIds,omitempty"`
	GroupIDs []string `json:"groupIds,omitempty"`
}

type RuleIR struct {
	ID          string          `json:"id"`
	Direction   Direction       `json:"direction"`
	Action      Verdict         `json:"action"`
	Source      NetworkSelector `json:"source"`
	Destination NetworkSelector `json:"destination"`
	Service     ServiceSelector `json:"service"`
	LogEnabled  bool            `json:"logEnabled"`
	Enabled     bool            `json:"enabled"`
	OrderIndex  int             `json:"orderIndex"`
	Description string          `json:"description,omitempty"`
}

type PolicyVersionIR struct {
	PolicyID      string                `json:"policyId"`
	VersionNumber int                   `json:"versionNumber"`
	EnvironmentID string                `json:"environmentId"`
	Metadata      PolicyMetadata        `json:"metadata"`
	Mode          PolicyMode            `json:"mode"`
	Defaults      PolicyDefaults        `json:"defaults"`
	SystemRules   SystemRuleOptions     `json:"systemRules"`
	Objects       ResolvedObjectCatalog `json:"objects"`
	InboundRules  []RuleIR              `json:"inboundRules"`
	OutboundRules []RuleIR              `json:"outboundRules"`
	Assignments   AssignmentContext     `json:"assignments"`
}
