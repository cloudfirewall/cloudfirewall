import { startTransition, useEffect, useState, type FormEvent } from "react";

type AgentSummary = {
	id: string;
	name: string;
	hostname: string;
	agentVersion: string;
	firewallVersion: string;
	enrolledAt: string;
	lastSeenAt?: string;
	online: boolean;
};

type ListAgentsResponse = { agents: AgentSummary[] };

type CreateEnrollmentTokenResponse = {
	token: string;
	tokenId: string;
	expiresAt: string;
};

type PolicyRuleDraft = {
	id: string;
	direction: "INBOUND" | "OUTBOUND";
	action: "ALLOW" | "DENY" | "REJECT";
	peerType: "PUBLIC_INTERNET" | "OFFICE_IPS" | "CIDR" | "THIS_NODE";
	peerValue?: string;
	protocol: "TCP" | "UDP";
	ports: number[];
	logEnabled: boolean;
	enabled: boolean;
	orderIndex: number;
	description?: string;
};

type PolicyDraft = {
	policyId?: string;
	versionNumber?: number;
	environmentId?: string;
	name: string;
	description?: string;
	defaultInboundAction: "ALLOW" | "DENY" | "REJECT";
	defaultOutboundAction: "ALLOW" | "DENY" | "REJECT";
	allowLoopback: boolean;
	allowEstablishedRelated: boolean;
	rules: PolicyRuleDraft[];
};

type FirewallConfigSummary = {
	id: string;
	name: string;
	version: string;
	updatedAt: string;
	isActive: boolean;
	nftablesConfig?: string;
	policy?: PolicyDraft;
};

type ListFirewallConfigsResponse = {
	configs: FirewallConfigSummary[];
};

type DashboardView = "agents" | "configs";
type ConfigDetailTab = "overview" | "rules";
type ConfigScreen = "list" | "details" | "rule";

const API_BASE = import.meta.env.VITE_API_BASE ?? "";
const POLL_INTERVAL_MS = 5000;
const TOKEN_KEY = "cloudfirewall_admin_token";
const CONFIGS_PAGE_SIZE = 6;

const emptyPolicy = (): PolicyDraft => ({
	name: "",
	description: "",
	defaultInboundAction: "DENY",
	defaultOutboundAction: "ALLOW",
	allowLoopback: true,
	allowEstablishedRelated: true,
	rules: [],
});

