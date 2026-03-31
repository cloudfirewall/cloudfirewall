package httpapi

import (
	"github.com/cloudfirewall/cloudfirewall/apps/api/types"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/policybuilder"
)

func openAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "Cloudfirewall API",
			"version": "0.1.0",
			"description": `REST API for the Cloudfirewall control plane.

## Authentication

Two authentication schemes are supported for admin endpoints:

| Scheme | Header | Value |
|---|---|---|
| Bearer token | ` + "`Authorization`" + ` | ` + "`Bearer <token>`" + ` from ` + "`POST /api/v1/admin/login`" + ` |
| API key | ` + "`X-API-Key`" + ` | Pre-configured static key |

Agent endpoints use a **separate bearer token** issued during enrollment via ` + "`POST /api/v1/enroll`" + `.

## Typical workflow

1. **Authenticate** — call ` + "`POST /api/v1/admin/login`" + ` to get an admin bearer token.
2. **Create a policy** — ` + "`POST /api/v1/policies`" + ` with a rule set.
3. **Apply it to the fleet** — ` + "`POST /api/v1/policies/{id}/apply`" + ` to mark it active.
4. **Enroll agents** — generate a token via ` + "`POST /api/v1/enrollment-tokens`" + `, then call ` + "`POST /api/v1/enroll`" + ` from each agent.
5. **Agents self-manage** — each agent polls ` + "`GET /api/v1/agents/self/config`" + ` and sends heartbeats to ` + "`POST /api/v1/agents/self/heartbeat`" + ` at the intervals returned during enrollment.`,
		},
		"servers": []map[string]any{
			{"url": "http://localhost:8080", "description": "Local development server"},
		},
		"tags": []map[string]any{
			{
				"name":        "Authentication",
				"description": "Admin login and session management. Obtain a bearer token here before calling any other admin endpoint.",
			},
			{
				"name":        "Enrollment",
				"description": "Issue single-use enrollment tokens and register new agents. Enrollment tokens expire after a configurable TTL (default 600 s). Each token can only be used once.",
			},
			{
				"name":        "Policies",
				"description": "Create, update, apply, and delete named firewall policies. Policies are versioned — the `version` field must be echoed back on updates to prevent lost-update conflicts. A policy must be explicitly applied before agents receive it.",
			},
			{
				"name":        "Agents",
				"description": "Fleet status, per-agent configuration delivery, and heartbeat recording. These endpoints are called by the agent process itself using the bearer token issued at enrollment.",
			},
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "opaque token",
					"description":  "Admin session token obtained from POST /api/v1/admin/login, or agent token obtained from POST /api/v1/enroll.",
				},
				"apiKeyAuth": map[string]any{
					"type":        "apiKey",
					"in":          "header",
					"name":        "X-API-Key",
					"description": "Pre-configured static API key. Alternative to bearer auth for admin endpoints.",
				},
			},
			"parameters": map[string]any{
				"policyId": map[string]any{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"description": "Unique identifier of the firewall policy.",
					"schema":      map[string]any{"type": "string"},
					"example":     "pol_01j9z4kxyz",
				},
			},
			"schemas": map[string]any{
				"AdminLoginRequest":             schemaFor(types.AdminLoginRequest{}),
				"AdminLoginResponse":            schemaFor(types.AdminLoginResponse{}),
				"CreateEnrollmentTokenRequest":  schemaFor(types.CreateEnrollmentTokenRequest{}),
				"CreateEnrollmentTokenResponse": schemaFor(types.CreateEnrollmentTokenResponse{}),
				"PolicyDraft":                   schemaFor(policybuilder.PolicyDraft{}),
				"PolicyRuleDraft":               schemaFor(policybuilder.RuleDraft{}),
				"CreatePolicyRequest":           schemaFor(types.CreateFirewallConfigRequest{}),
				"UpdatePolicyRequest":           schemaFor(types.UpdateFirewallConfigRequest{}),
				"UpdatePolicyResponse":          schemaFor(types.UpdateFirewallConfigResponse{}),
				"PolicySummary":                 schemaFor(types.FirewallConfigSummary{}),
				"ListPoliciesResponse":          schemaFor(types.ListFirewallConfigsResponse{}),
				"ApplyPolicyResponse":           schemaFor(types.ApplyFirewallConfigResponse{}),
				"EnrollAgentRequest":            schemaFor(types.EnrollAgentRequest{}),
				"EnrollAgentResponse":           schemaFor(types.EnrollAgentResponse{}),
				"AgentHeartbeatRequest":         schemaFor(types.AgentHeartbeatRequest{}),
				"AgentHeartbeatResponse":        schemaFor(types.AgentHeartbeatResponse{}),
				"AgentConfigResponse":           schemaFor(types.AgentConfigResponse{}),
				"AgentSummary":                  schemaFor(types.AgentSummary{}),
				"ListAgentsResponse":            schemaFor(types.ListAgentsResponse{}),
				"ErrorResponse": map[string]any{
					"type":        "object",
					"description": "A machine-readable error returned when a request cannot be completed.",
					"properties": map[string]any{
						"error": map[string]any{
							"type":        "string",
							"description": "Human-readable description of what went wrong.",
							"example":     "policy not found",
						},
					},
				},
			},
		},
		"paths": map[string]any{
			"/healthz": map[string]any{
				"get": map[string]any{
					"tags":        []string{"Authentication"},
					"operationId": "healthCheck",
					"summary":     "Health check",
					"description": "Returns `{\"status\":\"ok\"}` when the API server is reachable and ready to serve requests. Suitable for use as a liveness or readiness probe.",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Server is healthy.",
						},
					},
				},
			},
			"/api/v1/admin/login": map[string]any{
				"post": map[string]any{
					"tags":        []string{"Authentication"},
					"operationId": "adminLogin",
					"summary":     "Obtain an admin bearer token",
					"description": "Authenticate with the configured admin username and password. The returned `authToken` must be sent as `Authorization: Bearer <token>` on all subsequent admin API calls. Tokens do not expire automatically — call this again to rotate.",
					"requestBody": jsonRequestBody("AdminLoginRequest"),
					"responses": map[string]any{
						"200": jsonResponse("AdminLoginResponse", "Authentication successful. The response contains the admin bearer token."),
						"401": jsonResponse("ErrorResponse", "Invalid username or password."),
					},
				},
			},
			"/api/v1/enrollment-tokens": map[string]any{
				"post": map[string]any{
					"tags":        []string{"Enrollment"},
					"operationId": "createEnrollmentToken",
					"summary":     "Issue a one-time enrollment token",
					"description": "Generate a short-lived, single-use signed token that authorises one agent to enroll. Pass the token to the agent installer or the `POST /api/v1/enroll` endpoint. The token expires after `ttlSeconds` (default 600 s) and is invalidated on first use.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"requestBody": jsonRequestBody("CreateEnrollmentTokenRequest"),
					"responses": map[string]any{
						"201": jsonResponse("CreateEnrollmentTokenResponse", "Enrollment token created. Deliver this token to the agent before it expires."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key."),
					},
				},
			},
			"/api/v1/enroll": map[string]any{
				"post": map[string]any{
					"tags":        []string{"Enrollment"},
					"operationId": "enrollAgent",
					"summary":     "Register a new agent",
					"description": "Called by the agent installer to register the agent with the control plane. On success the response includes a bearer token the agent must use for all subsequent calls (`/heartbeat`, `/config`), plus the polling intervals it should respect.",
					"requestBody": jsonRequestBody("EnrollAgentRequest"),
					"responses": map[string]any{
						"201": jsonResponse("EnrollAgentResponse", "Agent enrolled. Store the `authToken` and use the returned intervals for heartbeat and config polling."),
						"401": jsonResponse("ErrorResponse", "Enrollment token is invalid, expired, or already used."),
					},
				},
			},
			"/api/v1/policies": map[string]any{
				"get": map[string]any{
					"tags":        []string{"Policies"},
					"operationId": "listPolicies",
					"summary":     "List all saved policies",
					"description": "Return all saved firewall policies ordered by last-updated time. Each item includes summary metadata and the `isActive` flag indicating whether the policy is currently being served to agents.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"responses": map[string]any{
						"200": jsonResponse("ListPoliciesResponse", "Saved policies returned successfully."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key."),
					},
				},
				"post": map[string]any{
					"tags":        []string{"Policies"},
					"operationId": "createPolicy",
					"summary":     "Create a new firewall policy",
					"description": "Save a new named policy with an optional rule set. The policy is persisted but **not yet applied** — agents continue to receive the previously active policy until you call `POST /api/v1/policies/{id}/apply`.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"requestBody": jsonRequestBody("CreatePolicyRequest"),
					"responses": map[string]any{
						"201": jsonResponse("PolicySummary", "Policy created. Use the returned `id` to update or apply it."),
						"400": jsonResponse("ErrorResponse", "Request body is missing, malformed, or contains invalid rules."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key."),
					},
				},
			},
			"/api/v1/policies/{id}": map[string]any{
				"parameters": []map[string]any{
					{"$ref": "#/components/parameters/policyId"},
				},
				"get": map[string]any{
					"tags":        []string{"Policies"},
					"operationId": "getPolicy",
					"summary":     "Fetch a policy by ID",
					"description": "Return a single saved policy including its full rule set, generated nftables output, and current status.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"responses": map[string]any{
						"200": jsonResponse("PolicySummary", "Policy found and returned."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key."),
						"404": jsonResponse("ErrorResponse", "No policy exists with the given ID."),
					},
				},
				"put": map[string]any{
					"tags":        []string{"Policies"},
					"operationId": "updatePolicy",
					"summary":     "Update a policy",
					"description": "Replace the rule set and metadata of an existing policy. The `version` field in the request body must match the value stored on the server — if they differ the update is rejected to prevent overwriting concurrent changes. The response contains the new version token to use in subsequent updates.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"requestBody": jsonRequestBody("UpdatePolicyRequest"),
					"responses": map[string]any{
						"200": jsonResponse("UpdatePolicyResponse", "Policy updated. Use the returned `version` in future update requests."),
						"400": jsonResponse("ErrorResponse", "Request body is invalid, or the `version` field does not match the stored version."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key."),
						"404": jsonResponse("ErrorResponse", "No policy exists with the given ID."),
					},
				},
				"delete": map[string]any{
					"tags":        []string{"Policies"},
					"operationId": "deletePolicy",
					"summary":     "Delete a policy",
					"description": "Permanently remove a saved policy. **Active policies** (those currently served to agents) cannot be deleted — apply a different policy first.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"responses": map[string]any{
						"204": map[string]any{"description": "Policy deleted successfully. No content is returned."},
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key."),
						"404": jsonResponse("ErrorResponse", "No policy exists with the given ID."),
						"409": jsonResponse("ErrorResponse", "Policy is currently active and cannot be deleted. Apply another policy first."),
					},
				},
			},
			"/api/v1/policies/{id}/apply": map[string]any{
				"parameters": []map[string]any{
					{"$ref": "#/components/parameters/policyId"},
				},
				"post": map[string]any{
					"tags":        []string{"Policies"},
					"operationId": "applyPolicy",
					"summary":     "Apply a policy to the fleet",
					"description": "Mark the given policy as the active fleet policy. All enrolled agents will receive the updated nftables configuration on their next config poll (within one `configPollIntervalSeconds` cycle).",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"responses": map[string]any{
						"200": jsonResponse("ApplyPolicyResponse", "Policy applied. Agents will receive the new configuration on their next poll."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key."),
						"404": jsonResponse("ErrorResponse", "No policy exists with the given ID."),
					},
				},
			},
			"/api/v1/policies/active": map[string]any{
				"put": map[string]any{
					"tags":        []string{"Policies"},
					"operationId": "updateActivePolicy",
					"summary":     "Directly replace the active nftables config",
					"description": "Bypass the versioned policy workflow and push a raw nftables configuration directly to the active slot. Agents will receive the new config on their next poll. Use `POST /api/v1/policies` + `apply` instead for auditable, versioned deployments.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"requestBody": jsonRequestBody("UpdatePolicyRequest"),
					"responses": map[string]any{
						"200": jsonResponse("UpdatePolicyResponse", "Active configuration updated."),
						"400": jsonResponse("ErrorResponse", "Request body is missing or malformed."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key."),
					},
				},
			},
			"/api/v1/agents": map[string]any{
				"get": map[string]any{
					"tags":        []string{"Agents"},
					"operationId": "listAgents",
					"summary":     "List enrolled agents",
					"description": "Return all enrolled agents with their last-seen heartbeat time, reported versions, and derived online status. An agent is considered online if its last heartbeat was received within two heartbeat intervals.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"responses": map[string]any{
						"200": jsonResponse("ListAgentsResponse", "Current fleet status returned."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key."),
					},
				},
			},
			"/api/v1/agents/self/heartbeat": map[string]any{
				"post": map[string]any{
					"tags":        []string{"Agents"},
					"operationId": "agentHeartbeat",
					"summary":     "Record an agent heartbeat",
					"description": "Called by the agent at the interval returned during enrollment (`heartbeatIntervalSeconds`). Updates the agent's last-seen timestamp and reported versions. Missing heartbeats cause the agent to appear offline in `GET /api/v1/agents`.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
					},
					"requestBody": jsonRequestBody("AgentHeartbeatRequest"),
					"responses": map[string]any{
						"200": jsonResponse("AgentHeartbeatResponse", "Heartbeat recorded."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid agent bearer token."),
					},
				},
			},
			"/api/v1/agents/self/config": map[string]any{
				"get": map[string]any{
					"tags":        []string{"Agents"},
					"operationId": "getAgentConfig",
					"summary":     "Fetch the active config for this agent",
					"description": "Called by the agent at the interval returned during enrollment (`configPollIntervalSeconds`). Returns the currently active nftables configuration and its version string. The agent should compare `version` against the locally applied version and only re-apply the ruleset if they differ.",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
					},
					"responses": map[string]any{
						"200": jsonResponse("AgentConfigResponse", "Current active configuration returned."),
						"401": jsonResponse("ErrorResponse", "Missing or invalid agent bearer token."),
					},
				},
			},
		},
	}
}

