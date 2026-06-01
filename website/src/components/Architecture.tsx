import { Cpu, Terminal, Network, ShieldAlert, Smartphone } from 'lucide-react';

export default function Architecture() {
  return (
    <section id="architecture" className="features">
      <div className="section-header" style={{ gridColumn: '1 / -1', textAlign: 'center', marginBottom: '2rem' }}>
        <h2 style={{ fontSize: '2.5rem', marginBottom: '1rem' }}>System Architecture</h2>
        <p style={{ color: 'var(--text-muted)', maxWidth: '700px', margin: '0 auto' }}>
          DrogonClaw replaces manual scripting with a persistent, intelligent C2 brain. 
          Discover how the framework orchestrates complex penetration testing workflows.
        </p>
      </div>

      <div className="feature-card">
        <div className="feature-icon">
          <Cpu size={24} />
        </div>
        <h3>The Intelligence Graph</h3>
        <p>A persistent, graph-based memory system that maps out discovered assets, IPs, credentials, and open ports across your entire engagement. The AI uses this context to automatically chain vulnerabilities together.</p>
      </div>
      
      <div className="feature-card">
        <div className="feature-icon">
          <Terminal size={24} />
        </div>
        <h3>Sandboxed Execution</h3>
        <p>100% genuine execution. The AI spins up isolated Docker containers to run real tools like Metasploit, Nmap, and weaponized Python payloads safely on your host machine without risk of self-compromise.</p>
      </div>

      <div className="feature-card">
        <div className="feature-icon">
          <Network size={24} />
        </div>
        <h3>Swarm Commander</h3>
        <p>The core orchestrator delegates sub-tasks to specialized autonomous subagents. While one agent runs directory fuzzing, another can simultaneously attempt brute-force authentication.</p>
      </div>

      <div className="feature-card">
        <div className="feature-icon">
          <Smartphone size={24} />
        </div>
        <h3>Telegram C2 Gateway</h3>
        <p>Control your agent swarm from anywhere. By passing your Telegram Chat ID, you can securely text instructions to DrogonClaw from your mobile device and receive real-time updates and exploit proofs.</p>
      </div>

      <div className="feature-card" style={{ gridColumn: '1 / -1', marginTop: '2rem', background: 'rgba(220, 20, 60, 0.05)', borderColor: 'rgba(220, 20, 60, 0.2)' }}>
        <div className="feature-icon" style={{ background: 'var(--accent)', color: 'white' }}>
          <ShieldAlert size={24} />
        </div>
        <h3>Zero Hallucinations: The Evidence Validator</h3>
        <p style={{ maxWidth: '800px' }}>
          Unlike standard LLMs that confidently hallucinate success, DrogonClaw is equipped with a strict Evidence Validation layer. If an agent claims it found a vulnerability, the Validator forces the agent to provide reproducible CLI outputs, screenshots, or extracted flags. If the proof is insufficient, the claim is rejected and the agent is forced to try again.
        </p>
      </div>
    </section>
  );
}
