import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { GitBranch, Sun, Moon } from 'lucide-react';

export default function Navbar() {
  const [isDark, setIsDark] = useState(() => {
    const saved = localStorage.getItem('theme');
    return saved ? saved === 'dark' : true;
  });

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light');
    localStorage.setItem('theme', isDark ? 'dark' : 'light');
  }, [isDark]);

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
        <a href="#cli">Demo</a>
        <a href="#quickstart">Quick Start</a>
        <a href="#features">Architecture</a>
        <a href="https://github.com/0xP4X/DrogoClaw" target="_blank" rel="noreferrer" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--text-main)' }}>
          <GitBranch size={18} /> GitHub
        </a>
        <button 
          className="theme-toggle"
          onClick={() => setIsDark(prev => !prev)}
          aria-label="Toggle theme"
          title={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          <motion.div
            key={isDark ? 'moon' : 'sun'}
            initial={{ rotate: -90, opacity: 0 }}
            animate={{ rotate: 0, opacity: 1 }}
            transition={{ duration: 0.3 }}
          >
            {isDark ? <Sun size={16} /> : <Moon size={16} />}
          </motion.div>
        </button>
      </div>
    </motion.nav>
  );
}
