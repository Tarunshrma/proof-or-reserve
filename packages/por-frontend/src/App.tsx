import React from 'react';
import './App.css';
import { ASSETS_CONFIG } from './config/assets';
import AssetCard from './components/AssetCard';

function App() {
  return (
    <>
      <header className="app-header">
        Proof of Reserve Dashboard
      </header>
      <main className="assets-grid">
        {ASSETS_CONFIG.map(asset => (
          <AssetCard key={asset.id} asset={asset} />
        ))}
      </main>
      <footer className="app-footer">
        <p>&copy; {new Date().getFullYear()} Proof of Reserve System. All rights reserved.</p>
      </footer>
    </>
  );
}

export default App;
