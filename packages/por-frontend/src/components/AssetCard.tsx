import React, { useState, useEffect, useCallback } from 'react';
import { ethers } from 'ethers';
import type { AssetConfig, ReserveDetailsOutput } from '../types';
import { getReserveDetails, submitSignature, API_BASE_URL } from '../services/api';
import { useWallet } from '../hooks/useWallet';
import { signProofTypedData, getValidUntil } from '../utils/signing';
import './AssetCard.css';
import ProofOfReserveArtifact from '../config/ProofOfReserve.json';

interface AssetCardProps {
  asset: AssetConfig;
  setGlobalMessage?: (msg: { type: 'success' | 'error', text: React.ReactNode } | null) => void;
}

// Helper function to truncate a string (e.g., Ethereum address or signature)
const truncateString = (str: string | undefined, startChars: number, endChars: number): string => {
  if (!str) return 'N/A';
  if (str.length <= startChars + endChars + 3) return str; // Don't truncate if it's already short
  return `${str.substring(0, startChars)}...${str.substring(str.length - endChars)}`;
};

function getFriendlyErrorMessage(error: any): string {
  if (!error) return 'Unknown error. Please try again.';
  const msg = typeof error === 'string' ? error : error.message || '';
  if (msg.includes('unknown account') || msg.includes('getAddress')) {
    return 'No wallet account found. Please connect your wallet and try again.';
  }
  if (msg.includes('user rejected') || msg.includes('User denied')) {
    return 'You rejected the transaction or signature request.';
  }
  if (msg.includes('No Ethereum provider')) {
    return 'No wallet provider found. Please install MetaMask or another wallet.';
  }
  if (msg.includes('UNSUPPORTED_OPERATION')) {
    return 'Wallet operation not supported. Please reconnect your wallet.';
  }
  if (msg.includes('Reserve balance below threshold')) {
    return 'The reserve wallet does not meet the minimum required balance. Please ensure the wallet holds at least the required threshold before verifying.';
  }
  return msg || 'Unknown error. Please try again.';
}