func jsonRequestBody(schemaName string) map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": ref(schemaName),
			},
		},
	}
}

func jsonResponse(schemaName, description string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": ref(schemaName),
			},
		},
	}
}

func ref(schemaName string) map[string]any {
	return map[string]any{
		"$ref": "#/components/schemas/" + schemaName,
	}
}

func schemaFor(v any) map[string]any {
	switch v.(type) {
	case types.AdminLoginRequest:
		return objectSchema(
			"Admin credentials.",
			descField(stringField("username"), "Admin username configured on the server."),
			descField(stringField("password"), "Admin password configured on the server."),
		)
	case types.AdminLoginResponse:
		return objectSchema(
			"Admin session token.",
			descField(stringField("authToken"), "Bearer token to include in the `Authorization` header of subsequent admin requests."),
		)
	case types.CreateEnrollmentTokenRequest:
		return objectSchema(
			"Parameters for the enrollment token to be created.",
			descField(intField("ttlSeconds"), "How long the token remains valid, in seconds. Defaults to 600 (10 minutes) if omitted."),
		)
	case types.CreateEnrollmentTokenResponse:
		return objectSchema(
			"A freshly issued enrollment token.",
			descField(stringField("token"), "Signed enrollment token to pass to the agent installer."),
			descField(stringField("tokenId"), "Opaque identifier for this token, useful for auditing."),
			descField(stringField("expiresAt"), "ISO 8601 timestamp after which the token can no longer be used."),
		)
	case types.UpdateFirewallConfigRequest:
		return objectSchema(
			"Request body for creating or replacing a firewall policy.",
			descField(stringField("name"), "Human-readable name for the policy."),
			descField(stringField("version"), "Opaque version token. On updates this must match the stored value — if it differs the request is rejected."),
			descField(stringField("nftablesConfig"), "Raw nftables configuration. Leave empty to have the server generate it from the `policy` rule set."),
			descField(refField("policy", "PolicyDraft"), "Structured rule set from which the nftables configuration is derived."),
		)
	case types.CreateFirewallConfigRequest:
		return objectSchema(
			"Request body for creating a new firewall policy.",
			descField(stringField("name"), "Human-readable name for the policy."),
			descField(stringField("version"), "Initial version token. Use an empty string or omit for new policies."),
			descField(stringField("nftablesConfig"), "Raw nftables configuration. Leave empty to have the server generate it from the `policy` rule set."),
			descField(refField("policy", "PolicyDraft"), "Structured rule set from which the nftables configuration is derived."),
		)
	case types.UpdateFirewallConfigResponse:
		return objectSchema(
			"Confirmation of a successful policy save.",
			descField(stringField("id"), "Unique identifier of the policy."),
			descField(stringField("name"), "Human-readable name of the policy."),
			descField(stringField("version"), "New version token. Store this and echo it back in the next update request."),
			descField(stringField("updatedAt"), "ISO 8601 timestamp of when the policy was last saved."),
		)
	case types.FirewallConfigSummary:
		return objectSchema(
			"Full metadata and content of a saved firewall policy.",
			descField(stringField("id"), "Unique identifier of the policy."),
			descField(stringField("name"), "Human-readable name of the policy."),
			descField(stringField("version"), "Opaque version token. Echo this back when updating the policy."),
			descField(stringField("updatedAt"), "ISO 8601 timestamp of the last modification."),
			descField(boolField("isActive"), "Whether this policy is currently being served to agents."),
			descField(stringField("nftablesConfig"), "Generated nftables configuration derived from the rule set. Empty until the policy has been saved at least once."),
			descField(refField("policy", "PolicyDraft"), "Structured rule set for this policy."),
		)
	case types.ListFirewallConfigsResponse:
		return map[string]any{
			"type":        "object",
			"description": "Paginated list of saved firewall policies.",
			"properties": map[string]any{
				"configs": map[string]any{
					"type":        "array",
					"description": "All saved policies ordered by last-updated time, newest first.",
					"items":       ref("PolicySummary"),
				},
			},
		}
	case types.ApplyFirewallConfigResponse:
		return map[string]any{
			"type":        "object",
			"description": "Confirmation that the policy has been set as the active fleet policy.",
			"properties": map[string]any{
				"config": map[string]any{
					"description": "The policy that is now active.",
					"$ref":        "#/components/schemas/PolicySummary",
				},
			},
		}
	case policybuilder.PolicyDraft:
		return objectSchema(
			"A structured firewall policy composed of ordered rules and default actions.",
			descField(stringField("policyId"), "Stable identifier for the policy, used when tracking versions across environments."),
			descField(intField("versionNumber"), "Monotonically increasing version counter managed by the server."),
			descField(stringField("environmentId"), "Optional identifier grouping policies by deployment environment."),
			descField(stringField("name"), "Human-readable policy name."),
			descField(stringField("description"), "Optional free-text description of the policy's purpose."),
			descField(stringField("defaultInboundAction"), "Action applied to inbound traffic that matches no rule. One of `ALLOW`, `DENY`, or `REJECT`."),
			descField(stringField("defaultOutboundAction"), "Action applied to outbound traffic that matches no rule. One of `ALLOW`, `DENY`, or `REJECT`."),
			descField(boolField("allowLoopback"), "When true, loopback (127.0.0.0/8, ::1) traffic is always permitted, regardless of rules."),
			descField(boolField("allowEstablishedRelated"), "When true, packets belonging to established or related connections are always permitted (CONNTRACK)."),
			descField(arrayRefField("rules", "PolicyRuleDraft"), "Ordered list of firewall rules evaluated top-to-bottom."),
		)
	case policybuilder.RuleDraft:
		return objectSchema(
			"A single firewall rule within a policy.",
			descField(stringField("id"), "Stable identifier for the rule, unique within the policy."),
			descField(stringField("direction"), "Traffic direction this rule applies to. One of `INBOUND` or `OUTBOUND`."),
			descField(stringField("action"), "Action to take when traffic matches. One of `ALLOW`, `DENY`, or `REJECT`."),
			descField(stringField("peerType"), "Source/destination peer type. One of `PUBLIC_INTERNET`, `OFFICE_IPS`, `CIDR`, or `THIS_NODE`."),
			descField(stringField("peerValue"), "CIDR range when `peerType` is `CIDR`, e.g. `203.0.113.0/24`. Ignored for other peer types."),
			descField(stringField("protocol"), "IP protocol this rule covers. One of `TCP` or `UDP`."),
			descField(intArrayField("ports"), "Port numbers matched by this rule. An empty array matches all ports. A two-element array is interpreted as an inclusive range [start, end]."),
			descField(boolField("logEnabled"), "When true, matching packets are written to the system log before the action is applied."),
			descField(boolField("enabled"), "When false the rule is stored but not compiled into the active nftables configuration."),
			descField(intField("orderIndex"), "Evaluation order relative to other rules in the same policy. Lower values are evaluated first."),
			descField(stringField("description"), "Optional human-readable note describing the rule's intent."),
		)
	case types.EnrollAgentRequest:
		return objectSchema(
			"Agent identity and credentials used to register with the control plane.",
			descField(stringField("enrollmentToken"), "Single-use signed token obtained from POST /api/v1/enrollment-tokens."),
			descField(stringField("agentName"), "Logical name for this agent, used for display in the admin UI."),
			descField(stringField("hostname"), "Hostname of the machine the agent is running on."),
			descField(stringField("agentVersion"), "Semantic version string of the agent binary, e.g. `1.2.3`."),
		)
	case types.EnrollAgentResponse:
		return objectSchema(
			"Credentials and polling configuration returned after successful enrollment.",
			descField(stringField("agentId"), "Stable unique identifier assigned to this agent."),
			descField(stringField("authToken"), "Bearer token the agent must use for all subsequent API calls."),
			descField(intField("heartbeatIntervalSeconds"), "How often the agent should call POST /api/v1/agents/self/heartbeat."),
			descField(intField("configPollIntervalSeconds"), "How often the agent should call GET /api/v1/agents/self/config."),
		)
	case types.AgentHeartbeatRequest:
		return objectSchema(
			"Agent liveness signal with current version metadata.",
			descField(stringField("hostname"), "Current hostname of the agent machine."),
			descField(stringField("agentVersion"), "Semantic version of the running agent binary."),
			descField(stringField("firewallVersion"), "Version string of the nftables configuration currently applied on this agent."),
		)
	case types.AgentHeartbeatResponse:
		return objectSchema(
			"Acknowledgement of a recorded heartbeat.",
			descField(stringField("receivedAt"), "ISO 8601 timestamp at which the heartbeat was recorded by the server."),
			descField(boolField("online"), "Server-side view of whether this agent is considered online at the time of the heartbeat."),
		)
	case types.AgentConfigResponse:
		return objectSchema(
			"The currently active firewall configuration the agent should apply.",
			descField(stringField("version"), "Version string of this configuration. Compare against the locally applied version to decide whether a re-apply is needed."),
			descField(stringField("nftablesConfig"), "Full nftables configuration to apply on the agent host."),
			descField(stringField("updatedAt"), "ISO 8601 timestamp of when this configuration was last changed."),
		)
	case types.AgentSummary:
		return objectSchema(
			"Summary of an enrolled agent and its current status.",
			descField(stringField("id"), "Unique identifier assigned at enrollment."),
			descField(stringField("name"), "Logical name provided at enrollment."),
			descField(stringField("hostname"), "Last-reported hostname."),
			descField(stringField("agentVersion"), "Last-reported agent binary version."),
			descField(stringField("firewallVersion"), "Version of the nftables configuration the agent last reported as applied."),
			descField(stringField("enrolledAt"), "ISO 8601 timestamp of when the agent enrolled."),
			descField(stringField("lastSeenAt"), "ISO 8601 timestamp of the most recent heartbeat received."),
			descField(boolField("online"), "True if a heartbeat was received within the expected interval window."),
		)
	case types.ListAgentsResponse:
		return map[string]any{
			"type":        "object",
			"description": "The current fleet roster.",
			"properties": map[string]any{
				"agents": map[string]any{
					"type":        "array",
					"description": "All enrolled agents.",
					"items":       ref("AgentSummary"),
				},
			},
		}
	default:
		return map[string]any{"type": "object"}
	}
}

