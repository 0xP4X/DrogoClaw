import { motion, type Variants } from 'framer-motion';
import { TerminalSquare, Shield, Crosshair, Radar, Code2, ServerCrash } from 'lucide-react';

const containerVariants: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.05 }
  }
};

const itemVariants: Variants = {
  hidden: { opacity: 0, scale: 0.95, y: 10 },
  visible: { opacity: 1, scale: 1, y: 0, transition: { duration: 0.4, ease: "easeOut" } }
};

export default function Arsenal() {
  return (
    <section id="arsenal" className="features-section" style={{ padding: '2rem 0 6rem' }}>
      <h2 className="section-title" style={{ fontSize: '2rem' }}>Supported Arsenal</h2>
      <p className="section-subtitle" style={{ marginBottom: '3rem' }}>
        DrogonClaw orchestrates real, industry-standard offensive tools natively inside its secure sandbox.
      </p>
      
      <motion.div 
        variants={containerVariants}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-50px" }}
        style={{ 
          display: 'grid', 
          gridTemplateColumns: 'repeat(3, 1fr)', 
          gap: '1.5rem', 
          maxWidth: '900px', 
          margin: '0 auto' 
        }}
      >
        <motion.div variants={itemVariants} style={{ display: 'flex', alignItems: 'center', gap: '1rem', background: 'rgba(255,255,255,0.02)', border: '1px solid var(--border-color)', padding: '1.25rem', borderRadius: '12px' }}>
          <Radar size={20} color="var(--text-muted)" />
          <span style={{ fontWeight: 600, fontSize: '1.05rem' }}>Nmap / RustScan</span>
        </motion.div>
        
        <motion.div variants={itemVariants} style={{ display: 'flex', alignItems: 'center', gap: '1rem', background: 'rgba(255,255,255,0.02)', border: '1px solid var(--border-color)', padding: '1.25rem', borderRadius: '12px' }}>
          <Shield size={20} color="var(--text-muted)" />
          <span style={{ fontWeight: 600, fontSize: '1.05rem' }}>Metasploit</span>
        </motion.div>

        <motion.div variants={itemVariants} style={{ display: 'flex', alignItems: 'center', gap: '1rem', background: 'rgba(255,255,255,0.02)', border: '1px solid var(--border-color)', padding: '1.25rem', borderRadius: '12px' }}>
          <ServerCrash size={20} color="var(--text-muted)" />
          <span style={{ fontWeight: 600, fontSize: '1.05rem' }}>SQLmap</span>
        </motion.div>

        <motion.div variants={itemVariants} style={{ display: 'flex', alignItems: 'center', gap: '1rem', background: 'rgba(255,255,255,0.02)', border: '1px solid var(--border-color)', padding: '1.25rem', borderRadius: '12px' }}>
          <Crosshair size={20} color="var(--text-muted)" />
          <span style={{ fontWeight: 600, fontSize: '1.05rem' }}>Gobuster / Ffuf</span>
        </motion.div>

        <motion.div variants={itemVariants} style={{ display: 'flex', alignItems: 'center', gap: '1rem', background: 'rgba(255,255,255,0.02)', border: '1px solid var(--border-color)', padding: '1.25rem', borderRadius: '12px' }}>
          <TerminalSquare size={20} color="var(--text-muted)" />
          <span style={{ fontWeight: 600, fontSize: '1.05rem' }}>Hydra</span>
        </motion.div>

        <motion.div variants={itemVariants} style={{ display: 'flex', alignItems: 'center', gap: '1rem', background: 'rgba(255, 42, 75, 0.05)', border: '1px solid rgba(255, 42, 75, 0.2)', padding: '1.25rem', borderRadius: '12px' }}>
          <Code2 size={20} color="var(--accent)" />
          <span style={{ fontWeight: 600, fontSize: '1.05rem', color: 'var(--accent)' }}>Custom Scripts</span>
        </motion.div>
      </motion.div>
    </section>
  );
}
