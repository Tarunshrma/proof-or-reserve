import React, { useState, useEffect, useCallback } from 'react';
import type { AssetConfig, ReserveDetailsOutput, VerificationResult } from '../types';
import { getReserveDetails, initiateOnchainVerification } from '../services/api';

interface AssetCardProps {
  asset: AssetConfig;
}

// Helper function to truncate a string (e.g., Ethereum address or signature)
const truncateString = (str: string | undefined, startChars: number, endChars: number): string => {
  if (!str) return 'N/A';
  if (str.length <= startChars + endChars + 3) return str; // Don't truncate if it's already short
  return `${str.substring(0, startChars)}...${str.substring(str.length - endChars)}`;
};

const AssetCard: React.FC<AssetCardProps> = ({ asset }) => {
  const [details, setDetails] = useState<ReserveDetailsOutput | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState<boolean>(false);
  const [verificationStatus, setVerificationStatus] = useState<{ success: boolean; message: string; signature?: string } | null>(null);
  const [verificationError, setVerificationError] = useState<string | null>(null);

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

  // useEffect to auto-hide success messages
  useEffect(() => {
    let timer: NodeJS.Timeout;
    if (verificationStatus && verificationStatus.success) {
      timer = setTimeout(() => {
        setVerificationStatus(null);
      }, 4000); // Auto-hide after 4 seconds
    }
    return () => {
      clearTimeout(timer); // Cleanup timer on component unmount or if status changes
    };
  }, [verificationStatus]);

  const handleVerify = async () => {
    try {
      setIsVerifying(true);
      setVerificationStatus(null);
      setVerificationError(null);
      const result: VerificationResult = await initiateOnchainVerification(asset.tokenAddress, asset.walletAddress);
      setVerificationStatus({
        success: result.onChainSuccess || false,
        message: result.message || (result.onChainSuccess ? 'Verification Successful' : 'Verification Failed'),
        signature: result.signatureUsed,
      });

      if (result.onChainSuccess) {
        setTimeout(() => {
          fetchAssetDetails();
        }, 2000);
      }
    } catch (err) {
      let errorMessage = 'An unknown error occurred during verification.';
      if (err instanceof Error) {
        errorMessage = err.message;
      }
      setVerificationError(errorMessage);
      setVerificationStatus({ success: false, message: 'Verification Failed on client-side' });
      console.error(`Error verifying ${asset.displayName}:`, err);
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

      {/* Moved Status Messages START */}
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
      {/* Moved Status Messages END */}

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
        </>
      )}

      <button onClick={handleVerify} disabled={isVerifying || !details?.isConfigured || isLoading}>
        {isVerifying ? 'Verifying...' : (details?.isConfigured === false ? 'Not Configured' : (isLoading ? 'Loading Data...' : 'Verify On-Chain'))}
      </button>
    </div>
  );
};

export default AssetCard;
