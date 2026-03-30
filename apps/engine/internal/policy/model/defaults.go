package model

type PolicyDefaults struct {
	Inbound  Verdict `json:"inbound"`
	Outbound Verdict `json:"outbound"`
}

type SystemRuleOptions struct {
	AllowLoopback           bool `json:"allowLoopback"`
	AllowEstablishedRelated bool `json:"allowEstablishedRelated"`
}
