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
    <section id="terminal" className="terminal-section" ref={ref}>
      <motion.div 
        className="terminal-window"
        initial={{ opacity: 0, rotateX: 20, y: 50 }}
        animate={isInView ? { opacity: 1, rotateX: 0, y: 0 } : {}}
        transition={{ duration: 0.8, ease: "easeOut" }}
      >
        <div className="terminal-header">
          <div className="terminal-dot dot-red"></div>
          <div className="terminal-dot dot-yellow"></div>
          <div className="terminal-dot dot-green"></div>
        </div>
        <div className="terminal-body">
          <div><span className="cmd-prompt">astra@kali:~$</span> <span className="cmd-text">{typedText}</span><span className="animate-pulse" style={{ animation: 'pulse 1s cubic-bezier(0.4, 0, 0.6, 1) infinite' }}>_</span></div>
          <br/>
          {typedText.length === fullText.length && (
            <motion.div 
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.5, delay: 0.3 }}
            >
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
            </motion.div>
          )}
        </div>
      </motion.div>
    </section>
  );
}