const AssetCard: React.FC<AssetCardProps> = ({ asset, setGlobalMessage }) => {
  const { account } = useWallet();
  const [details, setDetails] = useState<ReserveDetailsOutput | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState<boolean>(false);
  const [isSigningAndSubmitting, setIsSigningAndSubmitting] = useState<boolean>(false);
  const [success, setSuccess] = useState(false);

  const isReserveWallet = account?.toLowerCase() === asset.walletAddress.toLowerCase();

  const explorerBaseUrl = import.meta.env.VITE_EXPLORER_BASE_URL || 'https://explorer.apothem.network';
  const getExplorerAddressUrl = (address: string) => `${explorerBaseUrl}/address/${address}`;
  const getExplorerTxUrl = (txHash: string) => `${explorerBaseUrl}/tx/${txHash}`;

  const handleSignAndSubmit = async () => {
    if (!window.ethereum || !isReserveWallet || !account) return;

    setIsSigningAndSubmitting(true);
    if (setGlobalMessage) setGlobalMessage(null);

    try {
      const provider = new ethers.providers.Web3Provider(window.ethereum as any);
      const chainId = await provider.getNetwork().then(n => n.chainId);

      const contractAddress = import.meta.env.VITE_CONTRACT_ADDRESS;
      if (!contractAddress) {
        throw new Error('Contract address not configured');
      }

      //TODO: Move this in reserve config in backend. 
      const validUntil = getValidUntil(30); // 30 days validity

      console.log('Signing proof typed data for token:', asset.tokenAddress, 'wallet:', asset.walletAddress, 'validUntil:', validUntil);
      const signature = await signProofTypedData(
        provider,
        contractAddress,
        chainId,
        {
          token: asset.tokenAddress,
          wallet: asset.walletAddress,
          validUntil,
        }
      );

      console.log('Submitting signature for token:', asset.tokenAddress, 'wallet:', asset.walletAddress, 'signature:', signature, 'validUntil:', validUntil);
      await submitSignature(
        asset.tokenAddress,
        asset.walletAddress,
        signature,
        validUntil
      );

      console.log('Signature submitted and stored successfully!');

      if (setGlobalMessage) {
        setGlobalMessage({
          type: 'success',
          text: 'Signature submitted and stored successfully! You can now verify on-chain.'
        });
      }

      console.log('Fetching asset details...');

      setTimeout(() => {
        fetchAssetDetails();
      }, 2000);
    } catch (err) {
      console.error('Error signing and submitting:', err);
      if (setGlobalMessage) {
        setGlobalMessage({ type: 'error', text: getFriendlyErrorMessage(err) });
      }
    } finally {
      setIsSigningAndSubmitting(false);
    }
  };

  const fetchAssetDetails = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const data = await getReserveDetails(asset.tokenAddress, asset.walletAddress);
      console.log('Reserve details:', {
        target: data.target,
        targetType: typeof data.target,
        tokenAddress: asset.tokenAddress,
        symbol: data.symbol,
        rawData: data
      });
      setDetails(data);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('An unknown error occurred while fetching details.');
      }
      console.error(`Error fetching details for ${asset.displayName}:`, err);
    } finally {
      setIsLoading(false);
    }
  }, [asset.tokenAddress, asset.walletAddress, asset.displayName]);

  useEffect(() => {
    fetchAssetDetails();
  }, [fetchAssetDetails]);

  const handleVerify = async () => {
    try {
      setIsVerifying(true);
      if (setGlobalMessage) setGlobalMessage(null);
      setSuccess(false);

      // 1. Fetch the stored signature and validUntil from backend
      const sigResponse = await fetch(`${API_BASE_URL}/signature/${asset.tokenAddress}/${asset.walletAddress}`);
      if (!sigResponse.ok) {
        const data = await sigResponse.json();
        throw new Error(data.error || 'Failed to fetch stored signature');
      }
      const { signature, validUntil } = await sigResponse.json();

      // 2. Use ethers.js to call verifyProof on-chain
      if (!window.ethereum) throw new Error('No Ethereum provider found');
      const provider = new ethers.providers.Web3Provider(window.ethereum as any);
      const signer = provider.getSigner();
      const contractAddress = import.meta.env.VITE_CONTRACT_ADDRESS;
      const contract = new ethers.Contract(contractAddress, ProofOfReserveArtifact.abi, signer);

      const tx = await contract.verifyProof(
        asset.tokenAddress,
        asset.walletAddress,
        validUntil,
        signature
      );
      const getExplorerTxUrl = (txHash: string) => `https://explorer.apothem.network/tx/${txHash}`;
      setSuccess(true);
      if (setGlobalMessage) {
        setGlobalMessage({
          type: 'success',
          text: (
            <>
              Signature verified on-chain successfully.<br />
              <a
                href={getExplorerTxUrl(tx.hash)}
                target="_blank"
                rel="noopener noreferrer"
                style={{ color: '#1a7f37', fontWeight: 'bold', textDecoration: 'underline', display: 'inline-block', marginTop: 8 }}
              >
                View transaction: {tx.hash.slice(0, 8)}...{tx.hash.slice(-4)}
              </a>
            </>
          )
        });
      }

      // 3. Wait for the transaction to be mined
      const receipt = await tx.wait();
      if (receipt.status === 1) {
        // Update backend with tx hash
        await fetch(`${API_BASE_URL}/signature/txhash`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            token: asset.tokenAddress,
            wallet: asset.walletAddress,
            txHash: tx.hash,
          }),
        });
        if (setGlobalMessage) {
          setGlobalMessage({
            type: 'success',
            text: 'Signature verified on-chain successfully!'
          });
        }
        setTimeout(() => {
          fetchAssetDetails();
        }, 2000);
      } else {
        throw new Error('Transaction failed on-chain');
      }
    } catch (err) {
      console.error('Verification failed:', err);
      if (setGlobalMessage) {
        setGlobalMessage({ type: 'error', text: getFriendlyErrorMessage(err) });
      }
    } finally {
      setIsVerifying(false);
    }
  };

  const formatBalance = (balance?: string | number) => {
    if (balance === undefined || balance === null) return 'N/A';
    try {
      const balanceBigInt = BigInt(balance);
      if (balanceBigInt === BigInt(0)) return '0 ' + (details?.symbol || '');
      return (balanceBigInt / BigInt(10 ** 18)).toString() + ' ' + (details?.symbol || '');
    } catch (e) {
      console.error("Error formatting balance:", e);
      return 'Error';
    }
  };

  const formatTarget = (target?: string | number) => {
    if (target === undefined || target === null) return 'N/A';
    try {
      // For XDC (native token), the target is already in the correct unit
      if (asset.tokenAddress === '0x0000000000000000000000000000000000000000') {
        return target.toString() + ' ' + (details?.symbol || '');
      }

      // For CGO and other tokens, use the value as is since it's already in the correct unit
      return target.toString() + ' ' + (details?.symbol || '');
    } catch (e) {
      console.error("Error formatting target:", e);
      return 'Error';
    }
  };

  const calculateReserveRatio = () => {
    if (!details?.balance || !details?.target) return null;
    try {
      // Convert balance from wei to base unit
      const balance = BigInt(details.balance) / BigInt(10 ** 18);
      const target = BigInt(details.target);
      if (target === BigInt(0)) return null;
      return Number((balance * BigInt(100)) / target);
    } catch (e) {
      console.error("Error calculating reserve ratio:", e);
      return null;
    }
  };

  const getReserveRatioColor = (ratio: number) => {
    return ratio < 100 ? 'red' : 'green';
  };

  const getCardBorderStyle = (ratio: number | null) => {
    if (ratio === null) return {};
    return ratio < 100 ? {
      border: '2px solid #ff4444',
      boxShadow: '0 0 10px rgba(255, 68, 68, 0.3)'
    } : {};
  };

  const formatTimestamp = (timestamp?: string | number) => {
    if (timestamp === undefined || timestamp === null || Number(timestamp) === 0) return 'N/A';
    try {
      return new Date(Number(timestamp) * 1000).toLocaleString();
    } catch (e) {
      console.error("Error formatting timestamp:", e);
      return 'Error';
    }
  };

  return (
    <div className="asset-card" style={getCardBorderStyle(calculateReserveRatio())}>
      <div className="info-tooltip-container card-info-tooltip">
        <span className="info-icon">ⓘ</span>
        <div className="info-tooltip-banner">
          The amount on this webpage may not be updated immediately if there are movements in the wallet. For most accurate data, check the explorer by clicking the arrow on the right.
        </div>
      </div>

      {/* Status Messages */}
      {isLoading && <p className="status-message loading">Loading details...</p>}
      {error && !isLoading && <p className="status-message error">Error fetching details: {error}</p>}
      {!isLoading && details && !details.isConfigured && (
        <p className="status-message error" style={{marginTop: '10px'}}>
          This asset configuration is not found or not active in the smart contract.
        </p>
      )}

      <h3>
        {asset.logoUrl && <img src={asset.logoUrl} alt={`${asset.displayName} logo`} style={{ width: '24px', height: '24px', marginRight: '8px', verticalAlign: 'middle' }} />}
        {asset.displayName}
      </h3>

      {details && !isLoading && (
        <>
          <p><strong>Symbol:</strong> {details.symbol || 'N/A'}</p>
          <p><strong>Balance:</strong> {formatBalance(details.balance)}</p>
          <p className="token-address"><strong>Token Address:</strong> <a href={getExplorerAddressUrl(asset.tokenAddress)} target="_blank" rel="noopener noreferrer">{truncateString(asset.tokenAddress, 6, 4)}</a></p>
          <p className="wallet-address"><strong>Wallet Address:</strong> <a href={getExplorerAddressUrl(asset.walletAddress)} target="_blank" rel="noopener noreferrer">{truncateString(asset.walletAddress, 6, 4)}</a></p>
          <p><strong>Last Verified:</strong> {formatTimestamp(details.lastVerified)}
            {details.lastVerifiedTxHash && (
              <span className="tx-link"> (<a href={getExplorerTxUrl(details.lastVerifiedTxHash)} target="_blank" rel="noopener noreferrer">View Tx</a>)</span>
            )}
          </p>
          {details.target !== undefined && details.target !== null && (
            <>
              {calculateReserveRatio() !== null && (
                <p>
                  <strong>Reserve Ratio:</strong>{' '}
                  <span style={{ 
                    color: getReserveRatioColor(calculateReserveRatio()!), 
                    fontWeight: 'bold',
                    fontSize: '1.1em'
                  }}>
                    {calculateReserveRatio()}%
                  </span>
                </p>
              )}
            </>
          )}

          <div className="action-buttons" style={{ paddingBottom: isReserveWallet ? '20px' : '0' }}>
            {/* Verify button */}
            <button 
              onClick={handleVerify} 
              disabled={isVerifying || !details?.isConfigured || isLoading || !account}
              className={`verify-button ${success ? 'success' : ''}`}
            >
              {isVerifying ? 'Verifying...' : success ? 'Verified!' : 'Verify Now'}
            </button>

            {/* Sign & Submit button - only shown if this is the reserve wallet */}
            {isReserveWallet && details.isConfigured && (
              <button
                onClick={handleSignAndSubmit}
                disabled={isSigningAndSubmitting}
                className="sign-button"
              >
                {isSigningAndSubmitting ? 'Submitting...' : 'Update Signature'}
              </button>
            )}
          </div>
        </>
      )}
    </div>
  );
};

export default AssetCard;
