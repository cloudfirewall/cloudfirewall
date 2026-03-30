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

type ListAgentsResponse = {
	agents: AgentSummary[];
};

type CreateEnrollmentTokenResponse = {
	token: string;
	tokenId: string;
	expiresAt: string;
};

const API_BASE = import.meta.env.VITE_API_BASE ?? "";
const POLL_INTERVAL_MS = 5000;
const TOKEN_KEY = "cloudfirewall_admin_token";

export default function App() {
	const [authToken, setAuthToken] = useState(() => window.localStorage.getItem(TOKEN_KEY) ?? "");
	const [username, setUsername] = useState("admin");
	const [password, setPassword] = useState("");
	const [agents, setAgents] = useState<AgentSummary[]>([]);
	const [error, setError] = useState("");
	const [isLoading, setIsLoading] = useState(true);
	const [lastUpdated, setLastUpdated] = useState("");
	const [authError, setAuthError] = useState("");
	const [isAuthenticating, setIsAuthenticating] = useState(false);
	const [generatedToken, setGeneratedToken] = useState<CreateEnrollmentTokenResponse | null>(null);
	const [tokenError, setTokenError] = useState("");
	const [isCreatingToken, setIsCreatingToken] = useState(false);

	useEffect(() => {
		if (!authToken) {
			setAgents([]);
			setIsLoading(false);
			return;
		}

		let cancelled = false;
		setIsLoading(true);

		async function loadAgents() {
			try {
				const response = await fetch(`${API_BASE}/api/v1/agents`, {
					headers: {
						Authorization: `Bearer ${authToken}`,
					},
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
				if (cancelled) {
					return;
				}

				startTransition(() => {
					setAgents(payload.agents);
					setError("");
					setLastUpdated(new Date().toLocaleTimeString());
				});
			} catch (loadError) {
				if (cancelled) {
					return;
				}

				startTransition(() => {
					setError(loadError instanceof Error ? loadError.message : "Failed to load agents");
				});
			} finally {
				if (!cancelled) {
					setIsLoading(false);
				}
			}
		}

		void loadAgents();
		const timer = window.setInterval(() => {
			void loadAgents();
		}, POLL_INTERVAL_MS);

		return () => {
			cancelled = true;
			window.clearInterval(timer);
		};
	}, [authToken]);

	const onlineAgents = agents.filter((agent) => agent.online).length;

	async function handleLogin(event: FormEvent<HTMLFormElement>) {
		event.preventDefault();
		setIsAuthenticating(true);
		setAuthError("");

		try {
			const response = await fetch(`${API_BASE}/api/v1/admin/login`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
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
		setLastUpdated("");
		setError("");
	}

	async function handleCreateEnrollmentToken() {
		setIsCreatingToken(true);
		setTokenError("");

		try {
			const response = await fetch(`${API_BASE}/api/v1/enrollment-tokens`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${authToken}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify({ ttlSeconds: 600 }),
			});

			if (!response.ok) {
				if (response.status === 401) {
					window.localStorage.removeItem(TOKEN_KEY);
					setAuthToken("");
					throw new Error("Session expired. Please log in again.");
				}
				throw new Error(`Token creation failed with status ${response.status}`);
			}

			const payload = (await response.json()) as CreateEnrollmentTokenResponse;
			setGeneratedToken(payload);
		} catch (createError) {
			setTokenError(createError instanceof Error ? createError.message : "Failed to create token");
		} finally {
			setIsCreatingToken(false);
		}
	}

	if (!authToken) {
		return (
			<div className="app-shell">
				<div className="dashboard">
					<section className="hero">
						<div>
							<p className="eyebrow">Cloudfirewall Fleet</p>
							<h1>Admin login</h1>
						</div>
						<p>Sign in with the API's configured admin username and password to view enrolled agents.</p>
					</section>
					<form className="login-card" onSubmit={handleLogin}>
						<label className="field">
							<span>Username</span>
							<input value={username} onChange={(event) => setUsername(event.target.value)} />
						</label>
						<label className="field">
							<span>Password</span>
							<input
								type="password"
								value={password}
								onChange={(event) => setPassword(event.target.value)}
							/>
						</label>
						<button type="submit" disabled={isAuthenticating}>
							{isAuthenticating ? "Signing in..." : "Login"}
						</button>
						{authError ? <div className="error-banner">Unable to login: {authError}</div> : null}
					</form>
				</div>
			</div>
		);
	}

	return (
		<div className="app-shell">
			<div className="dashboard">
				<section className="hero">
					<div>
						<p className="eyebrow">Cloudfirewall Fleet</p>
						<h1>Agent heartbeat and firewall rollout dashboard</h1>
					</div>
					<p>
						Track which agents are online, when they last checked in, and which nftables
						firewall version each host is currently running.
					</p>
					<div className="stats">
						<div className="stat-card">
							<strong>{agents.length}</strong>
							<span>Total agents</span>
						</div>
						<div className="stat-card">
							<strong>{onlineAgents}</strong>
							<span>Online now</span>
						</div>
						<div className="stat-card">
							<strong>{Math.max(agents.length - onlineAgents, 0)}</strong>
							<span>Offline</span>
						</div>
					</div>
				</section>

				<div className="toolbar">
					<div className="toolbar-actions">
						<button onClick={() => window.location.reload()}>Refresh</button>
						<button onClick={() => void handleCreateEnrollmentToken()} disabled={isCreatingToken}>
							{isCreatingToken ? "Generating..." : "Generate Enrollment Token"}
						</button>
						<button className="secondary-button" onClick={handleLogout}>
							Logout
						</button>
					</div>
					<small>{isLoading ? "Loading agents..." : `Last updated ${lastUpdated || "just now"}`}</small>
				</div>

				{error ? <div className="error-banner">Unable to load agents: {error}</div> : null}
				{tokenError ? <div className="error-banner">Unable to create enrollment token: {tokenError}</div> : null}

				{generatedToken ? (
					<section className="token-card">
						<div className="token-card-header">
							<div>
								<h2>Latest enrollment token</h2>
								<p>One-time token for a new agent. Expires at {formatTime(generatedToken.expiresAt)}.</p>
							</div>
						</div>
						<code className="token-value">{generatedToken.token}</code>
					</section>
				) : null}

				{agents.length === 0 && !isLoading ? (
					<div className="empty-state">
						No agents have enrolled yet. Generate an enrollment token here, then run an agent
						with that one-time token to populate this dashboard.
					</div>
				) : null}

				<div className="agent-grid">
					{agents.map((agent) => (
						<article className="agent-card" key={agent.id}>
							<header>
								<div>
									<h2>{agent.name}</h2>
									<p>{agent.hostname || "hostname pending"}</p>
								</div>
								<span className={`badge ${agent.online ? "online" : "offline"}`}>
									{agent.online ? "Online" : "Offline"}
								</span>
							</header>

							<div className="meta-row">
								<span className="meta-label">Agent version</span>
								<span className="meta-value">{agent.agentVersion || "unknown"}</span>
							</div>
							<div className="meta-row">
								<span className="meta-label">Firewall version</span>
								<span className="meta-value">{agent.firewallVersion || "not applied"}</span>
							</div>
							<div className="meta-row">
								<span className="meta-label">Last heartbeat</span>
								<span className="meta-value">{formatTime(agent.lastSeenAt)}</span>
							</div>
							<div className="meta-row">
								<span className="meta-label">Enrolled</span>
								<span className="meta-value">{formatTime(agent.enrolledAt)}</span>
							</div>
						</article>
					))}
				</div>
			</div>
		</div>
	);
}

function formatTime(value?: string) {
	if (!value) {
		return "waiting for first heartbeat";
	}

	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return value;
	}

	return date.toLocaleString();
}
