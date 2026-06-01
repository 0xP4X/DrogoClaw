import Navbar from './components/Navbar';
import Hero from './components/Hero';
import Architecture from './components/Architecture';
import TerminalPreview from './components/TerminalPreview';
import Footer from './components/Footer';

function App() {
  return (
    <>
      <div className="bg-gradient"></div>
      <div className="grid-overlay"></div>

      <div className="container">
        <Navbar />
        <Hero />
        <TerminalPreview />
        <Architecture />
      </div>

      <Footer />

      <style dangerouslySetInnerHTML={{__html: `
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        .animate-pulse {
          animation: pulse 1s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0; }
        }
      `}} />
    </>
  );
}

export default App;
