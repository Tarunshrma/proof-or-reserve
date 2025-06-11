export interface AssetConfig {
  id: string; // e.g., 'xdc', 'usdc'
  displayName: string; // e.g., 'XDC', 'USDC.e'
  tokenAddress: string;
  walletAddress: string;
  logoUrl?: string; // Optional: for displaying token logo
  isNativeToken?: boolean;  // Flag to indicate if this is native XDC token
  name: string;
  symbol: string;
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
  signatureUsed?: string;
  message?: string;
  error?: string;
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

// Ethereum provider type
declare global {
  interface Window {
    ethereum?: {
      isMetaMask?: boolean;
      request: (args: { method: string; params: any[] }) => Promise<any>;
      on: (event: string, handler: (params?: any) => void) => void;
      removeListener: (event: string, handler: (params?: any) => void) => void;
    };
  }
} 