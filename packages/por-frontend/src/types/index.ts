export interface AssetConfig {
  id: string; // e.g., 'xdc', 'usdc'
  displayName: string; // e.g., 'XDC', 'USDC.e'
  tokenAddress: string;
  walletAddress: string;
  logoUrl?: string; // Optional: for displaying token logo
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
  token: string;
  wallet: string;
  verificationInitiated: boolean;
  onChainSuccess?: boolean;
  signatureUsed?: string;
  message: string;
  error?: string; // To capture any error messages during verification
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