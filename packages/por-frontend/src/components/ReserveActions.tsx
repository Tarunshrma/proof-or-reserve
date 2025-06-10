import React, { useState } from 'react';
import { ethers, utils } from 'ethers';
import { useReserve } from '../hooks/useReserve';
import { submitSignature } from '../services/api';

interface Props {
  walletAddress: string;
}

export const ReserveActions: React.FC<Props> = ({ walletAddress }) => {
  const { reserveInfo, isLoading, error } = useReserve(walletAddress);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const generatePayload = (token: string, wallet: string): string => {
    // Following the same format as your Go backend:
    // prefix + token address + wallet address
    const prefix = utils.toUtf8Bytes('ProofOfReserve:');
    const tokenAddr = utils.getAddress(token); // Normalize address
    const walletAddr = utils.getAddress(wallet); // Normalize address
    
    // Concatenate the bytes: prefix + token + wallet
    const payload = utils.concat([
      prefix,
      utils.arrayify(tokenAddr),
      utils.arrayify(walletAddr)
    ]);

    return utils.hexlify(payload);
  };

  const generateAndSubmitSignature = async () => {
    if (!window.ethereum || !reserveInfo.isReserve) return;

    setIsSubmitting(true);
    setSubmitError(null);

    try {
      // Generate payload locally
      const payload = generatePayload(reserveInfo.token, walletAddress);

      // Sign the payload with MetaMask
      const provider = new ethers.providers.Web3Provider(window.ethereum);
      const signer = provider.getSigner();
      const signature = await signer.signMessage(utils.arrayify(payload));

      // Submit the signature using our API service
      await submitSignature({
        token: reserveInfo.token,
        wallet: walletAddress,
        signature,
        payload,
      });

      // Success!
      alert('Signature submitted successfully!');
    } catch (err) {
      console.error('Error submitting signature:', err);
      setSubmitError(err instanceof Error ? err.message : 'Failed to submit signature');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoading) {
    return <div>Checking reserve status...</div>;
  }

  if (error) {
    return <div className="error-message">{error}</div>;
  }

  // Debug output
  console.log('Reserve Info:', reserveInfo);

  return (
    <div className="reserve-actions">
      <h3>Reserve Wallet Actions</h3>
      <div className="info">
        <p>Token: {reserveInfo.token}</p>
        <p>Wallet: {walletAddress}</p>
        <p>Is Reserve: {reserveInfo.isReserve ? 'Yes' : 'No'}</p>
      </div>
      
      {submitError && (
        <div className="error-message">
          {submitError}
        </div>
      )}

      {reserveInfo.isReserve && (
        <button
          onClick={generateAndSubmitSignature}
          disabled={isSubmitting}
          className="sign-button"
        >
          {isSubmitting ? 'Submitting...' : 'Generate & Submit Signature'}
        </button>
      )}
    </div>
  );
}; 