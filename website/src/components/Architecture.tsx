import { motion, type Variants } from 'framer-motion';

const containerVariants: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.1 }
  }
};

const itemVariants: Variants = {
  hidden: { opacity: 0, y: 10 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

export default function Architecture() {
  return (
    <section id="architecture" className="features-section">
      <motion.div 
        variants={containerVariants}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-50px" }}
        className="features-list"
      >
        <motion.div variants={itemVariants} className="feature-item">
          <h3>The Intelligence Graph</h3>
          <p>A persistent, graph-based memory system that maps out discovered assets, IPs, credentials, and open ports across your entire engagement. The AI uses this context to automatically chain vulnerabilities together.</p>
        </motion.div>
        
        <motion.div variants={itemVariants} className="feature-item">
          <h3>Sandboxed Execution</h3>
          <p>100% genuine execution. The AI spins up isolated Docker containers to run real tools like Metasploit, Nmap, and weaponized payloads safely on your host machine without risk of self-compromise.</p>
        </motion.div>

        <motion.div variants={itemVariants} className="feature-item">
          <h3>Zero Hallucinations: The Evidence Validator</h3>
          <p>Unlike standard LLMs that confidently hallucinate success, DrogonClaw is equipped with a strict Evidence Validation layer. If an agent claims it found a vulnerability, the Validator forces the agent to provide reproducible CLI outputs, screenshots, or extracted flags. If the proof is insufficient, the claim is rejected and the agent is forced to try again.</p>
        </motion.div>

        <motion.div variants={itemVariants} className="feature-item">
          <h3>Telegram C2 Gateway</h3>
          <p>Control your agent swarm from anywhere. By passing your Telegram Chat ID, you can securely text instructions to DrogonClaw from your mobile device and receive real-time updates and exploit proofs.</p>
        </motion.div>

        <motion.div variants={itemVariants} className="feature-item">
          <h3>Automated Reporting</h3>
          <p>Generates comprehensive, boardroom-ready PDF reports with detailed reproduction steps for every verified vulnerability.</p>
        </motion.div>
      </motion.div>
    </section>
  );
}
