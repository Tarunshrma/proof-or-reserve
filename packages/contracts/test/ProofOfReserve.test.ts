import { expect } from "chai";
import { ethers } from "hardhat";
import { Contract } from "ethers";

describe("ProofOfReserve", function () {
  let testToken: Contract;
  let proofOfReserve: Contract;
  let owner: any;
  let reserveWallet: any;
  let otherAccount: any;
  let reserveWalletAddress: string;

  beforeEach(async function () {
    // Get signers
    [owner, reserveWallet, otherAccount] = await ethers.getSigners();
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

  describe("Ownership", function () {
    it("Should set the right owner", async function () {
      expect(await proofOfReserve.owner()).to.equal(await owner.getAddress());
    });

    it("Should allow ownership transfer", async function () {
      const newOwnerAddress = await otherAccount.getAddress();
      await proofOfReserve.transferOwnership(newOwnerAddress);
      expect(await proofOfReserve.owner()).to.equal(newOwnerAddress);
    });

    it("Should prevent non-owners from transferring ownership", async function () {
      const newOwnerAddress = await otherAccount.getAddress();
      await expect(
        proofOfReserve.connect(otherAccount).transferOwnership(newOwnerAddress)
      ).to.be.revertedWith("Caller is not the owner");
    });
  });

  describe("Reserve Configuration", function () {
    it("Should allow owner to configure reserve wallet with valid signature", async function () {
      const tokenAddress = await testToken.getAddress();
      
      // Get message hash
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress);
      
      // Sign the message hash
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      
      // Configure reserve wallet
      await proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress, signature);
      
      // Check if wallet is configured
      expect(await proofOfReserve.isReserveWallet(tokenAddress, reserveWalletAddress)).to.be.true;
    });

    it("Should prevent non-owners from configuring reserve wallets", async function () {
      const tokenAddress = await testToken.getAddress();
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress);
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        proofOfReserve.connect(otherAccount).configureReserve(tokenAddress, reserveWalletAddress, signature)
      ).to.be.revertedWith("Caller is not the owner");
    });

    it("Should reject configuration with invalid signature", async function () {
      const tokenAddress = await testToken.getAddress();
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress);
      const signature = await owner.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress, signature)
      ).to.be.revertedWith("Invalid wallet signature");
    });
  });

  describe("Reserve Management", function () {
    beforeEach(async function () {
      const tokenAddress = await testToken.getAddress();
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress);
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      await proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress, signature);
    });

    it("Should allow owner to deactivate reserve", async function () {
      const tokenAddress = await testToken.getAddress();
      await proofOfReserve.deactivateReserve(tokenAddress, reserveWalletAddress);
      expect(await proofOfReserve.isReserveWallet(tokenAddress, reserveWalletAddress)).to.be.false;
    });

    it("Should allow wallet to deactivate its own reserve", async function () {
      const tokenAddress = await testToken.getAddress();
      await proofOfReserve.connect(reserveWallet).deactivateReserve(tokenAddress, reserveWalletAddress);
      expect(await proofOfReserve.isReserveWallet(tokenAddress, reserveWalletAddress)).to.be.false;
    });

    it("Should prevent unauthorized accounts from deactivating reserves", async function () {
      const tokenAddress = await testToken.getAddress();
      await expect(
        proofOfReserve.connect(otherAccount).deactivateReserve(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Caller is not authorized");
    });

    it("Should submit and verify proof of reserve", async function () {
      const tokenAddress = await testToken.getAddress();
      const tx = await proofOfReserve.submitProof(tokenAddress, reserveWalletAddress);
      const receipt = await tx.wait();

      const event = receipt?.logs[0];
      expect(event?.eventName).to.equal("ProofSubmitted");
      expect(event?.args[0]).to.equal(tokenAddress);
      expect(event?.args[1]).to.equal(reserveWalletAddress);
      expect(event?.args[2]).to.equal(ethers.parseEther("1000")); // balance
    });

    it("Should prevent operations on deactivated reserves", async function () {
      const tokenAddress = await testToken.getAddress();
      
      // Deactivate reserve
      await proofOfReserve.deactivateReserve(tokenAddress, reserveWalletAddress);
      
      // Try to get balance
      await expect(
        proofOfReserve.getReserveBalance(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Reserve not active");
      
      // Try to submit proof
      await expect(
        proofOfReserve.submitProof(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Reserve not active");
    });
  });
}); 