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
      await proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress, ethers.parseEther("1000"), 5);
      await expect(
        proofOfReserve.getReserveBalance(tokenAddress, reserveWalletAddress)
      ).not.to.be.revertedWith("Reserve not active");
    });

    it("Should prevent non-owners from configuring reserve wallets", async function () {
      const tokenAddress = await testToken.getAddress();
      await expect(
        proofOfReserve.connect(otherAccount).configureReserve(tokenAddress, reserveWalletAddress, ethers.parseEther("1000"), 5)
      ).to.be.revertedWith("Caller is not the owner");
    });

    it("Should reject zero addresses", async function () {
      const tokenAddress = await testToken.getAddress();
      const zeroAddress = ethers.ZeroAddress;
      await expect(
        proofOfReserve.configureReserve(tokenAddress, zeroAddress, ethers.parseEther("1000"), 5)
      ).to.be.revertedWith("Invalid wallet address");
    });
  });

  describe("Reserve Management", function () {
    beforeEach(async function () {
      const tokenAddress = await testToken.getAddress();
      await proofOfReserve.configureReserve(tokenAddress, reserveWalletAddress, ethers.parseEther("1000"), 5);
    });

    it("Should allow owner to deactivate reserve", async function () {
      const tokenAddress = await testToken.getAddress();
      await proofOfReserve.deactivateReserve(tokenAddress, reserveWalletAddress);
      await expect(
        proofOfReserve.getReserveBalance(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Reserve not active");
    });

    it("Should allow wallet to deactivate its own reserve", async function () {
      const tokenAddress = await testToken.getAddress();
      await proofOfReserve.connect(reserveWallet).deactivateReserve(tokenAddress, reserveWalletAddress);
      await expect(
        proofOfReserve.getReserveBalance(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Reserve not active");
    });

    it("Should prevent unauthorized accounts from deactivating reserves", async function () {
      const tokenAddress = await testToken.getAddress();
      await expect(
        proofOfReserve.connect(otherAccount).deactivateReserve(tokenAddress, reserveWalletAddress)
      ).to.be.revertedWith("Caller is not authorized");
    });

    it("Should revert verifyProof if reserve is not active", async function () {
      const tokenAddress = await testToken.getAddress();
      await proofOfReserve.deactivateReserve(tokenAddress, reserveWalletAddress);
      const dummySig = ethers.hexlify(ethers.randomBytes(65));
      const validUntil = BigInt((await time.latest()) + 7 * 24 * 60 * 60);
      await expect(
        proofOfReserve.verifyProof(tokenAddress, reserveWalletAddress, validUntil, dummySig)
      ).to.be.revertedWith("Reserve not active");
    });
  });
}); 