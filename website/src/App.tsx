import { useEffect } from 'react';
import Navbar from './components/Navbar';
import Hero from './components/Hero';
import Pillars from './components/Pillars';
import Architecture from './components/Architecture';
import TerminalPreview from './components/TerminalPreview';
import QuickStart from './components/QuickStart';
import Arsenal from './components/Arsenal';
import FAQ from './components/FAQ';
import Footer from './components/Footer';

function App() {
  useEffect(() => {
    const banner = `
  _____                                 _____ _               
 |  __ \\                               / ____| |              
 | |  | |_ __ ___  __ _  ___  _ __    | |    | | __ ___      __
 | |  | | '__/ _ \\/ _\` |/ _ \\| '_ \\   | |    | |/ _\` \\ \\ /\\ / /
 | |__| | | | (_) | (_| | (_) | | | |  | |____| | (_| |\\ V  V / 
 |_____/|_|  \\___/ \\__, |\\___/|_| |_|   \\_____|_|\\__,_| \\_/\\_/  
                    __/ |                                     
                   |___/                                      

[*] DrogonClaw Autonomous C2 Framework
[*] Warning: Authorized access only.
`;
    console.log('%c' + banner, 'color: #ff2a4b; font-weight: bold; font-family: monospace;');
  }, []);

  return (
    <>
      <div className="bg-gradient"></div>
      <div className="grid-overlay"></div>

      <div className="container">
        <Navbar />
        <Hero />
        <TerminalPreview />
        <QuickStart />
        <Pillars />
        <Arsenal />
        <Architecture />
        <FAQ />
      </div>

      <Footer />
    </>
  );
}

export default App;
