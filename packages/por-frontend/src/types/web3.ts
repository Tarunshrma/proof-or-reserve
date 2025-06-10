import { providers } from 'ethers';

declare global {
  interface Window {
    ethereum?: providers.ExternalProvider & {
      isMetaMask?: boolean;
      on: (event: string, handler: (params?: any) => void) => void;
      removeListener: (event: string, handler: (params?: any) => void) => void;
    };
  }
}

export interface WalletState {
  account: string | null;
  chainId: number | null;
  isConnecting: boolean;
  isConnected: boolean;
  error: string | null;
}

export interface ReserveInfo {
  isReserve: boolean;
  token: string;
  wallet: string;
} 