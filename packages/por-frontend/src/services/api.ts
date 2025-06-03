import type { ReserveDetailsResponse, VerificationResult } from '../types';

const API_BASE_URL = 'http://localhost:8082'; // As confirmed by the user

export async function getReserveDetails(
  tokenAddress: string,
  walletAddress: string
): Promise<ReserveDetailsResponse> {
  const response = await fetch(`${API_BASE_URL}/reserve-details/${tokenAddress}/${walletAddress}`);
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ message: "Failed to fetch reserve details" }));
    throw new Error(errorData.error || errorData.message || `HTTP error! status: ${response.status}`);
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