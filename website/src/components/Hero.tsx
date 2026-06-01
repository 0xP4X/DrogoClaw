import { Terminal, Lock, GitBranch } from 'lucide-react';

export default function Hero() {
  return (
    <section className="hero">
      <div className="hero-badge">
        <Lock size={14} style={{ display: 'inline', marginRight: '6px', verticalAlign: 'middle' }} />
        Authorized Offensive Security Only
      </div>
      
      <img 
        src="/logo.png" 
        alt="DrogonClaw Logo" 
        style={{ width: '180px', height: '180px', marginBottom: '2rem', filter: 'drop-shadow(0 0 20px rgba(220, 20, 60, 0.4))' }} 
      />

      <h1>Autonomous C2 Brain <br/> for Penetration Testing.</h1>
      <p>
        DrogonClaw is an AI-driven cyber operations framework. It plans attacks, orchestrates autonomous agent swarms, and executes native exploits inside sandboxed environments. No toys. No mock data. 100% genuine execution.
      </p>
      
      <div className="hero-cta">
        <a href="#terminal" className="btn btn-primary">
          <Terminal size={20} />
          Install CLI
        </a>
        <a href="https://github.com/0xP4X/DrogoClaw" target="_blank" rel="noreferrer" className="btn btn-secondary">
          <GitBranch size={20} />
          View Source
        </a>
      </div>
    </section>
  );
}
