import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";
import * as dotenv from "dotenv";

dotenv.config();

if (!process.env.PRIVATE_KEY) {
  throw new Error("Please set your PRIVATE_KEY in a .env file");
}

// Make sure PRIVATE_KEY is available and properly formatted
const PRIVATE_KEY = process.env.PRIVATE_KEY.startsWith("0x") ? 
  process.env.PRIVATE_KEY.slice(2) : 
  process.env.PRIVATE_KEY;

console.log("Config loaded. Network: Apothem XDC Testnet (Chain ID: 51)");

const config: HardhatUserConfig = {
  solidity: {
    version: "0.8.20",
    settings: {
      optimizer: {
        enabled: true,
        runs: 200
      }
    }
  },
  networks: {
    apothem: {
      url: process.env.XDC_RPC || "https://rpc.apothem.network",
      chainId: 51,
      accounts: [PRIVATE_KEY]
    },
    xdc: {
      url: process.env.XDC_RPC || "https://earpc.xinfin.network",
      chainId: 50,
      accounts: [PRIVATE_KEY]
    }
  }
};

export default config; 