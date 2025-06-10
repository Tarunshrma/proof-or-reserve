import { useState, useEffect, useCallback } from 'react';
import { ethers } from 'ethers';
import type { WalletState } from '../types/web3';

declare global {
  interface Window {
    ethereum?: {
      isMetaMask?: boolean;
      request?: (args: { method: string; params?: any[] }) => Promise<any>;
      on: (event: string, handler: (params?: any) => void) => void;
      removeListener: (event: string, handler: (params?: any) => void) => void;
    };
  }
}

const initialState: WalletState = {
  account: null,
  chainId: null,
  isConnecting: false,
  isConnected: false,
  error: null,
};

export const useWallet = () => {
  const [state, setState] = useState<WalletState>(initialState);

  // Check if MetaMask is installed
  const checkMetaMask = useCallback(() => {
    if (!window.ethereum?.isMetaMask) {
      setState(prev => ({ ...prev, error: 'Please install MetaMask!' }));
      return false;
    }
    return true;
  }, []);

  // Connect to MetaMask
  const connect = useCallback(async () => {
    if (!checkMetaMask()) return;

    try {
      setState(prev => ({ ...prev, isConnecting: true, error: null }));
      
      const provider = new ethers.providers.Web3Provider(window.ethereum as any);
      const accounts = await provider.send('eth_requestAccounts', []);
      const network = await provider.getNetwork();
      
      setState(prev => ({
        ...prev,
        account: accounts[0],
        chainId: network.chainId,
        isConnected: true,
        isConnecting: false,
      }));
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: 'Failed to connect to MetaMask',
        isConnecting: false,
      }));
    }
  }, [checkMetaMask]);

  // Disconnect from MetaMask
  const disconnect = useCallback(async () => {
    try {
      // Clear local state
      setState(initialState);

      // If using a modern version of MetaMask, try to disconnect properly
      if (window.ethereum?.request) {
        try {
          await window.ethereum.request({
            method: 'wallet_revokePermissions',
            params: [{ eth_accounts: {} }]
          });
        } catch (e) {
          // Ignore errors, not all MetaMask versions support this
          console.log('Permission revocation not supported');
        }
      }

      // Force a page reload to ensure clean state
      window.location.reload();
    } catch (error) {
      console.error('Error disconnecting:', error);
      // Even if there's an error, we want to reset the state
      setState(initialState);
    }
  }, []);

  // Handle account changes
  const handleAccountsChanged = useCallback(async (accounts: string[]) => {
    console.log('Accounts changed:', accounts);
    if (accounts.length === 0) {
      // MetaMask is locked or the user has not connected any accounts
      setState(prev => ({
        ...prev,
        account: null,
        isConnected: false,
      }));
    } else {
      // Get current network to maintain chain info
      try {
        const provider = new ethers.providers.Web3Provider(window.ethereum as any);
        const network = await provider.getNetwork();
        setState(prev => ({
          ...prev,
          account: accounts[0],
          chainId: network.chainId,
          isConnected: true,
        }));
      } catch (error) {
        console.error('Error updating network info:', error);
        setState(prev => ({
          ...prev,
          account: accounts[0],
          isConnected: true,
        }));
      }
    }
  }, []);

  // Handle chain changes
  const handleChainChanged = useCallback(async (chainId: string) => {
    console.log('Chain changed:', chainId);
    // MetaMask recommends reloading the page on chain changes
    window.location.reload();
  }, []);

  // Handle disconnect
  const handleDisconnect = useCallback(() => {
    console.log('MetaMask disconnected');
    setState(initialState);
  }, []);

  // Set up event listeners
  useEffect(() => {
    if (!window.ethereum) return;

    // Initial setup
    const setupInitialState = async () => {
      try {
        const provider = new ethers.providers.Web3Provider(window.ethereum as any);
        const accounts = await provider.listAccounts();
        if (accounts.length > 0) {
          const network = await provider.getNetwork();
          setState({
            account: accounts[0],
            chainId: network.chainId,
            isConnected: true,
            isConnecting: false,
            error: null,
          });
        }
      } catch (error) {
        console.error('Error during initial setup:', error);
      }
    };

    // Set up event listeners
    window.ethereum.on('accountsChanged', handleAccountsChanged);
    window.ethereum.on('chainChanged', handleChainChanged);
    window.ethereum.on('disconnect', handleDisconnect);

    // Run initial setup
    setupInitialState();

    // Cleanup function
    return () => {
      if (!window.ethereum) return;
      window.ethereum.removeListener('accountsChanged', handleAccountsChanged);
      window.ethereum.removeListener('chainChanged', handleChainChanged);
      window.ethereum.removeListener('disconnect', handleDisconnect);
    };
  }, [handleAccountsChanged, handleChainChanged, handleDisconnect]);

  return {
    ...state,
    connect,
    disconnect,
  };
}; 