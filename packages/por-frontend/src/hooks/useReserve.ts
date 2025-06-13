import { useState, useEffect } from 'react';
import type { ReserveInfo } from '../types/web3';
import { getReserveDetails } from '../services/api';
import { useContract } from './useContract';

export const useReserve = (walletAddress: string | null) => {
  const { address: contractAddress, isLoading: isLoadingContract, error: contractError } = useContract();
  const [reserveInfo, setReserveInfo] = useState<ReserveInfo>({
    isReserve: false,
    token: '',
    wallet: '',
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const checkReserveStatus = async () => {
      if (!walletAddress || !contractAddress) {
        setReserveInfo({ isReserve: false, token: '', wallet: '' });
        return;
      }

      setIsLoading(true);
      setError(null);

      try {
        // Use our backend API to get reserve details
        const details = await getReserveDetails(contractAddress, walletAddress);
        
        setReserveInfo({
          isReserve: details.isConfigured,
          token: details.tokenAddress ? details.tokenAddress.toLowerCase() : '',
          wallet: walletAddress.toLowerCase(),
        });
      } catch (err) {
        console.error('Error checking reserve status:', err);
        setError('Failed to check reserve status');
        setReserveInfo({ isReserve: false, token: '', wallet: '' });
      } finally {
        setIsLoading(false);
      }
    };

    if (contractError) {
      setError(contractError);
      return;
    }

    if (!isLoadingContract) {
      checkReserveStatus();
    }
  }, [walletAddress, contractAddress, contractError, isLoadingContract]);

  return { 
    reserveInfo, 
    isLoading: isLoading || isLoadingContract, 
    error: error || contractError 
  };
}; 