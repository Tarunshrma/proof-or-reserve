import type { ExternalProvider } from '@ethersproject/providers';

declare global {
  interface Window {
    ethereum?: ExternalProvider;
  }
}

export interface AssetConfig {
  address: string;
  tokenAddress: string;
  walletAddress: string;
  symbol: string;
  name: string;
  decimals: number;
  icon?: string;
  isNativeToken?: boolean;
  displayName?: string;
  logoUrl?: string;
}

// Matches the backend's service.ReserveDetailsOutput
export interface ReserveDetailsOutput {
  isConfigured: boolean;
  name: string;
  symbol: string;
  balance: string; // Backend sends as string after big.Int conversion
  lastVerified: string; // Backend sends as string
}

// Matches the relevant parts of the backend's verification response
export interface VerificationResult {
  success: boolean;
  signatureUsed?: string;
  message?: string;
}

// For the /reserve-details/:token/:wallet endpoint response structure
export interface ReserveDetailsResponse {
  tokenAddress: string;
  walletAddress: string;
  isConfigured: boolean;
  name: string;
  symbol: string;
  balance: string;
  lastVerified: string;
}

export interface ReserveDetails {
  isConfigured: boolean;
  name: string;
  symbol: string;
  balance: string;
  lastVerified: string;
} 