// objectSchema builds an OpenAPI object schema from a list of property field maps.
// The first argument is an optional description string for the object itself.
func objectSchema(description string, fields ...map[string]any) map[string]any {
	properties := make(map[string]any, len(fields))
	for _, field := range fields {
		name := field["x-name"].(string)
		delete(field, "x-name")
		properties[name] = field
	}
	return map[string]any{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}
}

// descField attaches a description to any field map returned by the field helpers.
func descField(field map[string]any, description string) map[string]any {
	field["description"] = description
	return field
}

func stringField(name string) map[string]any {
	return map[string]any{"x-name": name, "type": "string"}
}

func intField(name string) map[string]any {
	return map[string]any{"x-name": name, "type": "integer"}
}

func boolField(name string) map[string]any {
	return map[string]any{"x-name": name, "type": "boolean"}
}

func refField(name, schemaName string) map[string]any {
	return map[string]any{"x-name": name, "$ref": "#/components/schemas/" + schemaName}
}

func arrayRefField(name, schemaName string) map[string]any {
	return map[string]any{
		"x-name": name,
		"type":   "array",
		"items":  ref(schemaName),
	}
}

func intArrayField(name string) map[string]any {
	return map[string]any{
		"x-name": name,
		"type":   "array",
		"items": map[string]any{
			"type": "integer",
		},
	}
}
