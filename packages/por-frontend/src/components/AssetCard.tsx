import React, { useState, useEffect, useCallback } from 'react';
import { ethers, utils } from 'ethers';
import type { ExternalProvider } from '@ethersproject/providers';
import type { AssetConfig, ReserveDetailsOutput, VerificationResult } from '../types';
import { getReserveDetails, initiateOnchainVerification, submitSignature } from '../services/api';
import { useWallet } from '../hooks/useWallet';
import { signProofTypedData, getValidUntil } from '../utils/signing';
import './AssetCard.css';

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

declare global {
  interface Window {
    ethereum?: {
      isMetaMask?: boolean;
      request: (args: { method: string; params: any[]; }) => Promise<any>;
      on: (event: string, handler: (params?: any) => void) => void;
      removeListener: (event: string, handler: (params?: any) => void) => void;
    };
  }
}

const AssetCard: React.FC<AssetCardProps> = ({ asset }) => {
  const { account } = useWallet();
  const [details, setDetails] = useState<ReserveDetailsOutput | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState<boolean>(false);
  const [isSigningAndSubmitting, setIsSigningAndSubmitting] = useState<boolean>(false);
  const [verificationStatus, setVerificationStatus] = useState<{ success: boolean; message: string; signature?: string } | null>(null);
  const [verificationError, setVerificationError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const isReserveWallet = account?.toLowerCase() === asset.walletAddress.toLowerCase();

  const generatePayload = (token: string, wallet: string): string => {
    const prefix = utils.toUtf8Bytes('ProofOfReserve:');
    // Use address(0) for native XDC token
    const tokenAddr = asset.isNativeToken ? NATIVE_XDC_ADDRESS : utils.getAddress(token);
    const walletAddr = utils.getAddress(wallet);
    
    // Convert addresses to bytes without 0x prefix
    const tokenBytes = utils.arrayify(tokenAddr);
    const walletBytes = utils.arrayify(walletAddr);
    
    // Ensure token address is padded to 20 bytes
    const paddedTokenBytes = new Uint8Array(20);
    paddedTokenBytes.set(tokenBytes, paddedTokenBytes.length - tokenBytes.length);
    
    const payload = utils.concat([
      prefix,
      paddedTokenBytes,
      walletBytes
    ]);

    const hexPayload = utils.hexlify(payload).replace('0x', '');
    console.log('Generated payload:', {
      prefix: utils.hexlify(prefix),
      tokenAddr,
      walletAddr,
      hexPayload,
      expectedPayload: '50726f6f664f66526573657276653a0000000000000000000000000000000000000000fa4e7cfcdb5c280f887e8b0dbbfa4bdea19c3f7e'
    });

    return hexPayload;
  };

  const handleSignAndSubmit = async () => {
    if (!window.ethereum || !isReserveWallet || !account) return;

    setIsSigningAndSubmitting(true);
    setVerificationError(null);

    try {
      // Generate payload
      const tokenAddress = asset.isNativeToken ? NATIVE_XDC_ADDRESS : asset.tokenAddress;
      const payload = generatePayload(tokenAddress, asset.walletAddress);
      console.log('Generated Payload:', payload);

      // Hash the payload first using keccak256(abi.encodePacked())
      const payloadBytes = utils.arrayify('0x' + payload);
      const messageHash = utils.keccak256(payloadBytes);
      console.log('Message Hash:', messageHash);

      // Use personal_sign which is the recommended way to sign messages
      const signature = await window.ethereum.request({
        method: 'personal_sign',
        params: [messageHash, account],
      });
      console.log('Raw Signature:', signature);

      // Set validUntil to 30 days from now
      const validUntil = Math.floor(Date.now() / 1000) + (30 * 24 * 60 * 60);

      // Submit to backend
      await submitSignature({
        token: tokenAddress,
        wallet: asset.walletAddress,
        signature,
        payload, // Send payload without 0x prefix
        validUntil,
      });

      setVerificationStatus({
        success: true,
        message: 'Signature submitted successfully',
      });

      // Refresh details after successful submission
      setTimeout(() => {
        fetchAssetDetails();
      }, 2000);
    } catch (err) {
      console.error('Error signing and submitting:', err);
      setVerificationError(err instanceof Error ? err.message : 'Failed to sign and submit');
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
    let timer: NodeJS.Timeout;
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

      // Get the provider and signer
      const provider = new ethers.providers.Web3Provider(window.ethereum);
      const chainId = await provider.getNetwork().then(n => n.chainId);

      // Get the contract address from environment
      const contractAddress = process.env.REACT_APP_CONTRACT_ADDRESS;
      if (!contractAddress) {
        throw new Error('Contract address not configured');
      }

      // Sign the proof using EIP-712
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

      // Submit the proof to the backend
      const response = await fetch(`${process.env.REACT_APP_API_URL}/submit-signature`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          token: asset.tokenAddress,
          wallet: asset.walletAddress,
          signature,
          validUntil,
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to verify proof');
      }

      setSuccess(true);
    } catch (err) {
      console.error('Verification failed:', err);
      setVerificationError(err instanceof Error ? err.message : 'Failed to verify proof');
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
        <div className={`status-message ${verificationStatus.success ? 'success' : 'error'}`}>
          <p>{verificationStatus.message}</p>
        </div>
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
              disabled={isVerifying || !details?.isConfigured || isLoading}
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
