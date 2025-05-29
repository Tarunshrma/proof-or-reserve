import { ethers } from "hardhat";

async function main() {
  // Deploy TestToken
  const TestToken = await ethers.getContractFactory("TestToken");
  const testToken = await TestToken.deploy();
  await testToken.waitForDeployment();
  console.log("TestToken deployed to:", await testToken.getAddress());

  // Deploy ProofOfReserve
  const ProofOfReserve = await ethers.getContractFactory("ProofOfReserve");
  const proofOfReserve = await ProofOfReserve.deploy();
  await proofOfReserve.waitForDeployment();
  console.log("ProofOfReserve deployed to:", await proofOfReserve.getAddress());
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 