export default function App() {
	const [authToken, setAuthToken] = useState(() => window.localStorage.getItem(TOKEN_KEY) ?? "");
	const [username, setUsername] = useState("admin");
	const [password, setPassword] = useState("");
	const [activeView, setActiveView] = useState<DashboardView>("agents");
	const [configScreen, setConfigScreen] = useState<ConfigScreen>("list");
	const [configDetailTab, setConfigDetailTab] = useState<ConfigDetailTab>("overview");
	const [configListPage, setConfigListPage] = useState(1);
	const [agents, setAgents] = useState<AgentSummary[]>([]);
	const [configs, setConfigs] = useState<FirewallConfigSummary[]>([]);
	const [selectedConfigID, setSelectedConfigID] = useState("");
	const [selectedRuleIndex, setSelectedRuleIndex] = useState(-1);
	const [policyEditor, setPolicyEditor] = useState<PolicyDraft>(emptyPolicy);
	const [editorVersion, setEditorVersion] = useState("");
	const [generatedToken, setGeneratedToken] = useState<CreateEnrollmentTokenResponse | null>(null);
	const [error, setError] = useState("");
	const [configError, setConfigError] = useState("");
	const [authError, setAuthError] = useState("");
	const [tokenError, setTokenError] = useState("");
	const [lastUpdated, setLastUpdated] = useState("");
	const [isLoading, setIsLoading] = useState(true);
	const [isAuthenticating, setIsAuthenticating] = useState(false);
	const [isCreatingToken, setIsCreatingToken] = useState(false);
	const [isSavingConfig, setIsSavingConfig] = useState(false);
	const [isApplyingConfig, setIsApplyingConfig] = useState(false);
	const [isDeletingConfig, setIsDeletingConfig] = useState(false);

	useEffect(() => {
		if (!authToken) {
			setAgents([]);
			setConfigs([]);
			setIsLoading(false);
			return;
		}

		let cancelled = false;
		setIsLoading(true);

		async function loadInitialData() {
			try {
				const [agentsResponse, configsResponse] = await Promise.all([
					fetch(`${API_BASE}/api/v1/agents`, { headers: { Authorization: `Bearer ${authToken}` } }),
					fetch(`${API_BASE}/api/v1/firewall-configs`, { headers: { Authorization: `Bearer ${authToken}` } }),
				]);
				if (!agentsResponse.ok || !configsResponse.ok) {
					const status =
						agentsResponse.status === 401 || configsResponse.status === 401
							? 401
							: Math.max(agentsResponse.status, configsResponse.status);
					if (status === 401) {
						window.localStorage.removeItem(TOKEN_KEY);
						setAuthToken("");
						throw new Error("Session expired. Please log in again.");
					}
					throw new Error(`Request failed with status ${status}`);
				}

				const agentPayload = (await agentsResponse.json()) as ListAgentsResponse;
				const configPayload = (await configsResponse.json()) as ListFirewallConfigsResponse;
				if (cancelled) return;

				startTransition(() => {
					setAgents(agentPayload.agents);
					setConfigs(configPayload.configs);
					setError("");
					setLastUpdated(new Date().toLocaleTimeString());

					if (configPayload.configs.length > 0) {
						const currentSelected = configPayload.configs.find((config) => config.id === selectedConfigID);
						if (currentSelected) {
							syncConfigEditor(currentSelected);
							return;
						}

						const active = configPayload.configs.find((config) => config.isActive) ?? configPayload.configs[0];
						syncConfigEditor(active);
					}
				});
			} catch (loadError) {
				if (cancelled) return;
				startTransition(() => setError(loadError instanceof Error ? loadError.message : "Failed to load dashboard"));
			} finally {
				if (!cancelled) setIsLoading(false);
			}
		}

		void loadInitialData();
		return () => {
			cancelled = true;
		};
	}, [authToken]);

	useEffect(() => {
		if (!authToken || activeView !== "agents") {
			return;
		}

		let cancelled = false;

		async function loadAgents() {
			try {
				const response = await fetch(`${API_BASE}/api/v1/agents`, {
					headers: { Authorization: `Bearer ${authToken}` },
				});
				if (!response.ok) {
					if (response.status === 401) {
						window.localStorage.removeItem(TOKEN_KEY);
						setAuthToken("");
						throw new Error("Session expired. Please log in again.");
					}
					throw new Error(`Request failed with status ${response.status}`);
				}

				const payload = (await response.json()) as ListAgentsResponse;
				if (cancelled) return;

				startTransition(() => {
					setAgents(payload.agents);
					setError("");
					setLastUpdated(new Date().toLocaleTimeString());
				});
			} catch (loadError) {
				if (cancelled) return;
				startTransition(() => setError(loadError instanceof Error ? loadError.message : "Failed to load agents"));
			}
		}

		const timer = window.setInterval(() => void loadAgents(), POLL_INTERVAL_MS);
		return () => {
			cancelled = true;
			window.clearInterval(timer);
		};
	}, [authToken, activeView]);

	const onlineAgents = agents.filter((agent) => agent.online).length;
	const selectedConfig = configs.find((config) => config.id === selectedConfigID) ?? null;
	const selectedRule = selectedRuleIndex >= 0 ? policyEditor.rules[selectedRuleIndex] ?? null : null;
	const isPolicyEditable = !selectedConfig || Boolean(selectedConfig.policy);
	const configPageCount = Math.max(1, Math.ceil(configs.length / CONFIGS_PAGE_SIZE));
	const pagedConfigs = configs.slice((configListPage - 1) * CONFIGS_PAGE_SIZE, configListPage * CONFIGS_PAGE_SIZE);

	useEffect(() => {
		if (configListPage > configPageCount) {
			setConfigListPage(configPageCount);
		}
	}, [configListPage, configPageCount]);

	function syncConfigEditor(config: FirewallConfigSummary) {
		setSelectedConfigID(config.id);
		setEditorVersion(config.version);
		if (config.policy) {
			setPolicyEditor(config.policy);
			setSelectedRuleIndex(config.policy.rules.length > 0 ? 0 : -1);
		} else {
			setPolicyEditor({
				...emptyPolicy(),
				name: config.name,
			});
			setSelectedRuleIndex(-1);
		}
		setConfigError("");
	}

	function selectConfig(config: FirewallConfigSummary) {
		setConfigDetailTab("overview");
		syncConfigEditor(config);
	}

	function openConfigList() {
		setActiveView("configs");
		setConfigScreen("list");
	}

	function openConfigDetails(config: FirewallConfigSummary) {
		selectConfig(config);
		setConfigScreen("details");
	}

	function openRuleDetails(index: number) {
		setSelectedRuleIndex(index);
		setConfigScreen("rule");
	}

	function startNewConfig() {
		setActiveView("configs");
		setConfigScreen("details");
		setConfigDetailTab("overview");
		setSelectedConfigID("");
		setSelectedRuleIndex(-1);
		setEditorVersion("");
		setPolicyEditor(emptyPolicy());
		setConfigError("");
	}

	async function authorizedFetch(path: string, init?: RequestInit) {
		const response = await fetch(`${API_BASE}${path}`, {
			...init,
			headers: {
				Authorization: `Bearer ${authToken}`,
				...(init?.headers ?? {}),
			},
		});
		if (response.status === 401) {
			window.localStorage.removeItem(TOKEN_KEY);
			setAuthToken("");
			throw new Error("Session expired. Please log in again.");
		}
		return response;
	}

	async function reloadConfigs() {
		const response = await authorizedFetch("/api/v1/firewall-configs");
		const payload = (await response.json()) as ListFirewallConfigsResponse;
		setConfigs(payload.configs);
		return payload.configs;
	}

	async function handleLogin(event: FormEvent<HTMLFormElement>) {
		event.preventDefault();
		setIsAuthenticating(true);
		setAuthError("");
		try {
			const response = await fetch(`${API_BASE}/api/v1/admin/login`, {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ username, password }),
			});
			if (!response.ok) {
				throw new Error(response.status === 401 ? "Invalid username or password" : `Login failed with status ${response.status}`);
			}
			const payload = (await response.json()) as { authToken: string };
			window.localStorage.setItem(TOKEN_KEY, payload.authToken);
			setAuthToken(payload.authToken);
			setPassword("");
		} catch (loginError) {
			setAuthError(loginError instanceof Error ? loginError.message : "Login failed");
		} finally {
			setIsAuthenticating(false);
		}
	}

	function handleLogout() {
		window.localStorage.removeItem(TOKEN_KEY);
		setAuthToken("");
		setAgents([]);
		setConfigs([]);
		setSelectedConfigID("");
		setPolicyEditor(emptyPolicy());
		setEditorVersion("");
		setLastUpdated("");
		setError("");
	}

	async function handleCreateEnrollmentToken() {
		setIsCreatingToken(true);
		setTokenError("");
		try {
			const response = await authorizedFetch("/api/v1/enrollment-tokens", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ ttlSeconds: 600 }),
			});
			if (!response.ok) throw new Error(`Token creation failed with status ${response.status}`);
			setGeneratedToken((await response.json()) as CreateEnrollmentTokenResponse);
		} catch (createError) {
			setTokenError(createError instanceof Error ? createError.message : "Failed to create token");
		} finally {
			setIsCreatingToken(false);
		}
	}

	async function handleSavePolicy(event: FormEvent<HTMLFormElement>) {
		event.preventDefault();
		setIsSavingConfig(true);
		setConfigError("");

		try {
			const payload = {
				name: policyEditor.name,
				version: editorVersion,
				nftablesConfig: "",
				policy: policyEditor,
			};
			const response = selectedConfigID
				? await authorizedFetch(`/api/v1/firewall-configs/${selectedConfigID}`, {
						method: "PUT",
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify(payload),
					})
				: await authorizedFetch("/api/v1/firewall-configs", {
						method: "POST",
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify(payload),
					});
			if (!response.ok) throw new Error(await extractError(response, "Failed to save firewall policy"));
			const saved = (await response.json()) as { id?: string; version: string };
			const nextConfigs = await reloadConfigs();
			const next = nextConfigs.find((config) => config.id === (saved.id ?? selectedConfigID)) ?? nextConfigs[0];
			if (next) selectConfig(next);
		} catch (saveError) {
			setConfigError(saveError instanceof Error ? saveError.message : "Failed to save firewall policy");
		} finally {
			setIsSavingConfig(false);
		}
	}

	async function handleApplyConfig() {
		if (!selectedConfigID) return;
		setIsApplyingConfig(true);
		setConfigError("");
		try {
			const response = await authorizedFetch(`/api/v1/firewall-configs/${selectedConfigID}/apply`, { method: "POST" });
			if (!response.ok) throw new Error(await extractError(response, "Failed to apply firewall config"));
			const nextConfigs = await reloadConfigs();
			const next = nextConfigs.find((config) => config.id === selectedConfigID);
			if (next) selectConfig(next);
		} catch (applyError) {
			setConfigError(applyError instanceof Error ? applyError.message : "Failed to apply firewall config");
		} finally {
			setIsApplyingConfig(false);
		}
	}

	async function handleDeleteConfig() {
		if (!selectedConfigID) return;
		setIsDeletingConfig(true);
		setConfigError("");
		try {
			const response = await authorizedFetch(`/api/v1/firewall-configs/${selectedConfigID}`, { method: "DELETE" });
			if (!response.ok) throw new Error(await extractError(response, "Failed to delete firewall config"));
			const nextConfigs = await reloadConfigs();
			if (nextConfigs.length > 0) {
				selectConfig(nextConfigs[0]);
			} else {
				startNewConfig();
			}
		} catch (deleteError) {
			setConfigError(deleteError instanceof Error ? deleteError.message : "Failed to delete firewall config");
		} finally {
			setIsDeletingConfig(false);
		}
	}

	function updateRule(index: number, patch: Partial<PolicyRuleDraft>) {
		setPolicyEditor((current) => ({
			...current,
			rules: current.rules.map((rule, ruleIndex) => (ruleIndex === index ? { ...rule, ...patch } : rule)),
		}));
	}

	function addRule() {
		const nextIndex = policyEditor.rules.length;
		setPolicyEditor((current) => ({
			...current,
			rules: current.rules.concat({
				id: `rule-${current.rules.length + 1}`,
				direction: "INBOUND",
				action: "ALLOW",
				peerType: "PUBLIC_INTERNET",
				protocol: "TCP",
				ports: [443],
				logEnabled: false,
				enabled: true,
				orderIndex: (current.rules.length + 1) * 10,
				description: "",
			}),
		}));
		setConfigScreen("rule");
		setSelectedRuleIndex(nextIndex);
	}

	function removeRule(index: number) {
		setPolicyEditor((current) => ({
			...current,
			rules: current.rules.filter((_, ruleIndex) => ruleIndex !== index),
		}));
		setSelectedRuleIndex((current) => {
			if (policyEditor.rules.length <= 1) return -1;
			if (current > index) return current - 1;
			if (current === index) return Math.max(0, index - 1);
			return current;
		});
		setConfigScreen("details");
		setConfigDetailTab("rules");
	}

	if (!authToken) {
		return (
			<div className="shell login-shell">
				<div className="login-panel">
					<div className="login-copy">
						<p className="eyebrow">Cloudfirewall</p>
						<h1>Firewall control, organized around the fleet.</h1>
						<p>Sign in to issue agent enrollments, review firewall posture, and manage firewall rules from one admin workspace.</p>
					</div>
					<form className="login-card" onSubmit={handleLogin}>
						<label className="field">
							<span>Username</span>
							<input value={username} onChange={(event) => setUsername(event.target.value)} />
						</label>
						<label className="field">
							<span>Password</span>
							<input type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
						</label>
						<button type="submit" className="primary-button" disabled={isAuthenticating}>
							{isAuthenticating ? "Signing in..." : "Login"}
						</button>
						{authError ? <div className="notice error">{authError}</div> : null}
					</form>
				</div>
			</div>
		);
	}

	return (
		<div className="shell">
			<aside className="sidebar">
				<div className="brand-block">
					<p className="eyebrow">Cloudfirewall</p>
					<h1>Admin Portal</h1>
					<p>Policy-driven firewall operations for agents and fleet rollouts.</p>
				</div>

				<nav className="sidebar-nav" aria-label="Primary">
					<button
						type="button"
						className={`nav-item ${activeView === "agents" ? "active" : ""}`}
						onClick={() => setActiveView("agents")}
					>
						<span className="nav-label">Agents</span>
						<span className="nav-meta">{agents.length}</span>
					</button>
					<button
						type="button"
						className={`nav-item ${activeView === "configs" ? "active" : ""}`}
						onClick={openConfigList}
					>
						<span className="nav-label">Firewall Configs</span>
						<span className="nav-meta">{configs.length}</span>
					</button>
				</nav>

				<div className="sidebar-footer">
					<div className="sidebar-stat">
						<span>Online agents</span>
						<strong>{onlineAgents}</strong>
					</div>
					<div className="sidebar-stat">
						<span>Last refresh</span>
						<strong>{lastUpdated || "Waiting"}</strong>
					</div>
				</div>
			</aside>

			<main className="main-stage">
				<header className="topbar">
					<div>
						<p className="section-kicker">{activeView === "agents" ? "Fleet" : "Firewall authoring"}</p>
						<h2>
							{activeView === "agents"
								? "Agents"
								: configScreen === "list"
									? "Firewall Configs"
									: configScreen === "details"
										? policyEditor.name || "Firewall Config Details"
										: selectedRule?.description || selectedRule?.id || "Rule Details"}
						</h2>
						<p className="section-copy">
							{activeView === "agents"
								? "Review connected agents, heartbeat state, and create one-time enrollment tokens for new installs."
								: configScreen === "list"
									? "Browse saved firewall configs and choose the one you want to inspect."
									: configScreen === "details"
										? "Review one firewall config at a time and switch between overview and rule listings."
										: "Edit one rule at a time with focused actions for saving or deleting changes."}
						</p>
					</div>
					<div className="topbar-actions">
						<button type="button" className="ghost-button" onClick={() => window.location.reload()}>
							Refresh
						</button>
						{activeView === "agents" ? (
							<button
								type="button"
								className="primary-button"
								onClick={() => void handleCreateEnrollmentToken()}
								disabled={isCreatingToken}
							>
								{isCreatingToken ? "Creating..." : "Add Agent Enrollment"}
							</button>
						) : (
							<button type="button" className="primary-button" onClick={startNewConfig}>
								Add Firewall Config
							</button>
						)}
						<button type="button" className="ghost-button" onClick={handleLogout}>
							Logout
						</button>
					</div>
				</header>

				{error ? <div className="notice error">Unable to load dashboard: {error}</div> : null}
				{configError ? <div className="notice error">Firewall policy error: {configError}</div> : null}
				{tokenError ? <div className="notice error">Unable to create enrollment token: {tokenError}</div> : null}

				{activeView === "agents" ? (
					<section className="page-grid">
						<div className="content-stack">
							<section className="surface feature-surface">
								<div className="surface-header">
									<div>
										<p className="section-kicker">Fleet overview</p>
										<h3>Connected agents</h3>
									</div>
									<div className="stat-pills">
										<div className="stat-pill">
											<span>Total</span>
											<strong>{agents.length}</strong>
										</div>
										<div className="stat-pill">
											<span>Online</span>
											<strong>{onlineAgents}</strong>
										</div>
									</div>
								</div>

								{agents.length === 0 && !isLoading ? (
									<div className="empty-state">
										No agents have enrolled yet. Generate an enrollment token and install the agent on a server to populate this list.
									</div>
								) : null}

								<div className="agent-list">
									{agents.map((agent) => (
										<article className="agent-row" key={agent.id}>
											<div className="agent-row-main">
												<div className="identity-block">
													<h4>{agent.name}</h4>
													<p>{agent.hostname || "hostname pending"}</p>
												</div>
												<span className={`status-chip ${agent.online ? "online" : "offline"}`}>
													{agent.online ? "Online" : "Offline"}
												</span>
											</div>
											<div className="agent-row-meta">
												<div>
													<span>Agent version</span>
													<strong>{agent.agentVersion || "unknown"}</strong>
												</div>
												<div>
													<span>Firewall version</span>
													<strong>{agent.firewallVersion || "not applied"}</strong>
												</div>
												<div>
													<span>Last heartbeat</span>
													<strong>{formatTime(agent.lastSeenAt)}</strong>
												</div>
												<div>
													<span>Enrolled</span>
													<strong>{formatTime(agent.enrolledAt)}</strong>
												</div>
											</div>
										</article>
									))}
								</div>
							</section>
						</div>

						<aside className="rail-stack">
							<section className="surface rail-card">
								<p className="section-kicker">Enrollment</p>
								<h3>New agent token</h3>
								<p className="muted-copy">Issue a one-time enrollment token from the dashboard, then use it in the installer or agent CLI.</p>
								{generatedToken ? (
									<div className="token-box">
										<code>{generatedToken.token}</code>
										<span>Expires {formatTime(generatedToken.expiresAt)}</span>
									</div>
								) : (
									<div className="empty-state compact">No token generated yet.</div>
								)}
							</section>

							<section className="surface rail-card">
								<p className="section-kicker">Install</p>
								<h3>One-line agent setup</h3>
								<pre className="code-preview">
{`curl -fsSL https://raw.githubusercontent.com/cloudfirewall/cloudfirewall/main/scripts/install-agent.sh | sudo sh -s -- \\
  --api-url http://YOUR-API:8080 \\
  --enrollment-token <generated-enrollment-token> \\
  --name edge-01`}
								</pre>
							</section>
						</aside>
					</section>
				) : (
					<>
						{configScreen === "list" ? (
							<section className="surface page-surface">
								<div className="surface-header">
									<div>
										<p className="section-kicker">Library</p>
										<h3>All firewall configs</h3>
										<p className="muted-copy">Choose a config to inspect it, update overview settings, or drill into its rules.</p>
									</div>
								</div>

								<div className="config-list">
									{pagedConfigs.map((config) => (
										<button
											key={config.id}
											type="button"
											className="config-list-item"
											onClick={() => openConfigDetails(config)}
										>
											<div className="config-card-copy">
												<strong>{config.name}</strong>
												<span>{summarizePolicyRules(config.policy)}</span>
												<small>Updated {formatTime(config.updatedAt)}</small>
											</div>
											<div className="config-list-meta">
												<span className={`status-chip ${config.isActive ? "online" : "offline"}`}>
													{config.isActive ? "Active" : "Saved"}
												</span>
												<span className="config-version-badge">{config.version}</span>
											</div>
										</button>
									))}
								</div>

								<div className="pagination-bar">
									<span>
										Page {configListPage} of {configPageCount}
									</span>
									<div className="pagination-actions">
										<button type="button" className="ghost-button" disabled={configListPage <= 1} onClick={() => setConfigListPage((page) => page - 1)}>
											Previous
										</button>
										<button
											type="button"
											className="ghost-button"
											disabled={configListPage >= configPageCount}
											onClick={() => setConfigListPage((page) => page + 1)}
										>
											Next
										</button>
									</div>
								</div>
							</section>
						) : configScreen === "details" ? (
							<section className="surface page-surface">
								<nav className="breadcrumbs" aria-label="Breadcrumb">
									<button type="button" className="breadcrumb-link" onClick={openConfigList}>
										Firewall Configs
									</button>
									<span>/</span>
									<span>{policyEditor.name || "New config"}</span>
								</nav>

								<div className="surface-header">
									<div>
										<p className="section-kicker">Details</p>
										<h3>{selectedConfigID ? "Firewall config details" : "Create firewall config"}</h3>
									</div>
									<div className="detail-header-actions">
										<button type="submit" form="firewall-config-form" className="primary-button" disabled={!isPolicyEditable || isSavingConfig}>
											{isSavingConfig ? "Saving..." : selectedConfigID ? "Save config" : "Create config"}
										</button>
										<button
											type="button"
											className="ghost-button"
											disabled={!selectedConfigID || isApplyingConfig}
											onClick={() => void handleApplyConfig()}
										>
											{isApplyingConfig ? "Applying..." : "Apply to fleet"}
										</button>
									</div>
								</div>

								{selectedConfig && !selectedConfig.policy ? (
									<div className="notice neutral">
										This config was loaded from a legacy raw firewall definition. Create a new policy-based config to manage rules from the UI.
									</div>
								) : null}

								<form className="config-form" id="firewall-config-form" onSubmit={handleSavePolicy}>
									<div className="detail-tabs" role="tablist" aria-label="Firewall config details">
										<button
											type="button"
											role="tab"
											aria-selected={configDetailTab === "overview"}
											className={`detail-tab ${configDetailTab === "overview" ? "active" : ""}`}
											onClick={() => setConfigDetailTab("overview")}
										>
											Overview
										</button>
										<button
											type="button"
											role="tab"
											aria-selected={configDetailTab === "rules"}
											className={`detail-tab ${configDetailTab === "rules" ? "active" : ""}`}
											onClick={() => setConfigDetailTab("rules")}
										>
											Rules
										</button>
									</div>

									{configDetailTab === "overview" ? (
										<section className="tab-panel" role="tabpanel" aria-label="Overview">
											<div className="tab-intro">
												<h4>Overview</h4>
												<p>Set the config identity, defaults, and baseline safety behavior.</p>
											</div>

											<label className="field">
												<span>Policy name</span>
												<input value={policyEditor.name} onChange={(event) => setPolicyEditor((current) => ({ ...current, name: event.target.value }))} required />
											</label>
											<label className="field">
												<span>Description</span>
												<textarea rows={3} value={policyEditor.description ?? ""} onChange={(event) => setPolicyEditor((current) => ({ ...current, description: event.target.value }))} />
											</label>

											<div className="rule-grid">
												<label className="field">
													<span>Default inbound</span>
													<select
														value={policyEditor.defaultInboundAction}
														onChange={(event) =>
															setPolicyEditor((current) => ({
																...current,
																defaultInboundAction: event.target.value as PolicyDraft["defaultInboundAction"],
															}))
														}
													>
														<option value="DENY">Deny</option>
														<option value="ALLOW">Allow</option>
														<option value="REJECT">Reject</option>
													</select>
												</label>
												<label className="field">
													<span>Default outbound</span>
													<select
														value={policyEditor.defaultOutboundAction}
														onChange={(event) =>
															setPolicyEditor((current) => ({
																...current,
																defaultOutboundAction: event.target.value as PolicyDraft["defaultOutboundAction"],
															}))
														}
													>
														<option value="ALLOW">Allow</option>
														<option value="DENY">Deny</option>
														<option value="REJECT">Reject</option>
													</select>
												</label>
											</div>

											<div className="check-row">
												<label>
													<input type="checkbox" checked={policyEditor.allowLoopback} onChange={(event) => setPolicyEditor((current) => ({ ...current, allowLoopback: event.target.checked }))} /> Allow loopback
												</label>
												<label>
													<input
														type="checkbox"
														checked={policyEditor.allowEstablishedRelated}
														onChange={(event) => setPolicyEditor((current) => ({ ...current, allowEstablishedRelated: event.target.checked }))}
													/>{" "}
													Allow established/related
												</label>
											</div>
										</section>
									) : (
										<section className="tab-panel" role="tabpanel" aria-label="Rules">
											<div className="rules-toolbar">
												<div>
													<h4>Rules</h4>
													<p>Review all rules in short form, then open one to edit it in detail.</p>
												</div>
												<div className="rules-toolbar-actions">
													<button type="button" className="ghost-button" onClick={addRule}>
														Add rule
													</button>
												</div>
											</div>

											<div className="rules-list-page">
												{policyEditor.rules.length === 0 ? (
													<div className="empty-state">No rules yet. Add a rule to start defining traffic behavior.</div>
												) : null}
												{policyEditor.rules.map((rule, index) => (
													<button key={`${rule.id}-${index}`} type="button" className="rule-list-item" onClick={() => openRuleDetails(index)}>
														<strong>{rule.description || `Rule ${index + 1}`}</strong>
														<span>{rule.action} {rule.direction.toLowerCase()}</span>
														<small>{describeRule(rule)}</small>
													</button>
												))}
											</div>
										</section>
									)}

									<div className="form-actions">
										<button type="submit" className="primary-button" disabled={!isPolicyEditable || isSavingConfig}>
											{isSavingConfig ? "Saving..." : selectedConfigID ? "Save config" : "Create config"}
										</button>
										<button
											type="button"
											className="ghost-button"
											disabled={!selectedConfigID || isApplyingConfig}
											onClick={() => void handleApplyConfig()}
										>
											{isApplyingConfig ? "Applying..." : "Apply to fleet"}
										</button>
										<button
											type="button"
											className="ghost-button danger-ghost"
											disabled={!selectedConfigID || isDeletingConfig}
											onClick={() => void handleDeleteConfig()}
										>
											{isDeletingConfig ? "Deleting..." : "Delete"}
										</button>
									</div>
								</form>
							</section>
						) : (
							<section className="surface page-surface">
								<nav className="breadcrumbs" aria-label="Breadcrumb">
									<button type="button" className="breadcrumb-link" onClick={openConfigList}>
										Firewall Configs
									</button>
									<span>/</span>
									<button
										type="button"
										className="breadcrumb-link"
										onClick={() => {
											setConfigScreen("details");
											setConfigDetailTab("rules");
										}}
									>
										{policyEditor.name || "New config"}
									</button>
									<span>/</span>
									<span>Rules</span>
									<span>/</span>
									<span>{selectedRule?.description || selectedRule?.id || "Rule details"}</span>
								</nav>

								<div className="surface-header">
									<div>
										<p className="section-kicker">Rule Details</p>
										<h3>{selectedRule?.description || selectedRule?.id || "Rule details"}</h3>
									</div>
									<div className="detail-header-actions">
										<button type="submit" form="firewall-config-form" className="primary-button" disabled={!isPolicyEditable || isSavingConfig}>
											{isSavingConfig ? "Saving..." : "Save rule"}
										</button>
										<button
											type="button"
											className="ghost-button danger-ghost"
											disabled={selectedRuleIndex < 0}
											onClick={() => removeRule(selectedRuleIndex)}
										>
											Delete rule
										</button>
									</div>
								</div>

								<form className="config-form" id="firewall-config-form" onSubmit={handleSavePolicy}>
									{selectedRule ? (
										<div className="rule-editor-pane standalone">
											<div className="rule-grid">
												<label className="field">
													<span>Direction</span>
													<select value={selectedRule.direction} onChange={(event) => updateRule(selectedRuleIndex, { direction: event.target.value as PolicyRuleDraft["direction"] })}>
														<option value="INBOUND">Inbound</option>
														<option value="OUTBOUND">Outbound</option>
													</select>
												</label>
												<label className="field">
													<span>Action</span>
													<select value={selectedRule.action} onChange={(event) => updateRule(selectedRuleIndex, { action: event.target.value as PolicyRuleDraft["action"] })}>
														<option value="ALLOW">Allow</option>
														<option value="DENY">Deny</option>
														<option value="REJECT">Reject</option>
													</select>
												</label>
												<label className="field">
													<span>Peer</span>
													<select value={selectedRule.peerType} onChange={(event) => updateRule(selectedRuleIndex, { peerType: event.target.value as PolicyRuleDraft["peerType"] })}>
														<option value="PUBLIC_INTERNET">Public internet</option>
														<option value="OFFICE_IPS">Office IPs</option>
														<option value="CIDR">Custom CIDR</option>
														<option value="THIS_NODE">This node</option>
													</select>
												</label>
												<label className="field">
													<span>Protocol</span>
													<select value={selectedRule.protocol} onChange={(event) => updateRule(selectedRuleIndex, { protocol: event.target.value as PolicyRuleDraft["protocol"] })}>
														<option value="TCP">TCP</option>
														<option value="UDP">UDP</option>
													</select>
												</label>
											</div>

											{selectedRule.peerType === "CIDR" ? (
												<label className="field">
													<span>Peer CIDR</span>
													<input value={selectedRule.peerValue ?? ""} onChange={(event) => updateRule(selectedRuleIndex, { peerValue: event.target.value })} placeholder="203.0.113.0/24" />
												</label>
											) : null}

											<div className="rule-grid">
												<label className="field">
													<span>Ports</span>
													<input value={selectedRule.ports.join(",")} onChange={(event) => updateRule(selectedRuleIndex, { ports: parsePorts(event.target.value) })} placeholder="443,8443" />
												</label>
												<label className="field">
													<span>Rule ID</span>
													<input value={selectedRule.id} onChange={(event) => updateRule(selectedRuleIndex, { id: event.target.value })} />
												</label>
												<label className="field">
													<span>Order</span>
													<input type="number" value={selectedRule.orderIndex} onChange={(event) => updateRule(selectedRuleIndex, { orderIndex: Number(event.target.value) || 0 })} />
												</label>
											</div>

											<label className="field">
												<span>Description</span>
												<input value={selectedRule.description ?? ""} onChange={(event) => updateRule(selectedRuleIndex, { description: event.target.value })} />
											</label>

											<div className="check-row">
												<label>
													<input type="checkbox" checked={selectedRule.enabled} onChange={(event) => updateRule(selectedRuleIndex, { enabled: event.target.checked })} /> Enabled
												</label>
												<label>
													<input type="checkbox" checked={selectedRule.logEnabled} onChange={(event) => updateRule(selectedRuleIndex, { logEnabled: event.target.checked })} /> Log matched traffic
												</label>
											</div>
										</div>
									) : (
										<div className="empty-state">Select a rule from the rules tab before opening the rule details page.</div>
									)}
								</form>
							</section>
						)}
					</>
				)}
			</main>
		</div>
	);
}

function summarizePolicyRules(policy?: PolicyDraft) {
	if (!policy) return "Legacy config";
	const ruleCount = policy.rules.length;
	if (ruleCount === 0) return "No rules yet";
	if (ruleCount === 1) return "1 rule";
	return `${ruleCount} rules`;
}

function describeRule(rule: PolicyRuleDraft) {
	const ports = rule.ports.length > 0 ? rule.ports.join(", ") : "any port";
	const peer = rule.peerType === "CIDR" ? rule.peerValue || "custom CIDR" : rule.peerType.split("_").join(" ").toLowerCase();
	return `${rule.protocol} ${ports} from ${peer}`;
}

function parsePorts(value: string) {
	return value
		.split(",")
		.map((entry) => Number(entry.trim()))
		.filter((port) => Number.isInteger(port) && port > 0);
}

async function extractError(response: Response, fallback: string) {
	try {
		const payload = (await response.json()) as { error?: string };
		return payload.error || fallback;
	} catch {
		return fallback;
	}
}

function formatTime(value?: string) {
	if (!value) return "waiting for first heartbeat";
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return value;
	return date.toLocaleString();
}
