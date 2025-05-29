import { expect } from "chai";
import { ethers } from "hardhat";
import { Contract, Signer } from "ethers";
import { ProofOfReserve, TestToken } from "../typechain-types";

describe("ProofOfReserve", function () {
  let testToken: TestToken;
  let proofOfReserve: ProofOfReserve;
  let owner: Signer;
  let reserveWallet: Signer;
  let otherAccount: Signer;
  let reserveWalletAddress: string;

  beforeEach(async function () {
    // Get signers
    [owner, reserveWallet, otherAccount] = await ethers.getSigners();
    reserveWalletAddress = await reserveWallet.getAddress();

    // Deploy TestToken
    const TestToken = await ethers.getContractFactory("TestToken");
    testToken = await TestToken.deploy() as TestToken;
    await testToken.waitForDeployment();

    // Deploy ProofOfReserve
    const ProofOfReserve = await ethers.getContractFactory("ProofOfReserve");
    proofOfReserve = await ProofOfReserve.deploy() as ProofOfReserve;
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
        proofOfReserve.connect(otherAccount as any).transferOwnership(newOwnerAddress)
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
        proofOfReserve.connect(otherAccount as any).configureReserve(tokenAddress, reserveWalletAddress, signature)
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
      await proofOfReserve.connect(reserveWallet as any).deactivateReserve(tokenAddress, reserveWalletAddress);
      expect(await proofOfReserve.isReserveWallet(tokenAddress, reserveWalletAddress)).to.be.false;
    });

    it("Should prevent unauthorized accounts from deactivating reserves", async function () {
      const tokenAddress = await testToken.getAddress();
      await expect(
        proofOfReserve.connect(otherAccount as any).deactivateReserve(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Caller is not authorized");
    });

    it("Should verify proof of reserve with valid signature", async function () {
      const tokenAddress = await testToken.getAddress();
      
      // Create verification message
      const messageHash = ethers.keccak256(
        ethers.solidityPacked(
          ["address", "address", "address"],
          [tokenAddress, reserveWalletAddress, await proofOfReserve.getAddress()]
        )
      );
      
      // Sign the message
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      
      // Verify the proof
      const tx = await proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, signature);
      const receipt = await tx.wait();

      // Check event and verification status
      const event = receipt?.logs[0];
      expect(event?.eventName).to.equal("ProofVerified");
      expect(event?.args[0]).to.equal(tokenAddress);
      expect(event?.args[1]).to.equal(reserveWalletAddress);
      expect(event?.args[3]).to.be.true; // success

      // Check last verified timestamp
      const lastVerified = await proofOfReserve.getLastVerified(tokenAddress, reserveWalletAddress);
      expect(lastVerified).to.be.gt(0);
    });

    it("Should reject proof verification with invalid signature", async function () {
      const tokenAddress = await testToken.getAddress();
      
      // Create verification message
      const messageHash = ethers.keccak256(
        ethers.solidityPacked(
          ["address", "address", "address"],
          [tokenAddress, reserveWalletAddress, await proofOfReserve.getAddress()]
        )
      );
      
      // Sign with wrong wallet
      const signature = await otherAccount.signMessage(ethers.getBytes(messageHash));
      
      // Verify should return false
      const tx = await proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, signature);
      const receipt = await tx.wait();

      const event = receipt?.logs[0];
      expect(event?.eventName).to.equal("ProofVerified");
      expect(event?.args[3]).to.be.false; // success should be false
    });

    it("Should prevent operations on deactivated reserves", async function () {
      const tokenAddress = await testToken.getAddress();
      
      // Deactivate reserve
      await proofOfReserve.deactivateReserve(tokenAddress, reserveWalletAddress);
      
      // Try to get balance
      await expect(
        proofOfReserve.getReserveBalance(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Reserve not active");
      
      // Try to verify proof
      const messageHash = ethers.keccak256(
        ethers.solidityPacked(
          ["address", "address", "address"],
          [tokenAddress, reserveWalletAddress, await proofOfReserve.getAddress()]
        )
      );
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, signature)
      ).to.be.revertedWith("Reserve not active");
    });
  });
}); 