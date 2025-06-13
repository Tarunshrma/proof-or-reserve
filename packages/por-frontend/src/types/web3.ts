import type { ExternalProvider } from '@ethersproject/providers';

declare global {
  interface Window {
    ethereum?: ExternalProvider;
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