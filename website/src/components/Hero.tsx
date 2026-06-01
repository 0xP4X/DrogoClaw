import { Terminal, GitBranch } from 'lucide-react';
import { motion } from 'framer-motion';

export default function Hero() {
  return (
    <section className="hero">
      <motion.h1
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        Autonomous Offensive Security
      </motion.h1>

      <motion.p
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1 }}
      >
        DrogonClaw is an advanced C2 operations framework. It dynamically plans attack vectors, orchestrates concurrent agent swarms, and executes genuine exploits inside strictly isolated sandboxes. 
      </motion.p>
      
      <motion.div 
        className="hero-cta"
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.2 }}
      >
        <a href="#terminal" className="btn btn-primary">
          <Terminal size={16} />
          Install CLI
        </a>
        <a href="https://github.com/0xP4X/DrogoClaw" target="_blank" rel="noreferrer" className="btn btn-secondary">
          <GitBranch size={16} />
          GitHub
        </a>
      </motion.div>
    </section>
  );
}
