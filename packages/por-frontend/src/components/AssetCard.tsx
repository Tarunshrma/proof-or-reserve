import React, { useState, useEffect, useCallback } from 'react';
import { ethers, utils } from 'ethers';
import type { AssetConfig, ReserveDetailsOutput } from '../types';
import { getReserveDetails, submitSignature, API_BASE_URL } from '../services/api';
import { useWallet } from '../hooks/useWallet';
import { signProofTypedData, getValidUntil } from '../utils/signing';
import './AssetCard.css';
import ProofOfReserveArtifact from '../config/ProofOfReserve.json';

interface AssetCardProps {
  asset: AssetConfig;
}

const NATIVE_XDC_ADDRESS = "0x0000000000000000000000000000000000000000";

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
  return msg || 'Unknown error. Please try again.';
}

const AssetCard: React.FC<AssetCardProps> = ({ asset }) => {
  const { account } = useWallet();
  const [details, setDetails] = useState<ReserveDetailsOutput | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState<boolean>(false);
  const [isSigningAndSubmitting, setIsSigningAndSubmitting] = useState<boolean>(false);
  const [verificationStatus, setVerificationStatus] = useState<{ success: boolean; message: string; signature?: string; txHash?: string } | null>(null);
  const [verificationError, setVerificationError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const isReserveWallet = account?.toLowerCase() === asset.walletAddress.toLowerCase();

  const handleSignAndSubmit = async () => {
    if (!window.ethereum || !isReserveWallet || !account) return;

    setIsSigningAndSubmitting(true);
    setVerificationError(null);

    try {
      const provider = new ethers.providers.Web3Provider(window.ethereum as any);
      const chainId = await provider.getNetwork().then(n => n.chainId);

      const contractAddress = import.meta.env.VITE_CONTRACT_ADDRESS;
      if (!contractAddress) {
        throw new Error('Contract address not configured');
      }

      const validUntil = getValidUntil(30); // 30 days validity
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

      await submitSignature(
        asset.tokenAddress,
        asset.walletAddress,
        signature,
        validUntil
      );

      setVerificationStatus({
        success: true,
        message: 'Signature submitted and stored successfully! You can now verify on-chain.',
      });

      setTimeout(() => {
        fetchAssetDetails();
      }, 2000);
    } catch (err) {
      console.error('Error signing and submitting:', err);
      setVerificationError(getFriendlyErrorMessage(err));
    } finally {
      setIsSigningAndSubmitting(false);
    }
  };

  const fetchAssetDetails = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const data = await getReserveDetails(asset.tokenAddress, asset.walletAddress);
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

  // Auto-hide success messages
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>;
    if (verificationStatus && verificationStatus.success) {
      timer = setTimeout(() => {
        setVerificationStatus(null);
      }, 4000);
    }
    return () => {
      clearTimeout(timer);
    };
  }, [verificationStatus]);

  const handleVerify = async () => {
    try {
      setIsVerifying(true);
      setVerificationStatus(null);
      setVerificationError(null);
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
      setVerificationStatus({
        success: true,
        message: `Signature verified on-chain successfully!\n\nView transaction: <a href='${getExplorerTxUrl(tx.hash)}' target='_blank' rel='noopener noreferrer'>${tx.hash.slice(0, 10)}...</a>`,
        txHash: tx.hash,
      });

      // 3. Wait for the transaction to be mined
      const receipt = await tx.wait();
      if (receipt.status === 1) {
        setVerificationStatus({
          success: true,
          message: 'Signature verified on-chain successfully!',
          txHash: tx.hash,
        });
        setSuccess(true);
      } else {
        throw new Error('Transaction failed on-chain');
      }
    } catch (err) {
      console.error('Verification failed:', err);
      setVerificationStatus({
        success: false,
        message: `Verification failed: ${getFriendlyErrorMessage(err)}`,
      });
      setVerificationError(getFriendlyErrorMessage(err));
    } finally {
      setIsVerifying(false);
    }
  };

  const formatBalance = (balance?: string | number) => {
    if (balance === undefined || balance === null) return 'N/A';
    try {
      return (BigInt(balance) / BigInt(10 ** 18)).toString() + ' ' + (details?.symbol || '');
    } catch (e) {
      console.error("Error formatting balance:", e);
      return 'Error';
    }
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
    <div className="asset-card">
      <div className="info-tooltip-container card-info-tooltip">
        <span className="info-icon">ⓘ</span>
        <div className="info-tooltip-banner">
          The amount on this webpage may not be updated immediately if there are movements in the wallet. For most accurate data, check the explorer by clicking the arrow on the right.
        </div>
      </div>

      {/* Status Messages */}
      {isLoading && <p className="status-message loading">Loading details...</p>}
      {error && !isLoading && <p className="status-message error">Error fetching details: {error}</p>}
      {verificationStatus && (
        <div className={`status-message ${verificationStatus.success ? 'success' : 'error'}`}
             dangerouslySetInnerHTML={{ __html: verificationStatus.message }} />
      )}
      {verificationError && !verificationStatus && (
         <p className="status-message error">Verification Failed: {verificationError}</p>
      )}
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
          <p className="token-address"><strong>Token Address:</strong> {truncateString(asset.tokenAddress, 6, 4)}</p>
          <p className="wallet-address"><strong>Wallet Address:</strong> {truncateString(asset.walletAddress, 6, 4)}</p>
          <p><strong>Last Verified:</strong> {formatTimestamp(details.lastVerified)}</p>

          <div className="action-buttons">
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
