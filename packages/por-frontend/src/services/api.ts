import type { ReserveDetails, VerificationResult, AssetConfig } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8082';

export async function getConfiguredAssets(): Promise<AssetConfig[]> {
  const response = await fetch(`${API_BASE_URL}/assets/configured`);
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ message: 'Failed to fetch configured assets and parse error response' }));
    throw new Error(errorData.message || `Failed to fetch configured assets: ${response.statusText}`);
  }
  return response.json();
}

export async function getReserveDetails(token: string, wallet: string): Promise<ReserveDetails> {
  const response = await fetch(`${API_BASE_URL}/reserve-details/${token}/${wallet}`);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to fetch reserve details');
  }
  return response.json();
}

export async function initiateOnchainVerification(
  tokenAddress: string,
  walletAddress: string
): Promise<VerificationResult> {
  const response = await fetch(`${API_BASE_URL}/initiate-onchain-verification`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ token: tokenAddress, wallet: walletAddress }),
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ message: "Failed to initiate verification" }));
    throw new Error(errorData.error || errorData.message || `HTTP error! status: ${response.status}`);
  }
  return response.json();
}

interface ContractConfig {
  address: string;
  chainId: number;
}

export async function getContractConfig(): Promise<ContractConfig> {
  const response = await fetch(`${API_BASE_URL}/contract/config`);
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ message: "Failed to fetch contract configuration" }));
    throw new Error(errorData.error || errorData.message || `Failed to fetch contract configuration: ${response.statusText}`);
  }
  return response.json();
}

export async function submitSignature(
  token: string,
  wallet: string,
  signature: string,
  validUntil: number
): Promise<VerificationResult> {
  const response = await fetch(`${API_BASE_URL}/submit-signature`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      token,
      wallet,
      signature,
      validUntil,
    }),
  });

  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to submit signature');
  }

  return response.json();
} 