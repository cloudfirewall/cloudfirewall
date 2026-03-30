import { startTransition, useEffect, useState } from "react";

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

const API_BASE = import.meta.env.VITE_API_BASE ?? "";
const POLL_INTERVAL_MS = 5000;

export default function App() {
	const [agents, setAgents] = useState<AgentSummary[]>([]);
	const [error, setError] = useState("");
	const [isLoading, setIsLoading] = useState(true);
	const [lastUpdated, setLastUpdated] = useState("");

	useEffect(() => {
		let cancelled = false;

		async function loadAgents() {
			try {
				const response = await fetch(`${API_BASE}/api/v1/agents`);
				if (!response.ok) {
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
	}, []);

	const onlineAgents = agents.filter((agent) => agent.online).length;

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
					<button onClick={() => window.location.reload()}>Refresh</button>
					<small>
						{isLoading ? "Loading agents..." : `Last updated ${lastUpdated || "just now"}`}
					</small>
				</div>

				{error ? <div className="error-banner">Unable to load agents: {error}</div> : null}

				{agents.length === 0 && !isLoading ? (
					<div className="empty-state">
						No agents have enrolled yet. Start the API, then run an agent with an enrollment
						token to populate this dashboard.
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
