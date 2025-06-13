import React from 'react';
import { useWallet } from '../hooks/useWallet';
import { truncateAddress } from '../utils/address';
import './WalletConnect.css';

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