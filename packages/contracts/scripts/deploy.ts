import { ethers } from "hardhat";

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying contracts with the account:", deployer.address);

  console.log("Deploying ProofOfReserve contract...");
  const ProofOfReserve = await ethers.getContractFactory("ProofOfReserve");
  const proofOfReserve = await ProofOfReserve.deploy();

  console.log("Waiting for deployment...");
  await proofOfReserve.waitForDeployment();

  const address = await proofOfReserve.getAddress();
  
  // Convert the address to XDC format
  const xdcAddress = "xdc" + address.slice(2);
  
  console.log("ProofOfReserve deployed to:", address);
  console.log("XDC format address:", xdcAddress);
  
  console.log("\nVerification command:");
  console.log(`npx hardhat verify --network apothem ${address}`);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 