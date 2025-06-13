import { ethers } from 'ethers';

export interface ProofTypedData {
  token: string;
  wallet: string;
  validUntil: number;
}

const EIP712_DOMAIN = {
  name: 'ProofOfReserve',
  version: '1',
};

const TYPES = {
  Proof: [
    { name: 'token', type: 'address' },
    { name: 'wallet', type: 'address' },
    { name: 'validUntil', type: 'uint256' },
  ],
};

export async function signProofTypedData(
  provider: ethers.providers.Web3Provider,
  contractAddress: string,
  chainId: number,
  data: ProofTypedData
): Promise<string> {
  const domain = {
    ...EIP712_DOMAIN,
    chainId,
    verifyingContract: contractAddress,
  };

  const signer = provider.getSigner();
  const signature = await provider.send('eth_signTypedData_v4', [
    await signer.getAddress(),
    JSON.stringify({
      types: {
        EIP712Domain: [
          { name: 'name', type: 'string' },
          { name: 'version', type: 'string' },
          { name: 'chainId', type: 'uint256' },
          { name: 'verifyingContract', type: 'address' },
        ],
        ...TYPES,
      },
      primaryType: 'Proof',
      domain,
      message: data,
    }),
  ]);

  return signature;
}

export function getValidUntil(durationInDays: number = 30): number {
  return Math.floor(Date.now() / 1000) + (durationInDays * 24 * 60 * 60);
} 