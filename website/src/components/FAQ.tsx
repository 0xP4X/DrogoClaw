import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { ChevronDown } from 'lucide-react';

const faqs = [
  {
    question: "Do I need Docker installed to use DrogonClaw?",
    answer: "Yes. DrogonClaw relies on Docker to create isolated sandbox environments where it can safely execute potentially dangerous offensive security tools (Nmap, Metasploit, etc.) without compromising your host machine."
  },
  {
    question: "Which LLM providers are supported?",
    answer: "DrogonClaw supports OpenAI (GPT-4o), Anthropic (Claude 3.5 Sonnet), and OpenRouter for cloud inference. It also supports local inference via Ollama if you require a fully air-gapped or private engagement."
  },
  {
    question: "Is it safe to run on my host machine?",
    answer: "Yes. The AI Brain runs natively on your machine, but all exploit execution, network scanning, and payload generation occurs strictly within ephemeral Docker containers that are destroyed after the engagement."
  },
  {
    question: "How does the Evidence Validator prevent hallucination?",
    answer: "Standard LLMs often hallucinate success. DrogonClaw enforces a strict Evidence Validation layer: if an agent claims to have exploited a target, the Validator parses the CLI output or extracts the flag. If the evidence is insufficient, the claim is rejected."
  }
];

export default function FAQ() {
  const [openIndex, setOpenIndex] = useState<number | null>(null);

  return (
    <section id="faq" className="features-section">
      <h2 className="section-title">Frequently Asked Questions</h2>
      <p className="section-subtitle">
        Everything you need to know about deploying and operating DrogonClaw.
      </p>
      
      <div style={{ maxWidth: '800px', margin: '0 auto', display: 'flex', flexDirection: 'column', gap: '1rem' }}>
        {faqs.map((faq, index) => {
          const isOpen = openIndex === index;
          return (
            <motion.div 
              key={index}
              initial={{ opacity: 0, y: 10 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: index * 0.1 }}
              className="glass-card"
              style={{ padding: '1.5rem', cursor: 'pointer' }}
              onClick={() => setOpenIndex(isOpen ? null : index)}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <h3 style={{ fontSize: '1.15rem', fontWeight: 600, color: isOpen ? 'var(--accent)' : 'var(--text-main)', transition: 'color 0.3s' }}>
                  {faq.question}
                </h3>
                <motion.div animate={{ rotate: isOpen ? 180 : 0 }} transition={{ duration: 0.3 }}>
                  <ChevronDown size={20} color={isOpen ? 'var(--accent)' : 'var(--text-muted)'} />
                </motion.div>
              </div>
              
              <AnimatePresence>
                {isOpen && (
                  <motion.div
                    initial={{ height: 0, opacity: 0 }}
                    animate={{ height: 'auto', opacity: 1 }}
                    exit={{ height: 0, opacity: 0 }}
                    transition={{ duration: 0.3, ease: 'easeInOut' }}
                    style={{ overflow: 'hidden' }}
                  >
                    <p style={{ marginTop: '1rem', color: 'var(--text-muted)', lineHeight: '1.6' }}>
                      {faq.answer}
                    </p>
                  </motion.div>
                )}
              </AnimatePresence>
            </motion.div>
          );
        })}
      </div>
    </section>
  );
}
