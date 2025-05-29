import { expect } from "chai";
import { ethers } from "hardhat";
import { Contract } from "ethers";

describe("ProofOfReserve", function () {
  let testToken: Contract;
  let proofOfReserve: Contract;
  let owner: any;
  let reserveWallet: any;
  let reserveWalletAddress: string;

  beforeEach(async function () {
    // Get signers
    [owner, reserveWallet] = await ethers.getSigners();
    reserveWalletAddress = await reserveWallet.getAddress();

    // Deploy TestToken
    const TestToken = await ethers.getContractFactory("TestToken");
    testToken = await TestToken.deploy();
    await testToken.waitForDeployment();

    // Deploy ProofOfReserve
    const ProofOfReserve = await ethers.getContractFactory("ProofOfReserve");
    proofOfReserve = await ProofOfReserve.deploy();
    await proofOfReserve.waitForDeployment();

    // Transfer some tokens to reserve wallet
    const amount = ethers.parseEther("1000");
    await testToken.transfer(reserveWalletAddress, amount);
  });

  it("Should configure reserve wallet with valid signature", async function () {
    const tokenAddress = await testToken.getAddress();
    
    // Get message hash
    const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress);
    
    // Sign the message hash
    const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
    
    // Configure reserve wallet
    await proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress, signature);
    
    // Check if wallet is configured
    expect(await proofOfReserve.isReserveWallet(tokenAddress, reserveWalletAddress)).to.be.true;
    
    // Check balance
    const balance = await proofOfReserve.getReserveBalance(tokenAddress, reserveWalletAddress);
    expect(balance).to.equal(ethers.parseEther("1000"));
  });

  it("Should reject configuration with invalid signature", async function () {
    const tokenAddress = await testToken.getAddress();
    
    // Get message hash
    const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress);
    
    // Sign with wrong wallet (owner instead of reserveWallet)
    const signature = await owner.signMessage(ethers.getBytes(messageHash));
    
    // Try to configure reserve wallet with wrong signature
    await expect(
      proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress, signature)
    ).to.be.revertedWith("Invalid wallet signature");
  });

  it("Should submit and verify proof of reserve", async function () {
    const tokenAddress = await testToken.getAddress();
    
    // Get message hash
    const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress);
    const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
    
    // Configure reserve wallet
    await proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress, signature);

    // Submit proof
    const tx = await proofOfReserve.submitProof(tokenAddress, reserveWalletAddress);
    const receipt = await tx.wait();

    // Check event emission
    const event = receipt?.logs[0];
    expect(event?.eventName).to.equal("ProofSubmitted");
    expect(event?.args[0]).to.equal(tokenAddress);
    expect(event?.args[1]).to.equal(reserveWalletAddress);
    expect(event?.args[2]).to.equal(ethers.parseEther("1000")); // balance
  });

  it("Should deactivate reserve wallet", async function () {
    const tokenAddress = await testToken.getAddress();
    
    // Get message hash
    const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress);
    const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
    
    // Configure reserve wallet
    await proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress, signature);
    
    // Deactivate reserve wallet
    await proofOfReserve.deactivateReserve(tokenAddress, reserveWalletAddress);
    
    // Check if wallet is deactivated
    expect(await proofOfReserve.isReserveWallet(tokenAddress, reserveWalletAddress)).to.be.false;
    
    // Try to get balance (should fail)
    await expect(
      proofOfReserve.getReserveBalance(tokenAddress, reserveWalletAddress)
    ).to.be.revertedWith("Reserve not active");
  });
}); 