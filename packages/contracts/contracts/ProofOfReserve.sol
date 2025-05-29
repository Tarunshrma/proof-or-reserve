// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IERC20 {
    function balanceOf(address account) external view returns (uint256);
}

/**
 * @title ProofOfReserve
 * @notice A minimal contract to manage and verify proof of reserves for tokens and wallets
 */
contract ProofOfReserve {
    // State variables
    address public owner;
    mapping(address => mapping(address => bool)) public isReserveWallet;    // token => wallet => isActive
    mapping(address => mapping(address => uint256)) public lastVerifiedTimestamp;  // token => wallet => timestamp
    mapping(address => mapping(address => uint256)) public validUntilTimestamp;   // token => wallet => validUntil

    // Events
    event ReserveConfigured(address indexed token, address indexed wallet);
    event ReserveDeactivated(address indexed token, address indexed wallet);
    event ProofVerified(address indexed token, address indexed wallet, uint256 validUntil, bool success);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    // Modifiers
    modifier onlyOwner() {
        require(msg.sender == owner, "Caller is not the owner");
        _;
    }

    modifier onlyOwnerOrWallet(address wallet) {
        require(msg.sender == owner || msg.sender == wallet, "Caller is not authorized");
        _;
    }

    /**
     * @notice Contract constructor
     */
    constructor() {
        owner = msg.sender;
        emit OwnershipTransferred(address(0), msg.sender);
    }

    /**
     * @notice Transfer contract ownership
     * @param newOwner The address of the new owner
     */
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "New owner is the zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    /**
     * @notice Configure a new reserve wallet (one-time setup)
     * @param token The ERC20 token address
     * @param wallet The wallet address holding the reserves
     */
    function configureReserve(
        address token,
        address wallet
    ) external onlyOwner {
        require(token != address(0), "Invalid token address");
        require(wallet != address(0), "Invalid wallet address");
        
        isReserveWallet[token][wallet] = true;
        emit ReserveConfigured(token, wallet);
    }

    /**
     * @notice Deactivate a reserve wallet
     * @param token The ERC20 token address
     * @param wallet The wallet address to deactivate
     */
    function deactivateReserve(address token, address wallet) external onlyOwnerOrWallet(wallet) {
        require(isReserveWallet[token][wallet], "Reserve not active");
        isReserveWallet[token][wallet] = false;
        emit ReserveDeactivated(token, wallet);
    }

    /**
     * @notice Get the current balance of a reserve wallet
     * @param token The ERC20 token address
     * @param wallet The reserve wallet address
     */
    function getReserveBalance(address token, address wallet) external view returns (uint256) {
        require(isReserveWallet[token][wallet], "Reserve not active");
        return IERC20(token).balanceOf(wallet);
    }

    /**
     * @notice Submit and verify a new proof of reserve with validity period
     * @param token The ERC20 token address
     * @param wallet The reserve wallet address
     * @param signature The signature proving the wallet owns the reserves
     * @param validUntil The timestamp until which this proof is valid
     * @return success Whether the proof was valid
     */
    function verifyProof(
        address token,
        address wallet,
        bytes memory signature,
        uint256 validUntil
    ) external returns (bool success) {
        require(isReserveWallet[token][wallet], "Reserve not active");
        require(validUntil > block.timestamp, "Validity period must be in future");
        require(validUntil <= block.timestamp + 30 days, "Validity period too long");
        
        // Create and verify signature of the ownership claim
        bytes32 messageHash = getMessageHash(token, wallet, validUntil);
        bytes32 ethSignedMessageHash = getEthSignedMessageHash(messageHash);
        address signer = recoverSigner(ethSignedMessageHash, signature);
        
        // Verify the signature matches the wallet
        success = (signer == wallet);
        
        if (success) {
            lastVerifiedTimestamp[token][wallet] = block.timestamp;
            validUntilTimestamp[token][wallet] = validUntil;
        }
        
        emit ProofVerified(token, wallet, validUntil, success);
        return success;
    }

    /**
     * @notice Check if a proof is currently valid
     * @param token The ERC20 token address
     * @param wallet The reserve wallet address
     * @return isValid Whether the proof is currently valid
     * @return validUntil When the current proof expires (0 if no valid proof)
     */
    function isProofValid(
        address token,
        address wallet
    ) external view returns (bool isValid, uint256 validUntil) {
        require(isReserveWallet[token][wallet], "Reserve not active");
        validUntil = validUntilTimestamp[token][wallet];
        isValid = validUntil > block.timestamp;
        return (isValid, validUntil);
    }

    /**
     * @notice Creates a message hash from token, wallet and validity period
     * @param token The token address
     * @param wallet The wallet address
     * @param validUntil The timestamp until which this proof is valid
     */
    function getMessageHash(
        address token,
        address wallet,
        uint256 validUntil
    ) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(token, wallet, validUntil));
    }

    /**
     * @notice Creates Ethereum signed message hash
     */
    function getEthSignedMessageHash(
        bytes32 messageHash
    ) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(
            "\x19Ethereum Signed Message:\n32",
            messageHash
        ));
    }

    /**
     * @notice Split signature into r, s, v components and recover signer
     */
    function recoverSigner(
        bytes32 ethSignedMessageHash,
        bytes memory signature
    ) public pure returns (address) {
        require(signature.length == 65, "Invalid signature length");

        bytes32 r;
        bytes32 s;
        uint8 v;

        assembly {
            r := mload(add(signature, 32))
            s := mload(add(signature, 64))
            v := byte(0, mload(add(signature, 96)))
        }

        if (v < 27) {
            v += 27;
        }

        require(v == 27 || v == 28, "Invalid signature 'v' value");

        return ecrecover(ethSignedMessageHash, v, r, s);
    }
}