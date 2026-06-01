import { GitBranch } from 'lucide-react';

export default function Footer() {
  return (
    <footer>
      <div className="container" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
        <div>
          <p>&copy; {new Date().getFullYear()} DrogonClaw. Authorized security testing only.</p>
          <p style={{ fontSize: '0.85rem', marginTop: '0.5rem', color: '#666' }}>Developed by 0day (0xP4X). Licensed under MIT.</p>
        </div>
        
        <div style={{ display: 'flex', gap: '1.5rem' }}>
          <a href="https://github.com/0xP4X/DrogoClaw" target="_blank" rel="noreferrer" style={{ color: 'var(--text-muted)' }} aria-label="GitHub">
            <GitBranch size={20} />
          </a>
        </div>
      </div>
    </footer>
  );
}
