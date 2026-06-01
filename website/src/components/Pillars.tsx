import { motion, type Variants } from 'framer-motion';
import { BrainCircuit, ShieldAlert, Cpu } from 'lucide-react';

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

export default function Pillars() {
  return (
    <section id="pillars" className="features-section">
      <h2 className="section-title">Architectural Pillars</h2>
      <p className="section-subtitle">
        DrogonClaw is built on three core pillars that enable high-confidence, autonomous security testing.
      </p>

      <motion.div 
        variants={containerVariants}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-50px" }}
        className="features-grid"
      >
        <motion.div variants={itemVariants} className="glass-card feature-item">
          <div className="feature-icon">
            <BrainCircuit size={24} />
          </div>
          <h3>The Orchestration Core</h3>
          <p>
            Features a Mission Planner that breaks down objectives, an Intelligence Graph mapping discovered assets across engagements, and an AI Evidence Validator demanding reproducible proof over hallucinations.
          </p>
        </motion.div>
        
        <motion.div variants={itemVariants} className="glass-card feature-item">
          <div className="feature-icon">
            <Cpu size={24} />
          </div>
          <h3>The Skill Ecosystem</h3>
          <p>
            A modular plugin architecture integrating OSINT modules, network reconnaissance scanners, browser automation packs, and exploit validators for specialized attack vectors.
          </p>
        </motion.div>

        <motion.div variants={itemVariants} className="glass-card feature-item">
          <div className="feature-icon">
            <ShieldAlert size={24} />
          </div>
          <h3>Autonomous Execution Layer</h3>
          <p>
            Isolates operational risk through sandboxed execution. Real tools like Metasploit and Nmap run inside strictly isolated Docker containers, governed by robust Safety Monitors enforcing rate limits.
          </p>
        </motion.div>
      </motion.div>
    </section>
  );
}
