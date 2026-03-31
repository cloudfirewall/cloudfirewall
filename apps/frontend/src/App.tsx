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
type ConfigDetailTab = "overview" | "rules" | "nftables";
type ConfigScreen = "list" | "details" | "rule";
type ParsedRoute =
	| { view: "agents" }
	| { view: "configs"; screen: "list" }
	| { view: "configs"; screen: "details"; isNew: boolean; configId?: string; tab: ConfigDetailTab }
	| { view: "configs"; screen: "rule"; configId: string; ruleID: string };

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
	const [pathname, setPathname] = useState(() => normalizePathname(window.location.pathname));
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
	const [copiedItem, setCopiedItem] = useState("");
	const [isLoading, setIsLoading] = useState(true);
	const [isAuthenticating, setIsAuthenticating] = useState(false);
	const [isCreatingToken, setIsCreatingToken] = useState(false);
	const [isSavingConfig, setIsSavingConfig] = useState(false);
	const [isApplyingConfig, setIsApplyingConfig] = useState(false);
	const [isDeletingConfig, setIsDeletingConfig] = useState(false);

	useEffect(() => {
		function handlePopState() {
			setPathname(normalizePathname(window.location.pathname));
		}

		window.addEventListener("popstate", handlePopState);
		return () => window.removeEventListener("popstate", handlePopState);
	}, []);

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

	useEffect(() => {
		const route = parseRoute(pathname);

		if (route.view === "agents") {
			setActiveView("agents");
			return;
		}

		setActiveView("configs");
		setConfigScreen(route.screen);

		if (route.screen === "list") {
			return;
		}

		if (route.screen === "details" && route.isNew) {
			if (selectedConfigID !== "") {
				setSelectedConfigID("");
				setSelectedRuleIndex(-1);
				setEditorVersion("");
				setPolicyEditor(emptyPolicy());
				setConfigError("");
			}
			setConfigDetailTab(route.tab);
			return;
		}

		if (!route.configId || configs.length === 0) {
			return;
		}

		const config = configs.find((item) => item.id === route.configId);
		if (!config) {
			return;
		}

		if (selectedConfigID !== config.id) {
			syncConfigEditor(config);
		}

		if (route.screen === "details") {
			setConfigDetailTab(route.tab);
			return;
		}

		if (route.screen === "rule") {
			setConfigDetailTab("rules");
			const ruleIndex = config.policy?.rules.findIndex((rule) => rule.id === route.ruleID) ?? -1;
			if (ruleIndex >= 0) {
				setSelectedRuleIndex(ruleIndex);
			}
		}
	}, [pathname, configs, selectedConfigID]);

	const onlineAgents = agents.filter((agent) => agent.online).length;
	const selectedConfig = configs.find((config) => config.id === selectedConfigID) ?? null;
	const selectedRule = selectedRuleIndex >= 0 ? policyEditor.rules[selectedRuleIndex] ?? null : null;
	const isPolicyEditable = !selectedConfig || Boolean(selectedConfig.policy);
	const configPageCount = Math.max(1, Math.ceil(configs.length / CONFIGS_PAGE_SIZE));
	const pagedConfigs = configs.slice((configListPage - 1) * CONFIGS_PAGE_SIZE, configListPage * CONFIGS_PAGE_SIZE);
	const agentSetupScript = buildAgentSetupScript(generatedToken?.token);
	const inboundRules = policyEditor.rules.filter((rule) => rule.direction === "INBOUND");
	const outboundRules = policyEditor.rules.filter((rule) => rule.direction === "OUTBOUND");

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

	function navigateTo(path: string, replace = false) {
		const nextPath = normalizePathname(path);
		if (nextPath === pathname) {
			return;
		}
		const method = replace ? "replaceState" : "pushState";
		window.history[method](null, "", nextPath);
		setPathname(nextPath);
	}

	function openConfigList() {
		navigateTo("/firewall-configs");
	}

	function openConfigDetails(config: FirewallConfigSummary) {
		selectConfig(config);
		navigateTo(configPath(config.id, "overview"));
	}

	function startNewConfig() {
		navigateTo("/firewall-configs/new");
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

	async function copyToClipboard(text: string, key: string) {
		try {
			await navigator.clipboard.writeText(text);
			setCopiedItem(key);
			window.setTimeout(() => {
				setCopiedItem((current) => (current === key ? "" : current));
			}, 2000);
		} catch {
			setTokenError("Copy failed. Please copy it manually.");
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
			if (next) {
				selectConfig(next);
				if (configScreen === "rule" && selectedRuleIndex >= 0) {
					const nextRule = next.policy?.rules[selectedRuleIndex];
					if (nextRule) {
						navigateTo(rulePath(next.id, nextRule.id), true);
						return;
					}
				}
				navigateTo(selectedConfigID ? configPath(next.id, configDetailTab) : configPath(next.id, "overview"), true);
			}
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
			if (!response.ok) throw new Error(await extractError(response, "Failed to apply firewall policy"));
			const nextConfigs = await reloadConfigs();
			const next = nextConfigs.find((config) => config.id === selectedConfigID);
			if (next) {
				selectConfig(next);
				navigateTo(configPath(next.id, configDetailTab), true);
			}
		} catch (applyError) {
			setConfigError(applyError instanceof Error ? applyError.message : "Failed to apply firewall policy");
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
			if (!response.ok) throw new Error(await extractError(response, "Failed to delete firewall policy"));
			const nextConfigs = await reloadConfigs();
			if (nextConfigs.length > 0) {
				selectConfig(nextConfigs[0]);
				navigateTo("/firewall-configs", true);
			} else {
				navigateTo("/firewall-configs", true);
			}
		} catch (deleteError) {
			setConfigError(deleteError instanceof Error ? deleteError.message : "Failed to delete firewall policy");
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
		const nextRuleID = `rule-${nextIndex + 1}`;
		setPolicyEditor((current) => ({
			...current,
			rules: current.rules.concat({
				id: nextRuleID,
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
		setSelectedRuleIndex(nextIndex);
		if (selectedConfigID) {
			navigateTo(rulePath(selectedConfigID, nextRuleID));
		}
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
		if (selectedConfigID) {
			navigateTo(configPath(selectedConfigID, "rules"), true);
		} else {
			navigateTo("/firewall-configs/new/rules", true);
		}
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
							{isAuthenticating ? "Signing in..." : "Sign in"}
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
					<p>Manage firewall policies and monitor enrolled agents across your fleet.</p>
				</div>

				<nav className="sidebar-nav" aria-label="Primary">
					<button
						type="button"
						className={`nav-item ${activeView === "agents" ? "active" : ""}`}
						onClick={() => navigateTo("/agents")}
					>
						<span className="nav-label">Agents</span>
						<span className="nav-meta">{agents.length}</span>
					</button>
					<button
						type="button"
						className={`nav-item ${activeView === "configs" ? "active" : ""}`}
						onClick={openConfigList}
					>
						<span className="nav-label">Firewall Policies</span>
						<span className="nav-meta">{configs.length}</span>
					</button>
				</nav>

				<div className="sidebar-footer">
					<div className="sidebar-stat">
						<span>Online agents</span>
						<strong>{onlineAgents}</strong>
					</div>
					<div className="sidebar-stat">
						<span>Updated</span>
						<strong>{lastUpdated || "—"}</strong>
					</div>
				</div>
			</aside>

			<main className="main-stage">
				<header className="topbar">
					<div>
						<p className="section-kicker">{activeView === "agents" ? "Fleet" : "Policies"}</p>
						<h2>
							{activeView === "agents"
								? "Agents"
								: configScreen === "list"
									? "Firewall Policies"
									: configScreen === "details"
										? policyEditor.name || "Firewall Policy"
										: ruleDetailTitle(selectedRule)}
						</h2>
						<p className="section-copy">
							{activeView === "agents"
								? "Monitor connected agents and issue enrollment tokens for new installs."
								: configScreen === "list"
									? "Select a policy to review its rules, check its status, or apply it to the fleet."
									: configScreen === "details"
										? "Manage this policy's settings, rules, and generated nftables output."
										: "Configure this rule's traffic direction, source, protocol, and ports."}
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
								{isCreatingToken ? "Creating..." : "New Token"}
							</button>
						) : (
							<button type="button" className="primary-button" onClick={startNewConfig}>
								New Policy
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
										No agents enrolled. Generate a token and run the installer on a server to get started.
									</div>
								) : null}

								<div className="agent-list">
									{agents.map((agent) => (
										<article className="agent-row" key={agent.id}>
											<div className="agent-row-main">
												<div className="identity-block">
													<h4>{agent.name}</h4>
													<p>{agent.hostname || "no hostname"}</p>
												</div>
												<span className={`status-chip ${agent.online ? "online" : "offline"}`}>
													{agent.online ? "Online" : "Offline"}
												</span>
											</div>
											<div className="agent-row-meta">
												<div>
													<span>Agent version</span>
													<strong>{agent.agentVersion || "—"}</strong>
												</div>
												<div>
													<span>Firewall version</span>
													<strong>{agent.firewallVersion || "none"}</strong>
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
								<h3>New Enrollment Token</h3>
								<p className="muted-copy">Generate a one-time token to enroll a new agent. Tokens expire after 10 minutes.</p>
								{generatedToken ? (
									<div className="token-box">
										<div className="copy-row">
											<strong>Enrollment token</strong>
											<button type="button" className="ghost-button copy-button" onClick={() => void copyToClipboard(generatedToken.token, "token")}>
												{copiedItem === "token" ? "Copied" : "Copy token"}
											</button>
										</div>
										<code>{generatedToken.token}</code>
										<span>Expires {formatTime(generatedToken.expiresAt)}</span>
									</div>
								) : (
									<div className="empty-state compact">No token yet. Click "New Token" to generate one.</div>
								)}
							</section>

							<section className="surface rail-card">
								<p className="section-kicker">Install</p>
								<h3>Quick Install</h3>
								<div className="copy-row">
									<span className="muted-copy">Run this on the target server to install and enroll the agent.</span>
									<button type="button" className="ghost-button copy-button" onClick={() => void copyToClipboard(agentSetupScript, "setup-script")}>
										{copiedItem === "setup-script" ? "Copied" : "Copy"}
									</button>
								</div>
								<pre className="code-preview">{agentSetupScript}</pre>
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
										<h3>All firewall policies</h3>
										<p className="muted-copy">Select a policy to review its rules, check its status, or apply it to the fleet.</p>
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
													{config.isActive ? "Applied" : "Saved"}
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
										Firewall Policies
									</button>
									<span>/</span>
									<span>{policyEditor.name || "New Policy"}</span>
								</nav>

								<div className="firewall-page-header">
									<div className="firewall-page-titlebar">
										<div className="firewall-page-title-group">
											<h3>{policyEditor.name || (selectedConfigID ? "Firewall Policy" : "New Policy")}</h3>
											{selectedConfig ? (
												<span className={`fw-status-badge ${selectedConfig.isActive ? "applied" : "saved"}`}>
													{selectedConfig.isActive ? "✓ Fully applied" : "Saved"}
												</span>
											) : null}
										</div>
										<div className="fw-header-actions">
											{configDetailTab === "rules" ? (
												<button type="button" className="primary-button fw-add-rule-btn" onClick={addRule}>
													Add rule ↓
												</button>
											) : null}
											<button type="submit" form="firewall-config-form" className="ghost-button" disabled={!isPolicyEditable || isSavingConfig}>
												{isSavingConfig ? "Saving..." : selectedConfigID ? "Save" : "Create"}
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
									</div>
									<div className="firewall-page-stats">
										<span>Rules {policyEditor.rules.length}</span>
										{selectedConfig ? <span>{selectedConfig.isActive ? "Active" : "Draft"}</span> : null}
										{editorVersion ? <span>v{editorVersion}</span> : null}
									</div>
									<div className="detail-tabs detail-tabs-line" role="tablist" aria-label="Firewall config details">
										<button
											type="button"
											role="tab"
											aria-selected={configDetailTab === "overview"}
											className={`detail-tab detail-tab-line ${configDetailTab === "overview" ? "active" : ""}`}
											onClick={() => navigateTo(selectedConfigID ? configPath(selectedConfigID, "overview") : "/firewall-configs/new")}
										>
											Overview
										</button>
										<button
											type="button"
											role="tab"
											aria-selected={configDetailTab === "rules"}
											className={`detail-tab detail-tab-line ${configDetailTab === "rules" ? "active" : ""}`}
											onClick={() => navigateTo(selectedConfigID ? configPath(selectedConfigID, "rules") : "/firewall-configs/new/rules")}
										>
											Rules
										</button>
										<button
											type="button"
											role="tab"
											aria-selected={configDetailTab === "nftables"}
											className={`detail-tab detail-tab-line ${configDetailTab === "nftables" ? "active" : ""}`}
											onClick={() => navigateTo(selectedConfigID ? configPath(selectedConfigID, "nftables") : "/firewall-configs/new/nftables")}
										>
											NFTables
										</button>
									</div>
								</div>

								{selectedConfig && !selectedConfig.policy ? (
									<div className="notice neutral">
										This config uses a legacy raw format. Create a new policy to manage rules from the UI.
									</div>
								) : null}

								<form className="config-form" id="firewall-config-form" onSubmit={handleSavePolicy}>
									{configDetailTab === "overview" ? (
										<section className="tab-panel" role="tabpanel" aria-label="Overview">
											<div className="tab-intro">
												<h4>Overview</h4>
												<p>Name this policy and configure its default traffic handling behavior.</p>
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
									) : configDetailTab === "rules" ? (
										<section className="tab-panel" role="tabpanel" aria-label="Rules">
											<div className="inline-rules-section">
												{policyEditor.rules.length === 0 ? (
													<div className="empty-state">No rules yet. Click &quot;Add rule&quot; to start defining traffic behavior.</div>
												) : (
													<>
														{inboundRules.length > 0 ? (
															<div className="inline-direction-section">
																<div className="inline-direction-label">
																	<span>INBOUND</span>
																	<span className="direction-arrow">←</span>
																</div>
																{inboundRules.map((rule) => {
																	const index = policyEditor.rules.findIndex((item) => item.id === rule.id);
																	return (
																		<div key={rule.id} className="inline-rule-card">
																			<button type="button" className="rule-remove-btn" onClick={() => removeRule(index)} title="Remove rule">×</button>
																			<input
																				className="inline-rule-desc"
																				value={rule.description ?? ""}
																				onChange={(e) => updateRule(index, { description: e.target.value })}
																				placeholder="Add description"
																			/>
																			<div className="inline-rule-fields">
																				<div className="peer-chips-wrap">
																					{rule.peerType === "PUBLIC_INTERNET" ? (
																						<>
																							<span className="peer-chip">Any IPv4</span>
																							<span className="peer-chip">Any IPv6</span>
																						</>
																					) : rule.peerType === "OFFICE_IPS" ? (
																						<span className="peer-chip">Office IPs</span>
																					) : rule.peerType === "THIS_NODE" ? (
																						<span className="peer-chip">This node</span>
																					) : rule.peerType === "CIDR" && rule.peerValue ? (
																						<span className="peer-chip">{rule.peerValue}</span>
																					) : null}
																					<select
																						className="peer-type-select"
																						value={rule.peerType}
																						onChange={(e) => updateRule(index, { peerType: e.target.value as PolicyRuleDraft["peerType"] })}
																					>
																						<option value="PUBLIC_INTERNET">Any IPv4+IPv6</option>
																						<option value="OFFICE_IPS">Office IPs</option>
																						<option value="CIDR">Custom CIDR…</option>
																						<option value="THIS_NODE">This node</option>
																					</select>
																				</div>
																				{rule.peerType === "CIDR" ? (
																					<input
																						className="peer-cidr-input"
																						value={rule.peerValue ?? ""}
																						onChange={(e) => updateRule(index, { peerValue: e.target.value })}
																						placeholder="0.0.0.0/0"
																					/>
																				) : null}
																				<div className="inline-labeled-field">
																					<span>Protocol *</span>
																					<select
																						className="inline-select"
																						value={rule.protocol}
																						onChange={(e) => updateRule(index, { protocol: e.target.value as PolicyRuleDraft["protocol"] })}
																					>
																						<option value="TCP">TCP</option>
																						<option value="UDP">UDP</option>
																					</select>
																				</div>
																				<div className="inline-labeled-field">
																					<span>Port *</span>
																					<input
																						className="inline-port-input"
																						type="number"
																						min={1}
																						max={65535}
																						value={rule.ports[0] ?? ""}
																						onChange={(e) => {
																							const val = Number(e.target.value);
																							updateRule(index, { ports: val > 0 ? [val, ...rule.ports.slice(1)] : rule.ports.slice(1) });
																						}}
																						placeholder="Any"
																					/>
																				</div>
																				<span className="port-sep">—</span>
																				<div className="inline-labeled-field">
																					<span>Port range</span>
																					<input
																						className="inline-port-input"
																						type="number"
																						min={1}
																						max={65535}
																						value={rule.ports.length > 1 ? rule.ports[rule.ports.length - 1] : ""}
																						onChange={(e) => {
																							const val = Number(e.target.value);
																							const base = rule.ports[0] ?? 0;
																							updateRule(index, { ports: val > 0 ? [base, val] : base > 0 ? [base] : [] });
																						}}
																						placeholder="Port range"
																					/>
																				</div>
																			</div>
																		</div>
																	);
																})}
															</div>
														) : null}
														{outboundRules.length > 0 ? (
															<div className="inline-direction-section">
																<div className="inline-direction-label">
																	<span>OUTBOUND</span>
																	<span className="direction-arrow">→</span>
																</div>
																{outboundRules.map((rule) => {
																	const index = policyEditor.rules.findIndex((item) => item.id === rule.id);
																	return (
																		<div key={rule.id} className="inline-rule-card">
																			<button type="button" className="rule-remove-btn" onClick={() => removeRule(index)} title="Remove rule">×</button>
																			<input
																				className="inline-rule-desc"
																				value={rule.description ?? ""}
																				onChange={(e) => updateRule(index, { description: e.target.value })}
																				placeholder="Add description"
																			/>
																			<div className="inline-rule-fields">
																				<div className="peer-chips-wrap">
																					{rule.peerType === "PUBLIC_INTERNET" ? (
																						<>
																							<span className="peer-chip">Any IPv4</span>
																							<span className="peer-chip">Any IPv6</span>
																						</>
																					) : rule.peerType === "OFFICE_IPS" ? (
																						<span className="peer-chip">Office IPs</span>
																					) : rule.peerType === "THIS_NODE" ? (
																						<span className="peer-chip">This node</span>
																					) : rule.peerType === "CIDR" && rule.peerValue ? (
																						<span className="peer-chip">{rule.peerValue}</span>
																					) : null}
																					<select
																						className="peer-type-select"
																						value={rule.peerType}
																						onChange={(e) => updateRule(index, { peerType: e.target.value as PolicyRuleDraft["peerType"] })}
																					>
																						<option value="PUBLIC_INTERNET">Any IPv4+IPv6</option>
																						<option value="OFFICE_IPS">Office IPs</option>
																						<option value="CIDR">Custom CIDR…</option>
																						<option value="THIS_NODE">This node</option>
																					</select>
																				</div>
																				{rule.peerType === "CIDR" ? (
																					<input
																						className="peer-cidr-input"
																						value={rule.peerValue ?? ""}
																						onChange={(e) => updateRule(index, { peerValue: e.target.value })}
																						placeholder="0.0.0.0/0"
																					/>
																				) : null}
																				<div className="inline-labeled-field">
																					<span>Protocol *</span>
																					<select
																						className="inline-select"
																						value={rule.protocol}
																						onChange={(e) => updateRule(index, { protocol: e.target.value as PolicyRuleDraft["protocol"] })}
																					>
																						<option value="TCP">TCP</option>
																						<option value="UDP">UDP</option>
																					</select>
																				</div>
																				<div className="inline-labeled-field">
																					<span>Port *</span>
																					<input
																						className="inline-port-input"
																						type="number"
																						min={1}
																						max={65535}
																						value={rule.ports[0] ?? ""}
																						onChange={(e) => {
																							const val = Number(e.target.value);
																							updateRule(index, { ports: val > 0 ? [val, ...rule.ports.slice(1)] : rule.ports.slice(1) });
																						}}
																						placeholder="Any"
																					/>
																				</div>
																				<span className="port-sep">—</span>
																				<div className="inline-labeled-field">
																					<span>Port range</span>
																					<input
																						className="inline-port-input"
																						type="number"
																						min={1}
																						max={65535}
																						value={rule.ports.length > 1 ? rule.ports[rule.ports.length - 1] : ""}
																						onChange={(e) => {
																							const val = Number(e.target.value);
																							const base = rule.ports[0] ?? 0;
																							updateRule(index, { ports: val > 0 ? [base, val] : base > 0 ? [base] : [] });
																						}}
																						placeholder="Port range"
																					/>
																				</div>
																			</div>
																		</div>
																	);
																})}
															</div>
														) : null}
													</>
												)}
											</div>
										</section>
									) : (
										<section className="tab-panel" role="tabpanel" aria-label="NFTables">
											<div className="tab-intro">
												<h4>nftables Output</h4>
												<p>The nftables configuration generated from this policy's rules.</p>
											</div>

											{selectedConfig?.nftablesConfig ? (
												<pre className="code-preview">{selectedConfig.nftablesConfig}</pre>
											) : (
												<div className="empty-state">
													Save the policy first to see its generated nftables configuration.
												</div>
											)}
										</section>
									)}

									<div className="form-actions-bar">
										<button type="submit" className="primary-button" disabled={!isPolicyEditable || isSavingConfig}>
											{isSavingConfig ? "Saving..." : selectedConfigID ? "Save policy" : "Create policy"}
										</button>
									</div>
								</form>
							</section>
						) : (
							<section className="surface page-surface">
								<nav className="breadcrumbs" aria-label="Breadcrumb">
									<button type="button" className="breadcrumb-link" onClick={openConfigList}>
										Firewall Policies
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
									<span>{ruleDetailTitle(selectedRule)}</span>
								</nav>

								<div className="policy-hero rule-hero">
									<div className="title-block">
										<p className="section-kicker">Rule</p>
										<h3>{ruleDetailTitle(selectedRule)}</h3>
										{selectedRule ? (
											<div className="policy-hero-meta">
												<span>{selectedRule.direction === "INBOUND" ? "Inbound" : "Outbound"}</span>
												<span>{selectedRule.action}</span>
												<span>{formatPeerLabel(selectedRule)}</span>
												<span>{formatPortLabel(selectedRule)}</span>
											</div>
										) : null}
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
											<div className="tab-intro">
												<h4>Rule settings</h4>
												<p>Set the traffic direction, source, protocol, ports, and action for this rule.</p>
											</div>
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
												<textarea rows={4} value={selectedRule.description ?? ""} onChange={(event) => updateRule(selectedRuleIndex, { description: event.target.value })} />
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
										<div className="empty-state">No rule selected. Open a rule from the Rules tab.</div>
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

function ruleDetailTitle(rule: PolicyRuleDraft | null) {
	return rule?.id || "Rule Details";
}

function buildAgentSetupScript(token?: string) {
	const enrollmentToken = token?.trim() ? token : "<generated-enrollment-token>";
	return `curl -fsSL https://raw.githubusercontent.com/cloudfirewall/cloudfirewall/main/scripts/install-agent.sh | sudo sh -s -- \\
  --api-url http://YOUR-API:8080 \\
  --enrollment-token ${enrollmentToken} \\
  --name edge-01`;
}

function normalizePathname(pathname: string) {
	if (!pathname || pathname === "/") return "/agents";
	return pathname.replace(/\/+$/, "") || "/agents";
}

function configPath(configID: string, tab: ConfigDetailTab) {
	if (tab === "rules") {
		return `/firewall-configs/${encodeURIComponent(configID)}/rules`;
	}
	if (tab === "nftables") {
		return `/firewall-configs/${encodeURIComponent(configID)}/nftables`;
	}
	return `/firewall-configs/${encodeURIComponent(configID)}`;
}

function rulePath(configID: string, ruleID: string) {
	return `/firewall-configs/${encodeURIComponent(configID)}/rules/${encodeURIComponent(ruleID)}`;
}

function parseRoute(pathname: string): ParsedRoute {
	const normalized = normalizePathname(pathname);
	const parts = normalized.split("/").filter(Boolean).map(decodeURIComponent);

	if (parts.length === 0 || parts[0] === "agents") {
		return { view: "agents" };
	}

	if (parts[0] !== "firewall-configs") {
		return { view: "agents" };
	}

	if (parts.length === 1) {
		return { view: "configs", screen: "list" };
	}

	if (parts[1] === "new") {
		if (parts[2] === "rules") {
			return { view: "configs", screen: "details", isNew: true, tab: "rules" };
		}
		if (parts[2] === "nftables") {
			return { view: "configs", screen: "details", isNew: true, tab: "nftables" };
		}
		return { view: "configs", screen: "details", isNew: true, tab: "overview" };
	}

	const configID = parts[1];
	if (parts.length === 2) {
		return { view: "configs", screen: "details", isNew: false, configId: configID, tab: "overview" };
	}

	if (parts[2] === "rules" && parts.length === 3) {
		return { view: "configs", screen: "details", isNew: false, configId: configID, tab: "rules" };
	}

	if (parts[2] === "nftables" && parts.length === 3) {
		return { view: "configs", screen: "details", isNew: false, configId: configID, tab: "nftables" };
	}

	if (parts[2] === "rules" && parts[3]) {
		return { view: "configs", screen: "rule", configId: configID, ruleID: parts[3] };
	}

	return { view: "configs", screen: "list" };
}


function formatPeerLabel(rule: PolicyRuleDraft) {
	if (rule.peerType === "CIDR") {
		return rule.peerValue || "Custom CIDR";
	}
	if (rule.peerType === "PUBLIC_INTERNET") {
		return "Public internet";
	}
	if (rule.peerType === "OFFICE_IPS") {
		return "Office IPs";
	}
	if (rule.peerType === "THIS_NODE") {
		return "This node";
	}
	return rule.peerType;
}

function formatPortLabel(rule: PolicyRuleDraft) {
	if (rule.ports.length === 0) return "Any port";
	if (rule.ports.length === 1) return `Port ${rule.ports[0]}`;
	return `Ports ${rule.ports.join(", ")}`;
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
	if (!value) return "never";
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return value;
	return date.toLocaleString();
}
