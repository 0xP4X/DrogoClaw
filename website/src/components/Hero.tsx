import { Terminal, GitBranch } from 'lucide-react';
import { motion } from 'framer-motion';

export default function Hero() {
  return (
    <section className="hero">
      <motion.img 
        src="/logo.png" 
        alt="DrogonClaw Logo"
        className="hero-logo"
        initial={{ opacity: 0, scale: 0.5, rotate: -10 }}
        animate={{ opacity: 1, scale: 1, rotate: 0 }}
        transition={{ duration: 0.7, type: "spring", bounce: 0.4 }}
        style={{ width: '120px', height: '120px', marginBottom: '1.5rem', filter: 'drop-shadow(0 0 20px rgba(255, 42, 75, 0.4))' }}
      />
      
      <motion.h1
        initial={{ opacity: 0, y: 15 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, delay: 0.1 }}
      >
        DrogonClaw
      </motion.h1>
      
      <motion.h2
        initial={{ opacity: 0, y: 15 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, delay: 0.15 }}
        style={{ fontSize: '1.5rem', fontWeight: 500, color: 'var(--text-main)', marginBottom: '1.5rem', textAlign: 'center' }}
      >
        Autonomous Offensive Security Framework
      </motion.h2>

      <motion.p
        initial={{ opacity: 0, y: 15 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, delay: 0.2 }}
      >
        DrogonClaw operates as a Command-and-Control (C2) Brain. It understands objectives, plans attack workflows, adapts to new discoveries, and orchestrates a swarm of specialized autonomous agents through a unified intelligence core.
      </motion.p>
      
      <motion.div 
        className="hero-cta"
        initial={{ opacity: 0, y: 15 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, delay: 0.3 }}
      >
        <a href="#quickstart" className="btn btn-primary">
          <Terminal size={18} />
          Install CLI
        </a>
        <a href="https://github.com/0xP4X/DrogoClaw" target="_blank" rel="noreferrer" className="btn btn-secondary">
          <GitBranch size={18} />
          View Source
        </a>
      </motion.div>
    </section>
  );
}
