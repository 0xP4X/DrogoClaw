import { useState } from 'react';
import { motion } from 'framer-motion';
import { Copy, Check } from 'lucide-react';

export default function QuickStart() {
  const [copied, setCopied] = useState(false);
  const command = "npm install -g drogonclaw";

  const handleCopy = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section id="quickstart" className="quickstart-section">
      <h2 className="section-title">Quick Start</h2>
      <p className="section-subtitle">
        Get up and running with DrogonClaw in seconds.
      </p>

      <div style={{ maxWidth: '800px', margin: '0 auto' }}>
        <motion.div 
          className="glass-card"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
        >
          <h3 style={{ fontSize: '1.25rem', marginBottom: '1rem' }}>Global Installation</h3>
          <p style={{ color: 'var(--text-muted)' }}>
            DrogonClaw is published on NPM and can be installed globally. Once installed, simply run <code>drogonclaw</code> from anywhere to launch the configuration wizard.
          </p>
          
          <div className="code-block">
            <code>{command}</code>
            <button onClick={handleCopy} className="copy-btn" aria-label="Copy code">
              {copied ? <Check size={18} color="#98c379" /> : <Copy size={18} />}
            </button>
          </div>
          
          <div style={{ marginTop: '2.5rem' }}>
            <h3 style={{ fontSize: '1.25rem', marginBottom: '1rem' }}>Initialization Wizard</h3>
            <ul style={{ color: 'var(--text-muted)', paddingLeft: '1.5rem', lineHeight: '1.8' }}>
              <li>Select your preferred AI Provider (OpenAI, Anthropic, OpenRouter, or local Ollama).</li>
              <li>Securely enter your API keys.</li>
              <li>Optionally configure a Telegram Gateway by providing your <code>TELEGRAM_CHAT_ID</code> for remote mobile C2 operations.</li>
            </ul>
          </div>
        </motion.div>
      </div>
    </section>
  );
}
