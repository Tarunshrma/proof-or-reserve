import React, { useState, useEffect, useRef } from 'react';
import './App.css';
// import { ASSETS_CONFIG } from './config/assets'; // Will be removed
import AssetCard from './components/AssetCard';
import { getConfiguredAssets } from './services/api';
import type { AssetConfig } from './types'; // Import AssetConfig type
import { WalletConnect } from './components/WalletConnect';

const App: React.FC = () => {
  const [assets, setAssets] = useState<AssetConfig[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [globalMessage, setGlobalMessage] = useState<{ type: 'success' | 'error', text: React.ReactNode } | null>(null);
  const [fadeOut, setFadeOut] = useState(false);
  const fadeTimeout = useRef<NodeJS.Timeout | null>(null);
  const hideTimeout = useRef<NodeJS.Timeout | null>(null);

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

  useEffect(() => {
    if (!globalMessage) return;
    setFadeOut(false);
    if (fadeTimeout.current) clearTimeout(fadeTimeout.current);
    if (hideTimeout.current) clearTimeout(hideTimeout.current);
    fadeTimeout.current = setTimeout(() => {
      setFadeOut(true);
    }, 1500); // Start fade out after 1.5s
    hideTimeout.current = setTimeout(() => {
      setGlobalMessage(null);
    }, 2000); // Remove after 2s
    return () => {
      if (fadeTimeout.current) clearTimeout(fadeTimeout.current);
      if (hideTimeout.current) clearTimeout(hideTimeout.current);
    };
  }, [globalMessage]);

  return (
    <div className="app">
      <header>
        <div className="header-content">
          <h1>Proof of Reserve</h1>
          <WalletConnect />
        </div>
      </header>

      {globalMessage && (
        <div className={`global-message ${globalMessage.type}${fadeOut ? ' fade-out' : ''}`} style={{
          margin: '24px auto 0 auto',
          maxWidth: 600,
          padding: '16px',
          borderRadius: '8px',
          fontWeight: 'bold',
          textAlign: 'center',
          background: globalMessage.type === 'success' ? '#e6ffed' : '#fff1f0',
          color: globalMessage.type === 'success' ? '#1a7f37' : '#cf1322',
          border: globalMessage.type === 'success' ? '1px solid #b7eb8f' : '1px solid #ffa39e',
          position: 'relative'
        }}>
          <button
            onClick={() => setGlobalMessage(null)}
            style={{
              position: 'absolute',
              top: 8,
              right: 12,
              background: 'transparent',
              border: 'none',
              fontSize: 20,
              fontWeight: 'bold',
              color: '#888',
              cursor: 'pointer',
              lineHeight: 1
            }}
            aria-label="Close notification"
          >
            ×
          </button>
          {globalMessage.text}
        </div>
      )}

      <main>
        <div className="assets-grid">
          {isLoading && <p style={{ textAlign: 'center', fontSize: '1.2rem' }}>Loading asset configurations...</p>}
          {error && <p style={{ textAlign: 'center', color: 'var(--error-color)', fontSize: '1.2rem' }}>Error: {error}</p>}
          {!isLoading && !error && assets.length === 0 && (
            <p style={{ textAlign: 'center', fontSize: '1.2rem' }}>No asset configurations found.</p>
          )}
          {!isLoading && !error && assets.map(asset => (
            <AssetCard key={asset.id} asset={asset} setGlobalMessage={setGlobalMessage} />
          ))}
        </div>
      </main>

      <footer className="app-footer">
        <p>&copy; {new Date().getFullYear()} Proof of Reserve System. All rights reserved.</p>
      </footer>
    </div>
  );
};

export default App;
