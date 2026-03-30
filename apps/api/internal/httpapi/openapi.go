package httpapi

import "github.com/cloudfirewall/cloudfirewall/apps/api/types"

func openAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Cloudfirewall API",
			"version":     "0.1.0",
			"description": "Enrollment, heartbeat, config delivery, and fleet status endpoints for cloudfirewall agents.",
		},
		"servers": []map[string]any{
			{"url": "http://localhost:8080"},
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "token",
				},
				"apiKeyAuth": map[string]any{
					"type": "apiKey",
					"in":   "header",
					"name": "X-API-Key",
				},
			},
			"schemas": map[string]any{
				"AdminLoginRequest":             schemaFor(types.AdminLoginRequest{}),
				"AdminLoginResponse":            schemaFor(types.AdminLoginResponse{}),
				"CreateEnrollmentTokenRequest":  schemaFor(types.CreateEnrollmentTokenRequest{}),
				"CreateEnrollmentTokenResponse": schemaFor(types.CreateEnrollmentTokenResponse{}),
				"EnrollAgentRequest":            schemaFor(types.EnrollAgentRequest{}),
				"EnrollAgentResponse":           schemaFor(types.EnrollAgentResponse{}),
				"AgentHeartbeatRequest":         schemaFor(types.AgentHeartbeatRequest{}),
				"AgentHeartbeatResponse":        schemaFor(types.AgentHeartbeatResponse{}),
				"AgentConfigResponse":           schemaFor(types.AgentConfigResponse{}),
				"AgentSummary":                  schemaFor(types.AgentSummary{}),
				"ListAgentsResponse":            schemaFor(types.ListAgentsResponse{}),
				"ErrorResponse": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"error": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
		"paths": map[string]any{
			"/healthz": map[string]any{
				"get": map[string]any{
					"summary": "Health check",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Service health",
						},
					},
				},
			},
			"/api/v1/admin/login": map[string]any{
				"post": map[string]any{
					"summary":     "Login as the configured admin user",
					"requestBody": jsonRequestBody("AdminLoginRequest"),
					"responses": map[string]any{
						"200": jsonResponse("AdminLoginResponse", "Admin session token"),
						"401": jsonResponse("ErrorResponse", "Invalid admin credentials"),
					},
				},
			},
			"/api/v1/enrollment-tokens": map[string]any{
				"post": map[string]any{
					"summary": "Create a one-time signed enrollment token",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"requestBody": jsonRequestBody("CreateEnrollmentTokenRequest"),
					"responses": map[string]any{
						"201": jsonResponse("CreateEnrollmentTokenResponse", "New enrollment token"),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key"),
					},
				},
			},
			"/api/v1/enroll": map[string]any{
				"post": map[string]any{
					"summary":     "Enroll an agent with an enrollment token",
					"requestBody": jsonRequestBody("EnrollAgentRequest"),
					"responses": map[string]any{
						"201": jsonResponse("EnrollAgentResponse", "Agent successfully enrolled"),
						"401": jsonResponse("ErrorResponse", "Invalid enrollment token"),
					},
				},
			},
			"/api/v1/agents/self/heartbeat": map[string]any{
				"post": map[string]any{
					"summary": "Record an authenticated agent heartbeat",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
					},
					"requestBody": jsonRequestBody("AgentHeartbeatRequest"),
					"responses": map[string]any{
						"200": jsonResponse("AgentHeartbeatResponse", "Heartbeat accepted"),
						"401": jsonResponse("ErrorResponse", "Missing or invalid bearer token"),
					},
				},
			},
			"/api/v1/agents/self/config": map[string]any{
				"get": map[string]any{
					"summary": "Fetch the current nftables configuration for the authenticated agent",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
					},
					"responses": map[string]any{
						"200": jsonResponse("AgentConfigResponse", "Current firewall config"),
						"401": jsonResponse("ErrorResponse", "Missing or invalid bearer token"),
					},
				},
			},
			"/api/v1/agents": map[string]any{
				"get": map[string]any{
					"summary": "List enrolled agents and their heartbeat state",
					"security": []map[string]any{
						{"bearerAuth": []any{}},
						{"apiKeyAuth": []any{}},
					},
					"responses": map[string]any{
						"200": jsonResponse("ListAgentsResponse", "Current fleet status"),
						"401": jsonResponse("ErrorResponse", "Missing or invalid admin bearer token or API key"),
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
			stringField("username"),
			stringField("password"),
		)
	case types.AdminLoginResponse:
		return objectSchema(
			stringField("authToken"),
		)
	case types.CreateEnrollmentTokenRequest:
		return objectSchema(
			intField("ttlSeconds"),
		)
	case types.CreateEnrollmentTokenResponse:
		return objectSchema(
			stringField("token"),
			stringField("tokenId"),
			stringField("expiresAt"),
		)
	case types.EnrollAgentRequest:
		return objectSchema(
			stringField("enrollmentToken"),
			stringField("agentName"),
			stringField("hostname"),
			stringField("agentVersion"),
		)
	case types.EnrollAgentResponse:
		return objectSchema(
			stringField("agentId"),
			stringField("authToken"),
			intField("heartbeatIntervalSeconds"),
			intField("configPollIntervalSeconds"),
		)
	case types.AgentHeartbeatRequest:
		return objectSchema(
			stringField("hostname"),
			stringField("agentVersion"),
			stringField("firewallVersion"),
		)
	case types.AgentHeartbeatResponse:
		return objectSchema(
			stringField("receivedAt"),
			boolField("online"),
		)
	case types.AgentConfigResponse:
		return objectSchema(
			stringField("version"),
			stringField("nftablesConfig"),
			stringField("updatedAt"),
		)
	case types.AgentSummary:
		return objectSchema(
			stringField("id"),
			stringField("name"),
			stringField("hostname"),
			stringField("agentVersion"),
			stringField("firewallVersion"),
			stringField("enrolledAt"),
			stringField("lastSeenAt"),
			boolField("online"),
		)
	case types.ListAgentsResponse:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agents": map[string]any{
					"type":  "array",
					"items": ref("AgentSummary"),
				},
			},
		}
	default:
		return map[string]any{"type": "object"}
	}
}

func objectSchema(fields ...map[string]any) map[string]any {
	properties := make(map[string]any, len(fields))
	for _, field := range fields {
		name := field["x-name"].(string)
		delete(field, "x-name")
		properties[name] = field
	}
	return map[string]any{
		"type":       "object",
		"properties": properties,
	}
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
