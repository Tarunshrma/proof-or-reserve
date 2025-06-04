import React, { useState, useEffect } from 'react';
import './App.css';
// import { ASSETS_CONFIG } from './config/assets'; // Will be removed
import AssetCard from './components/AssetCard';
import { getConfiguredAssets } from './services/api';
import type { AssetConfig } from './types'; // Import AssetConfig type

function App() {
  const [assets, setAssets] = useState<AssetConfig[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadAssets = async () => {
      try {
        setIsLoading(true);
        const fetchedAssets = await getConfiguredAssets();
        setAssets(fetchedAssets);
        setError(null);
      } catch (err) {
        if (err instanceof Error) {
          setError(err.message);
        } else {
          setError('An unknown error occurred while fetching asset configurations.');
        }
        console.error("Failed to fetch asset configurations:", err);
        setAssets([]); // Clear assets on error
      } finally {
        setIsLoading(false);
      }
    };

    loadAssets();
  }, []);

  return (
    <>
      <header className="app-header">
        Proof of Reserve Dashboard
      </header>
      <main className="assets-grid">
        {isLoading && <p style={{ textAlign: 'center', fontSize: '1.2rem' }}>Loading asset configurations...</p>}
        {error && <p style={{ textAlign: 'center', color: 'var(--error-color)', fontSize: '1.2rem' }}>Error: {error}</p>}
        {!isLoading && !error && assets.length === 0 && (
          <p style={{ textAlign: 'center', fontSize: '1.2rem' }}>No asset configurations found.</p>
        )}
        {!isLoading && !error && assets.map(asset => (
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
