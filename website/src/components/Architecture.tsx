import { motion, type Variants } from 'framer-motion';
import { Database, Box, CheckCircle, Smartphone, FileText, Code2 } from 'lucide-react';

const containerVariants: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.15 }
  }
};

const itemVariants: Variants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.5, ease: "easeOut" } }
};

export default function Architecture() {
  return (
    <section id="features" className="features-section">
      <h2 className="section-title">Core Features</h2>
      <p className="section-subtitle">
        Everything you need to orchestrate advanced, verifiable cyber operations.
      </p>
      
      <motion.div 
        variants={containerVariants}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-50px" }}
        className="features-grid"
        style={{ gridTemplateColumns: 'repeat(3, 1fr)' }}
      >
        <motion.div variants={itemVariants} className="glass-card feature-item">
          <div className="feature-icon">
            <Database size={24} />
          </div>
          <h3>The Intelligence Graph</h3>
          <p>A persistent, graph-based memory system that maps out discovered assets, IPs, credentials, and open ports. The AI uses this context to automatically chain vulnerabilities together.</p>
        </motion.div>
        
        <motion.div variants={itemVariants} className="glass-card feature-item">
          <div className="feature-icon">
            <Box size={24} />
          </div>
          <h3>Sandboxed Execution</h3>
          <p>100% genuine execution. The AI spins up isolated Docker containers to run real tools like Metasploit, Nmap, and payloads safely on your host machine without risk of self-compromise.</p>
        </motion.div>

        <motion.div variants={itemVariants} className="glass-card feature-item">
          <div className="feature-icon">
            <CheckCircle size={24} />
          </div>
          <h3>Zero Hallucinations</h3>
          <p>DrogonClaw is equipped with a strict Evidence Validation layer. It rejects unverified claims by forcing the agent to provide reproducible CLI outputs, screenshots, or extracted flags.</p>
        </motion.div>

        <motion.div variants={itemVariants} className="glass-card feature-item">
          <div className="feature-icon">
            <Smartphone size={24} />
          </div>
          <h3>Telegram C2 Gateway</h3>
          <p>Control your agent swarm from anywhere. Pass your Telegram Chat ID during initialization to securely text instructions to DrogonClaw from your mobile device and receive real-time updates.</p>
        </motion.div>

        <motion.div variants={itemVariants} className="glass-card feature-item">
          <div className="feature-icon">
            <FileText size={24} />
          </div>
          <h3>Automated Reporting</h3>
          <p>Generates comprehensive, boardroom-ready PDF reports with detailed reproduction steps for every verified vulnerability, ensuring clear communication of operational findings.</p>
        </motion.div>

        <motion.div variants={itemVariants} className="glass-card feature-item">
          <div className="feature-icon">
            <Code2 size={24} />
          </div>
          <h3>Extensible Architecture</h3>
          <p>Write custom modules in TypeScript. DrogonClaw dynamically loads new attack vectors, reconnaissance scripts, and API integrations into its autonomous decision matrix.</p>
        </motion.div>
      </motion.div>
    </section>
  );
}
