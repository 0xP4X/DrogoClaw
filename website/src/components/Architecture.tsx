import { Cpu, Terminal, Network, ShieldAlert, Smartphone, Fingerprint, Database, Zap } from 'lucide-react';
import { motion, type Variants } from 'framer-motion';

const containerVariants: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.15 }
  }
};

const itemVariants: Variants = {
  hidden: { opacity: 0, y: 40 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.6, ease: "easeOut" } }
};

export default function Architecture() {
  return (
    <section id="architecture" className="features-section">
      <div className="section-header" style={{ textAlign: 'center', marginBottom: '4rem' }}>
        <motion.h2 
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: "-100px" }}
          style={{ fontSize: '3rem', marginBottom: '1rem', fontWeight: 800, letterSpacing: '-1px' }}
        >
          System Architecture
        </motion.h2>
        <motion.p 
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: "-100px" }}
          style={{ color: 'var(--text-muted)', maxWidth: '700px', margin: '0 auto', fontSize: '1.2rem', lineHeight: '1.8' }}
        >
          DrogonClaw replaces manual scripting with a persistent, intelligent C2 brain. 
          Discover how the framework orchestrates complex penetration testing workflows autonomously.
        </motion.p>
      </div>

      <motion.div 
        variants={containerVariants}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-50px" }}
        className="features-grid"
      >
        {/* Core Pillars - 2x2 Grid */}
        <div className="features-2x2">
          <motion.div variants={itemVariants} className="feature-card">
            <div className="feature-icon"><Cpu size={24} /></div>
            <h3>The Intelligence Graph</h3>
            <p>A persistent, graph-based memory system that maps out discovered assets, IPs, credentials, and open ports across your entire engagement. The AI uses this context to automatically chain vulnerabilities together.</p>
          </motion.div>
          
          <motion.div variants={itemVariants} className="feature-card">
            <div className="feature-icon"><Terminal size={24} /></div>
            <h3>Sandboxed Execution</h3>
            <p>100% genuine execution. The AI spins up isolated Docker containers to run real tools like Metasploit, Nmap, and weaponized payloads safely on your host machine without risk of self-compromise.</p>
          </motion.div>

          <motion.div variants={itemVariants} className="feature-card">
            <div className="feature-icon"><Network size={24} /></div>
            <h3>Swarm Commander</h3>
            <p>The core orchestrator delegates sub-tasks to specialized autonomous subagents. While one agent runs directory fuzzing, another can simultaneously attempt brute-force authentication.</p>
          </motion.div>

          <motion.div variants={itemVariants} className="feature-card">
            <div className="feature-icon"><Smartphone size={24} /></div>
            <h3>Telegram C2 Gateway</h3>
            <p>Control your agent swarm from anywhere. By passing your Telegram Chat ID, you can securely text instructions to DrogonClaw from your mobile device and receive real-time updates and exploit proofs.</p>
          </motion.div>
        </div>

        {/* Highlighted Full-Width Card */}
        <motion.div variants={itemVariants} className="feature-card highlight-card">
          <div className="feature-icon highlight-icon">
            <ShieldAlert size={24} />
          </div>
          <div className="highlight-content">
            <h3>Zero Hallucinations: The Evidence Validator</h3>
            <p>
              Unlike standard LLMs that confidently hallucinate success, DrogonClaw is equipped with a strict Evidence Validation layer. If an agent claims it found a vulnerability, the Validator forces the agent to provide reproducible CLI outputs, screenshots, or extracted flags. If the proof is insufficient, the claim is rejected and the agent is forced to try again.
            </p>
          </div>
        </motion.div>

        {/* Additional Info Row - 3 Columns */}
        <div className="features-3col">
          <motion.div variants={itemVariants} className="feature-card small-card">
            <div className="feature-icon small-icon"><Fingerprint size={20} /></div>
            <h4>Stealth Mode Operations</h4>
            <p>Built-in rate limiting and evasion techniques to bypass basic intrusion detection systems during scans.</p>
          </motion.div>
          
          <motion.div variants={itemVariants} className="feature-card small-card">
            <div className="feature-icon small-icon"><Database size={20} /></div>
            <h4>Automated Reporting</h4>
            <p>Generates comprehensive, boardroom-ready PDF reports with detailed reproduction steps for every vulnerability.</p>
          </motion.div>

          <motion.div variants={itemVariants} className="feature-card small-card">
            <div className="feature-icon small-icon"><Zap size={20} /></div>
            <h4>Dynamic Extensibility</h4>
            <p>Write custom modules in TypeScript. DrogonClaw dynamically loads new attack vectors and integrates them into its decision matrix.</p>
          </motion.div>
        </div>
      </motion.div>
    </section>
  );
}
