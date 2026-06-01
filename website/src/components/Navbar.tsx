import { GitBranch } from 'lucide-react';
import { motion } from 'framer-motion';

export default function Navbar() {
  return (
    <motion.nav 
      initial={{ y: -50, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ duration: 0.6, ease: "easeOut" }}
    >
      <div className="nav-brand">
        <img src="/logo.png" alt="DrogonClaw" className="nav-logo" />
        <span>DrogonClaw</span>
      </div>
      <div className="nav-links">
        <a href="#pillars">Pillars</a>
        <a href="#features">Features</a>
        <a href="#cli">CLI</a>
        <a href="#quickstart">Quick Start</a>
        <a href="https://github.com/0xP4X/DrogoClaw" target="_blank" rel="noreferrer" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--text-main)' }}>
          <GitBranch size={18} /> GitHub
        </a>
      </div>
    </motion.nav>
  );
}
