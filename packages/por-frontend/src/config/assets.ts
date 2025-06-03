import type { AssetConfig } from '../types';

// IMPORTANT: Replace these placeholder addresses with your actual deployed token and reserve wallet addresses.
export const ASSETS_CONFIG: AssetConfig[] = [
  {
    id: 'wxdc',
    displayName: 'Wrapped XDC (Test)',
    // Example: Replace with your deployed TestToken (WXDC) address
    tokenAddress: '0x13eb437Cc117ec6D1f2e2cd02d023Ef00a69Bd1a', 
    walletAddress: '0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA',
    logoUrl: 'https://s2.coinmarketcap.com/static/img/coins/64x64/2634.png' // XDC logo
  },
  // {
  //   id: 'cgo',
  //   displayName: 'CGO',
  //   tokenAddress: '0xPLEASE_REPLACE_CGO_TOKEN_ADDRESS',
  //   walletAddress: '0xPLEASE_REPLACE_CGO_WALLET_ADDRESS',
  //   logoUrl: '' // Add a URL to CGO logo if available
  // },
  // {
  //   id: 'usdc',
  //   displayName: 'USDC.e',
  //   tokenAddress: '0xPLEASE_REPLACE_USDC_TOKEN_ADDRESS',
  //   walletAddress: '0xPLEASE_REPLACE_USDC_WALLET_ADDRESS',
  //   logoUrl: 'https://s2.coinmarketcap.com/static/img/coins/64x64/3408.png' // USDC logo
  // },
  // {
  //   id: 'fxd',
  //   displayName: 'FXD',
  //   tokenAddress: '0xPLEASE_REPLACE_FXD_TOKEN_ADDRESS',
  //   walletAddress: '0xPLEASE_REPLACE_FXD_WALLET_ADDRESS',
  //   logoUrl: '' // Add a URL to FXD logo if available
  // },
]; 