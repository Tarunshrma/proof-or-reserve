import { expect } from "chai";
import { ethers } from "hardhat";
import { ProofOfReserve, TestToken } from "../typechain-types";
import { time } from "@nomicfoundation/hardhat-network-helpers";
import { SignerWithAddress } from "@nomicfoundation/hardhat-ethers/signers";

describe("ProofOfReserve", function () {
  let testToken: TestToken;
  let proofOfReserve: ProofOfReserve;
  let owner: SignerWithAddress;
  let reserveWallet: SignerWithAddress;
  let otherAccount: SignerWithAddress;
  let reserveWalletAddress: string;

  beforeEach(async function () {
    // Get signers
    [owner, reserveWallet, otherAccount] = await ethers.getSigners();
    reserveWalletAddress = await reserveWallet.getAddress();

    // Deploy TestToken
    const TestToken = await ethers.getContractFactory("TestToken");
    const testTokenContract = await TestToken.deploy();
    testToken = (await testTokenContract.waitForDeployment()) as unknown as TestToken;

    // Deploy ProofOfReserve
    const ProofOfReserve = await ethers.getContractFactory("ProofOfReserve");
    const proofOfReserveContract = await ProofOfReserve.deploy();
    proofOfReserve = (await proofOfReserveContract.waitForDeployment()) as unknown as ProofOfReserve;

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
    it("Should allow owner to configure reserve wallet", async function () {
      const tokenAddress = await testToken.getAddress();
      await proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress);
      expect(await proofOfReserve.isReserveWallet(tokenAddress, reserveWalletAddress)).to.be.true;
    });

    it("Should prevent non-owners from configuring reserve wallets", async function () {
      const tokenAddress = await testToken.getAddress();
      await expect(
        proofOfReserve.connect(otherAccount).configureReserve(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Caller is not the owner");
    });

    it("Should reject zero addresses", async function () {
      const tokenAddress = await testToken.getAddress();
      const zeroAddress = ethers.ZeroAddress;
      
      await expect(
        proofOfReserve.configureReserve(zeroAddress, reserveWalletAddress)
      ).to.be.revertedWith("Invalid token address");

      await expect(
        proofOfReserve.configureReserve(tokenAddress, zeroAddress)
      ).to.be.revertedWith("Invalid wallet address");
    });
  });

  describe("Reserve Management", function () {
    beforeEach(async function () {
      const tokenAddress = await testToken.getAddress();
      await proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress);
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

    it("Should verify proof of reserve with valid signature", async function () {
      const tokenAddress = await testToken.getAddress();
      const validUntil = BigInt((await time.latest()) + 7 * 24 * 60 * 60); // 7 days from now
      
      // Create and sign message
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress, validUntil);
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      
      // Verify the proof
      const tx = await proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, signature, validUntil);
      const receipt = await tx.wait();
      const event = receipt?.logs[0];
      expect(event?.topics.length).to.equal(3); // Event signature + 2 indexed params
      expect(event?.topics[0]).to.equal(proofOfReserve.interface.getEvent("ProofVerified").topicHash);
      expect(event?.topics[1]).to.equal(ethers.zeroPadValue(tokenAddress.toLowerCase(), 32)); // token
      expect(event?.topics[2]).to.equal(ethers.zeroPadValue(reserveWalletAddress.toLowerCase(), 32)); // wallet
      const decodedData = proofOfReserve.interface.decodeEventLog("ProofVerified", event?.data || "", event?.topics || []);
      expect(decodedData.success).to.be.true;
      expect(decodedData.validUntil).to.equal(validUntil);

      // Check proof validity
      const { isValid, validUntil: storedValidUntil } = await proofOfReserve.isProofValid(tokenAddress, reserveWalletAddress);
      expect(isValid).to.be.true;
      expect(storedValidUntil).to.equal(validUntil);
    });

    it("Should reject proof verification with invalid signature", async function () {
      const tokenAddress = await testToken.getAddress();
      const validUntil = BigInt((await time.latest()) + 7 * 24 * 60 * 60); // 7 days from now
      
      // Create message hash and sign with wrong wallet
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress, validUntil);
      const signature = await otherAccount.signMessage(ethers.getBytes(messageHash));
      
      // Verify should return false
      const tx = await proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, signature, validUntil);
      const receipt = await tx.wait();
      const event = receipt?.logs[0];
      expect(event?.topics.length).to.equal(3); // Event signature + 2 indexed params
      expect(event?.topics[0]).to.equal(proofOfReserve.interface.getEvent("ProofVerified").topicHash);
      expect(event?.topics[1]).to.equal(ethers.zeroPadValue(tokenAddress.toLowerCase(), 32)); // token
      expect(event?.topics[2]).to.equal(ethers.zeroPadValue(reserveWalletAddress.toLowerCase(), 32)); // wallet
      const decodedData = proofOfReserve.interface.decodeEventLog("ProofVerified", event?.data || "", event?.topics || []);
      expect(decodedData.success).to.be.false;
      expect(decodedData.validUntil).to.equal(validUntil);
    });

    it("Should reject proof with validity period too far in future", async function () {
      const tokenAddress = await testToken.getAddress();
      const validUntil = BigInt((await time.latest()) + 31 * 24 * 60 * 60); // 31 days from now
      
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress, validUntil);
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, signature, validUntil)
      ).to.be.revertedWith("Validity period too long");
    });

    it("Should reject proof with past validity period", async function () {
      const tokenAddress = await testToken.getAddress();
      const validUntil = BigInt(await time.latest()); // Now (which will be in the past by the time we verify)
      
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress, validUntil);
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, signature, validUntil)
      ).to.be.revertedWith("Validity period must be in future");
    });

    it("Should correctly track validity periods", async function () {
      const tokenAddress = await testToken.getAddress();
      const validUntil = BigInt((await time.latest()) + 7 * 24 * 60 * 60); // 7 days from now
      
      // Initially should be invalid
      const { isValid: initialIsValid, validUntil: initialValidUntil } = await proofOfReserve.isProofValid(tokenAddress, reserveWalletAddress);
      expect(initialIsValid).to.be.false;
      expect(initialValidUntil).to.equal(0n);

      // Submit valid proof
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress, validUntil);
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      const tx = await proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, signature, validUntil);
      const receipt = await tx.wait();
      const event = receipt?.logs[0];
      const decodedData = proofOfReserve.interface.decodeEventLog("ProofVerified", event?.data || "", event?.topics || []);
      expect(decodedData.success).to.be.true;
      expect(decodedData.validUntil).to.equal(validUntil);

      // Should now be valid
      const { isValid: isValidAfter, validUntil: storedValidUntil } = await proofOfReserve.isProofValid(tokenAddress, reserveWalletAddress);
      expect(isValidAfter).to.be.true;
      expect(storedValidUntil).to.equal(validUntil);

      // Advance time past validity period
      await time.increase(8 * 24 * 60 * 60); // 8 days

      // Should now be invalid but validUntil should remain unchanged
      const { isValid: isValidFinal, validUntil: finalValidUntil } = await proofOfReserve.isProofValid(tokenAddress, reserveWalletAddress);
      expect(isValidFinal).to.be.false;
      expect(finalValidUntil).to.equal(validUntil);
    });

    it("Should prevent operations on deactivated reserves", async function () {
      const tokenAddress = await testToken.getAddress();
      const validUntil = BigInt((await time.latest()) + 7 * 24 * 60 * 60); // 7 days from now
      
      // Deactivate reserve
      await proofOfReserve.deactivateReserve(tokenAddress, reserveWalletAddress);
      
      // Try to get balance
      await expect(
        proofOfReserve.getReserveBalance(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Reserve not active");
      
      // Try to verify proof
      const messageHash = await proofOfReserve.getMessageHash(tokenAddress, reserveWalletAddress, validUntil);
      const signature = await reserveWallet.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, signature, validUntil)
      ).to.be.revertedWith("Reserve not active");

      // Try to check proof validity
      await expect(
        proofOfReserve.isProofValid(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Reserve not active");
    });
  });
}); 