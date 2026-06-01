import { useEffect, useState } from 'react';
import { Terminal, ShieldAlert, Cpu, Network, Lock, GitBranch } from 'lucide-react';

function App() {
  const [typedText, setTypedText] = useState('');
  const fullText = "npm install -g drogonclaw";

  useEffect(() => {
    let currentText = '';
    let i = 0;
    const interval = setInterval(() => {
      if (i < fullText.length) {
        currentText += fullText[i];
        setTypedText(currentText);
        i++;
      } else {
        clearInterval(interval);
      }
    }, 100);
    return () => clearInterval(interval);
  }, []);

  return (
    <>
      <div className="bg-gradient"></div>
      <div className="grid-overlay"></div>

      <div className="container">
        <nav>
          <div className="nav-brand">
            <img src="/logo.png" alt="DrogonClaw" className="nav-logo" />
            <span>DROGONCLAW</span>
          </div>
          <div className="nav-links">
            <a href="#features">Features</a>
            <a href="#documentation">Documentation</a>
            <a href="https://github.com/0xP4X/DrogoClaw" target="_blank" rel="noreferrer" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <GitBranch size={18} /> GitHub
            </a>
          </div>
        </nav>

        <section className="hero">
          <div className="hero-badge">
            <Lock size={14} style={{ display: 'inline', marginRight: '6px', verticalAlign: 'middle' }} />
            Authorized Offensive Security Only
          </div>
          <h1>Autonomous C2 Brain <br/> for Penetration Testing.</h1>
          <p>
            DrogonClaw is an AI-driven cyber operations framework. It plans attacks, orchestrates autonomous agent swarms, and executes native exploits inside sandboxed environments.
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

        <section id="terminal" className="terminal-section">
          <div className="terminal-window">
            <div className="terminal-header">
              <div className="terminal-dot dot-red"></div>
              <div className="terminal-dot dot-yellow"></div>
              <div className="terminal-dot dot-green"></div>
            </div>
            <div className="terminal-body">
              <div><span className="cmd-prompt">astra@kali:~$</span> <span className="cmd-text">{typedText}</span><span className="animate-pulse">_</span></div>
              <br/>
              {typedText.length === fullText.length && (
                <div style={{ opacity: 0, animation: 'fadeIn 0.5s forwards 0.5s' }}>
                  <div style={{ color: '#27c93f' }}>+ drogonclaw@0.2.0</div>
                  <div style={{ color: '#888' }}>added 1 package, and audited 283 packages in 3s</div>
                  <br/>
                  <div><span className="cmd-prompt">astra@kali:~$</span> <span className="cmd-text">drogonclaw</span></div>
                  <div style={{ color: '#dc143c', fontWeight: 'bold', marginTop: '1rem' }}>  [*] DROGONCLAW</div>
                  <div style={{ color: '#dc143c' }}>  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</div>
                  <div style={{ color: '#888' }}>  Autonomous Offensive Security Framework <span style={{ color: '#dc143c'}}>v0.2.0</span></div>
                  <br/>
                  <div style={{ color: '#00ffff' }}>drogon&gt; Scan 10.10.10.5 for vulnerabilities and pivot.</div>
                  <div style={{ color: '#888' }}>  ┃  ┌─ <span style={{ color: '#ffbd2e'}}>[🧠] Neural Processing</span></div>
                  <div style={{ color: '#888' }}>  ┃  ├─ <span style={{ color: '#00ffff'}}>Executing nmap_scan...</span></div>
                </div>
              )}
            </div>
          </div>
        </section>

        <section id="features" className="features">
          <div className="feature-card">
            <div className="feature-icon">
              <Cpu size={24} />
            </div>
            <h3>Intelligence Graph</h3>
            <p>Persistent graph-based memory mapping discovered assets, credentials, and open ports across your entire engagement.</p>
          </div>
          
          <div className="feature-card">
            <div className="feature-icon">
              <Terminal size={24} />
            </div>
            <h3>Sandboxed Execution</h3>
            <p>100% genuine execution. The AI spins up isolated Docker containers to run Metasploit, Nmap, and weaponized Python payloads safely.</p>
          </div>

          <div className="feature-card">
            <div className="feature-icon">
              <Network size={24} />
            </div>
            <h3>Swarm Commander</h3>
            <p>The core orchestrator delegates sub-tasks to specialized subagents for simultaneous parallel reconnaissance and exploitation.</p>
          </div>

          <div className="feature-card">
            <div className="feature-icon">
              <ShieldAlert size={24} />
            </div>
            <h3>Zero Hallucinations</h3>
            <p>The Evidence Validator module forces the agent to provide reproducible proof of exploitation before confirming success.</p>
          </div>
        </section>
      </div>

      <footer>
        <div className="container">
          <img src="/logo.png" alt="DrogonClaw" style={{ width: '32px', opacity: 0.5, marginBottom: '1rem' }} />
          <p>&copy; {new Date().getFullYear()} DrogonClaw. Developed by 0day (0xP4X).</p>
          <p style={{ fontSize: '0.875rem', marginTop: '0.5rem' }}>For authorized offensive security testing only.</p>
        </div>
      </footer>

      <style dangerouslySetInnerHTML={{__html: `
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        .animate-pulse {
          animation: pulse 1s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0; }
        }
      `}} />
    </>
  );
}

export default App;
