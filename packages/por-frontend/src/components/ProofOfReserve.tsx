import React, { useEffect, useState } from 'react';
import AssetCard from './AssetCard';
import { getConfiguredAssets } from '../services/api';
import type { AssetConfig } from '../types';

const ProofOfReserve: React.FC = () => {
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
        setAssets([]);
      } finally {
        setIsLoading(false);
      }
    };
    loadAssets();
  }, []);

  return (
    <div className="assets-grid">
      {isLoading && <p style={{ textAlign: 'center', fontSize: '1.2rem' }}>Loading asset configurations...</p>}
      {error && <p style={{ textAlign: 'center', color: 'var(--error-color)', fontSize: '1.2rem' }}>Error: {error}</p>}
      {!isLoading && !error && assets.length === 0 && (
        <p style={{ textAlign: 'center', fontSize: '1.2rem' }}>No asset configurations found.</p>
      )}
      {!isLoading && !error && assets.map(asset => (
        <AssetCard key={asset.id} asset={asset} />
      ))}
    </div>
  );
};

export default ProofOfReserve; 