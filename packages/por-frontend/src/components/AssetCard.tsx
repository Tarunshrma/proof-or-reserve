import React, { useState, useEffect, useCallback } from 'react';
import type { AssetConfig, ReserveDetailsOutput, VerificationResult } from '../types';
import { getReserveDetails, initiateOnchainVerification } from '../services/api';

interface AssetCardProps {
  asset: AssetConfig;
}

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
        // Add a small delay before re-fetching to allow blockchain state to propagate
        setTimeout(() => {
          fetchAssetDetails();
        }, 2000); // 2-second delay
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
      <h3>
        {asset.logoUrl && <img src={asset.logoUrl} alt={`${asset.displayName} logo`} style={{ width: '24px', height: '24px', marginRight: '8px', verticalAlign: 'middle' }} />}
        {asset.displayName}
      </h3>

      {isLoading && <p className="status-message loading">Loading details...</p>}
      {error && !isLoading && <p className="status-message error">Error: {error}</p>}
      
      {details && !isLoading && (
        <>
          <p><strong>Symbol:</strong> {details.symbol || 'N/A'}</p>
          <p><strong>Balance:</strong> {formatBalance(details.balance)}</p>
          <p className="token-address"><strong>Token Address:</strong> {asset.tokenAddress}</p>
          <p className="wallet-address"><strong>Wallet Address:</strong> {asset.walletAddress}</p>
          <p><strong>Last Verified:</strong> {formatTimestamp(details.lastVerified)}</p>
        </>
      )}

      <button onClick={handleVerify} disabled={isVerifying || !details?.isConfigured || isLoading}>
        {isVerifying ? 'Verifying...' : (details?.isConfigured === false ? 'Not Configured' : (isLoading ? 'Loading Data...' : 'Verify On-Chain'))}
      </button>

      {verificationStatus && (
        <div className={`status-message ${verificationStatus.success ? 'success' : 'error'}`}>
          <p>{verificationStatus.message}</p>
          {verificationStatus.signature && <p style={{ fontSize: '0.75rem', wordBreak: 'break-all' }}><strong>Signature:</strong> {verificationStatus.signature}</p>}
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
    </div>
  );
};

export default AssetCard;
