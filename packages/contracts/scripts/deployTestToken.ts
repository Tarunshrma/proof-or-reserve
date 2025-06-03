import { ethers, network } from "hardhat";
import hre from "hardhat";

async function main() {
  // Get the deployer account
  const [deployer] = await ethers.getSigners();
  const deployerAddress = await deployer.getAddress();
  console.log(`\nDeploying contracts with the account: ${deployerAddress}`);

  // Get the contract factory for TestToken
  const TestTokenFactory = await ethers.getContractFactory("TestToken");

  console.log("Deploying TestToken contract...");
  const testToken = await TestTokenFactory.connect(deployer).deploy();
  
  // Wait for the deployment to complete
  await testToken.waitForDeployment();
  const testTokenAddress = await testToken.getAddress();

  console.log(`TestToken deployed to: ${testTokenAddress}`);

  // XDC Network uses a different address prefix, convert if necessary
  if (network.name === "apothem" || network.name === "xdc") {
    console.log(`XDC format address: xdc${testTokenAddress.substring(2)}`);
  }

  // Optional: Verify on block explorer (if network supports it)
  if (network.config.chainId && network.config.chainId !== 31337 && process.env.ETHERSCAN_API_KEY) {
    console.log("\nVerifying contract on Etherscan...");
    await hre.run("verify:verify", {
      address: testTokenAddress,
      constructorArguments: [], // No constructor arguments for TestToken
    });
    console.log("Contract verified successfully!");
  } else if (network.config.chainId === 31337) {
    console.log("\nSkipping verification for local Hardhat network.");
  } else {
    console.log(`\nVerification command (if needed):\nnpx hardhat verify --network ${network.name} ${testTokenAddress}`);
  }

}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 