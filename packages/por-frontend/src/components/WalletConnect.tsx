import React from 'react';
import { useWallet } from '../hooks/useWallet';
import { truncateAddress } from '../utils/address';
import './WalletConnect.css';

const WalletIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path d="M21 7V17C21 18.1046 20.1046 19 19 19H5C3.89543 19 3 18.1046 3 17V7C3 5.89543 3.89543 5 5 5H19C20.1046 5 21 5.89543 21 7Z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M3 7L12 13L21 7" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
  </svg>
);

const MetaMaskIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path d="M20.61 4L13.03 9.08L14.4 5.85L20.61 4Z" fill="#E2761B" stroke="#E2761B" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M3.38 4L10.9 9.15L9.6 5.85L3.38 4Z" fill="#E4761B" stroke="#E4761B" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M17.95 15.92L16.14 18.77L19.87 19.88L21 16.01L17.95 15.92Z" fill="#E4761B" stroke="#E4761B" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M3.01001 16.01L4.13001 19.88L7.86001 18.77L6.05001 15.92L3.01001 16.01Z" fill="#E4761B" stroke="#E4761B" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M7.86001 18.77L10.9 13.69L9.60001 15.92L7.86001 18.77Z" fill="#E4761B" stroke="#E4761B" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M16.14 18.77L13.03 13.69L14.4 15.92L16.14 18.77Z" fill="#E4761B" stroke="#E4761B" strokeLinecap="round" strokeLinejoin="round"/>
  </svg>
);

export const WalletConnect: React.FC = () => {
  const { isConnected, account, chainId, connect, disconnect } = useWallet();

  const getNetworkName = (chainId: number | null) => {
    switch (chainId) {
      case 50:
        return 'XDC Mainnet';
      case 51:
        return 'XDC Testnet';
      default:
        return 'Unknown Network';
    }
  };

  return (
    <div className="wallet-connect-container">
      {isConnected ? (
        <div className="wallet-info">
          <div className="network-badge">
            <span className="network-indicator"></span>
            {getNetworkName(chainId)}
          </div>
          <div className="address-container">
            <span className="address">{truncateAddress(account || '')}</span>
            <button onClick={disconnect} className="disconnect-button">
              Disconnect
            </button>
          </div>
        </div>
      ) : (
        <button onClick={connect} className="connect-button">
          Connect Wallet
        </button>
      )}
    </div>
  );
}; 