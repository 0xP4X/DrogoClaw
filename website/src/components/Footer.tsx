export default function Footer() {
  return (
    <footer>
      <div className="container">
        <img src="/logo.png" alt="DrogonClaw" style={{ width: '40px', opacity: 0.5, marginBottom: '1rem' }} />
        <p>&copy; {new Date().getFullYear()} DrogonClaw. Developed by 0day (0xP4X).</p>
        <p style={{ fontSize: '0.875rem', marginTop: '0.5rem' }}>For authorized offensive security testing only.</p>
      </div>
    </footer>
  );
}
