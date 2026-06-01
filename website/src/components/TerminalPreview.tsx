import { useEffect, useState } from 'react';

export default function TerminalPreview() {
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
  );
}
