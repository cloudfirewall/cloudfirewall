package model

type Verdict string

const (
	VerdictAllow  Verdict = "ALLOW"
	VerdictDeny   Verdict = "DENY"
	VerdictReject Verdict = "REJECT"
)

type Direction string

const (
	DirectionInbound  Direction = "INBOUND"
	DirectionOutbound Direction = "OUTBOUND"
)

type PolicyModeType string

const (
	PolicyModeHostFiltering PolicyModeType = "HOST_FILTERING"
)

type Family string

const (
	FamilyINET Family = "INET"
)

type Protocol string

const (
	ProtocolTCP    Protocol = "TCP"
	ProtocolUDP    Protocol = "UDP"
	ProtocolICMP   Protocol = "ICMP"
	ProtocolICMPv6 Protocol = "ICMPV6"
	ProtocolAny    Protocol = "ANY"
)

type RefType string

const (
	RefTypeObject  RefType = "OBJECT"
	RefTypeLiteral RefType = "LITERAL"
	RefTypePseudo  RefType = "PSEUDO"
)

type PseudoNetworkType string

const (
	PseudoPublicInternet PseudoNetworkType = "PUBLIC_INTERNET"
	PseudoThisNode       PseudoNetworkType = "THIS_NODE"
	PseudoLoopback       PseudoNetworkType = "LOOPBACK"
)
