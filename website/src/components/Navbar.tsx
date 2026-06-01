import { GitBranch } from 'lucide-react';
import { motion } from 'framer-motion';

export default function Navbar() {
  return (
    <motion.nav 
      initial={{ y: -100 }}
      animate={{ y: 0 }}
      transition={{ duration: 0.5, ease: "easeOut" }}
    >
      <div className="nav-brand">
        <img src="/logo.png" alt="DrogonClaw" className="nav-logo" />
        <span>DROGONCLAW</span>
      </div>
      <div className="nav-links">
        <a href="#architecture">Architecture</a>
        <a href="#terminal">CLI</a>
        <a href="https://github.com/0xP4X/DrogoClaw" target="_blank" rel="noreferrer" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <GitBranch size={18} /> GitHub
        </a>
      </div>
    </motion.nav>
  );
}
