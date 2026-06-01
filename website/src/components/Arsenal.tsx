import { motion, type Variants } from 'framer-motion';
import { TerminalSquare, Shield, Crosshair, Radar, Code2, ServerCrash } from 'lucide-react';

const containerVariants: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.1 }
  }
};

const itemVariants: Variants = {
  hidden: { opacity: 0, scale: 0.9, y: 20 },
  visible: { opacity: 1, scale: 1, y: 0, transition: { duration: 0.5, ease: "easeOut" } }
};

export default function Arsenal() {
  return (
    <section id="arsenal" className="features-section">
      <h2 className="section-title">Supported Arsenal</h2>
      <p className="section-subtitle">
        DrogonClaw orchestrates real, industry-standard offensive tools natively inside its secure sandbox.
      </p>
      
      <motion.div 
        variants={containerVariants}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-50px" }}
        className="features-grid"
        style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))' }}
      >
        <motion.div variants={itemVariants} className="glass-card feature-item" style={{ textAlign: 'center', padding: '2rem' }}>
          <div className="feature-icon" style={{ margin: '0 auto 1.5rem' }}>
            <Radar size={24} />
          </div>
          <h3 style={{ fontSize: '1.25rem' }}>Nmap / RustScan</h3>
        </motion.div>
        
        <motion.div variants={itemVariants} className="glass-card feature-item" style={{ textAlign: 'center', padding: '2rem' }}>
          <div className="feature-icon" style={{ margin: '0 auto 1.5rem' }}>
            <Shield size={24} />
          </div>
          <h3 style={{ fontSize: '1.25rem' }}>Metasploit</h3>
        </motion.div>

        <motion.div variants={itemVariants} className="glass-card feature-item" style={{ textAlign: 'center', padding: '2rem' }}>
          <div className="feature-icon" style={{ margin: '0 auto 1.5rem' }}>
            <ServerCrash size={24} />
          </div>
          <h3 style={{ fontSize: '1.25rem' }}>SQLmap</h3>
        </motion.div>

        <motion.div variants={itemVariants} className="glass-card feature-item" style={{ textAlign: 'center', padding: '2rem' }}>
          <div className="feature-icon" style={{ margin: '0 auto 1.5rem' }}>
            <Crosshair size={24} />
          </div>
          <h3 style={{ fontSize: '1.25rem' }}>Gobuster / Ffuf</h3>
        </motion.div>

        <motion.div variants={itemVariants} className="glass-card feature-item" style={{ textAlign: 'center', padding: '2rem' }}>
          <div className="feature-icon" style={{ margin: '0 auto 1.5rem' }}>
            <TerminalSquare size={24} />
          </div>
          <h3 style={{ fontSize: '1.25rem' }}>Hydra</h3>
        </motion.div>

        <motion.div variants={itemVariants} className="glass-card feature-item" style={{ textAlign: 'center', padding: '2rem' }}>
          <div className="feature-icon" style={{ margin: '0 auto 1.5rem', background: 'rgba(255, 42, 75, 0.2)', color: '#ff2a4b' }}>
            <Code2 size={24} />
          </div>
          <h3 style={{ fontSize: '1.25rem' }}>Custom Scripts</h3>
        </motion.div>
      </motion.div>
    </section>
  );
}
