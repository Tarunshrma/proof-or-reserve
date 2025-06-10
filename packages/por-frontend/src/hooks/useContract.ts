import { useState, useEffect } from 'react';
import { getContractConfig } from '../services/api';

interface ContractState {
  address: string | null;
  chainId: number | null;
  isLoading: boolean;
  error: string | null;
}

export function useContract() {
  const [state, setState] = useState<ContractState>({
    address: null,
    chainId: null,
    isLoading: true,
    error: null,
  });

  useEffect(() => {
    async function loadConfig() {
      try {
        const config = await getContractConfig();
        setState({
          address: config.address,
          chainId: config.chainId,
          isLoading: false,
          error: null,
        });
      } catch (err) {
        console.error('Failed to load contract config:', err);
        setState(prev => ({
          ...prev,
          isLoading: false,
          error: 'Failed to load contract configuration',
        }));
      }
    }

    loadConfig();
  }, []);

  return state;
} 