import { useEffect, useState, useRef } from 'react';
import { motion, useInView } from 'framer-motion';

export default function TerminalPreview() {
  const [typedText, setTypedText] = useState('');
  const fullText = "npm install -g drogonclaw";
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-100px" });

  useEffect(() => {
    if (!isInView) return;
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
    }, 80);
    return () => clearInterval(interval);
  }, [isInView]);

  return (
    <section id="cli" className="terminal-section" ref={ref}>
      <motion.div 
        className="terminal-window"
        initial={{ opacity: 0, rotateX: 10, y: 30 }}
        animate={isInView ? { opacity: 1, rotateX: 0, y: 0 } : {}}
        transition={{ duration: 0.8, ease: "easeOut" }}
      >
        <div className="terminal-header">
          <div className="terminal-dot dot-red"></div>
          <div className="terminal-dot dot-yellow"></div>
          <div className="terminal-dot dot-green"></div>
        </div>
        <div className="terminal-body">
          <div><span className="cmd-prompt">operator@c2:~$</span> <span className="cmd-text">{typedText}</span><span className="animate-pulse">_</span></div>
          <br/>
          {typedText.length === fullText.length && (
            <motion.div 
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.5, delay: 0.3 }}
            >
              <div className="cmd-success">+ drogonclaw@0.2.0</div>
              <div style={{ color: '#5c6370' }}>added 1 package, and audited 283 packages in 3s</div>
              <br/>
              <div><span className="cmd-prompt">operator@c2:~$</span> <span className="cmd-text">drogonclaw</span></div>
              <div style={{ color: '#ff2a4b', fontWeight: '700', marginTop: '1rem', letterSpacing: '1px' }}>  [*] DROGONCLAW</div>
              <div style={{ color: '#ff2a4b' }}>  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</div>
              <div style={{ color: '#8a8f98' }}>  Autonomous Offensive Security Framework <span style={{ color: '#ff2a4b'}}>v0.2.0</span></div>
              <br/>
              <div className="cmd-info">drogon&gt; Scan 10.10.10.5 for vulnerabilities and pivot.</div>
              <div style={{ color: '#5c6370' }}>  ┃  ┌─ <span style={{ color: '#e5c07b'}}>[🧠] Neural Processing: Breaking down objective...</span></div>
              <div style={{ color: '#5c6370' }}>  ┃  ├─ <span className="cmd-info">Executing nmap_scan inside Docker sandbox...</span></div>
              <div style={{ color: '#5c6370' }}>  ┃  └─ <span className="cmd-success">Discovered Open Ports: 22/tcp, 80/tcp</span></div>
            </motion.div>
          )}
        </div>
      </motion.div>
    </section>
  );
}
