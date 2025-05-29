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
    // Structs
    struct ReserveConfig {
        bool isActive;          // Whether this reserve configuration is active
        address wallet;         // The wallet holding the reserves and signing proofs
    }

    // Mapping of token to reserve wallets
    mapping(address => mapping(address => bool)) public isReserveWallet;    // token => wallet => isActive

    // Events
    event ReserveConfigured(address indexed token, address indexed wallet);
    event ReserveDeactivated(address indexed token, address indexed wallet);
    event ProofSubmitted(address indexed token, address indexed wallet, uint256 balance, uint256 timestamp);

    // EIP-712 type hashes
    bytes32 private constant EIP712_DOMAIN_TYPEHASH = keccak256(
        "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
    );

    bytes32 private constant PROOF_TYPEHASH = keccak256(
        "Proof(address token,address wallet,uint256 balance,uint256 timestamp)"
    );

    bytes32 private immutable DOMAIN_SEPARATOR;

    constructor() {
        DOMAIN_SEPARATOR = keccak256(abi.encode(
            EIP712_DOMAIN_TYPEHASH,
            keccak256(bytes("Proof of Reserve")),
            keccak256(bytes("1")),
            block.chainid,
            address(this)
        ));
    }

    /**
     * @notice Configure a new reserve wallet
     * @param token The ERC20 token address
     * @param wallet The wallet address holding the reserves and authorized to sign proofs
     * @param signature The wallet's signature to prove ownership
     */
    function configureReserve(
        address token,
        address wallet,
        bytes memory signature
    ) external {
        require(token != address(0), "Invalid token address");
        require(wallet != address(0), "Invalid wallet address");
        
        // Create message for signing
        bytes32 messageHash = getMessageHash(token, wallet);
        bytes32 ethSignedMessageHash = getEthSignedMessageHash(messageHash);
        
        // Verify wallet ownership
        address signer = recoverSigner(ethSignedMessageHash, signature);
        require(signer == wallet, "Invalid wallet signature");

        isReserveWallet[token][wallet] = true;
        emit ReserveConfigured(token, wallet);
    }

    /**
     * @notice Deactivate a reserve wallet
     * @param token The ERC20 token address
     * @param wallet The wallet address to deactivate
     */
    function deactivateReserve(address token, address wallet) external {
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
     * @notice Submit a new proof of reserve
     * @param token The ERC20 token address
     * @param wallet The reserve wallet address
     */
    function submitProof(address token, address wallet) external {
        require(isReserveWallet[token][wallet], "Reserve not active");
        uint256 balance = IERC20(token).balanceOf(wallet);
        emit ProofSubmitted(token, wallet, balance, block.timestamp);
    }

    /**
     * @notice Creates a message hash from token and wallet
     */
    function getMessageHash(
        address token,
        address wallet
    ) public view returns (bytes32) {
        return keccak256(abi.encodePacked(token, wallet, address(this)));
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

    function getDomainSeparator() external view returns (bytes32) {
        return DOMAIN_SEPARATOR;
    }
}