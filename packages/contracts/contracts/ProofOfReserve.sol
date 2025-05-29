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
    bytes32 public constant DOMAIN_TYPE_HASH = keccak256(
        "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
    );

    bytes32 public constant PROOF_TYPE_HASH = keccak256(
        "Proof(address token,address wallet,uint256 balance,uint256 timestamp)"
    );

    // EIP-712 domain separator
    bytes32 public immutable DOMAIN_SEPARATOR;

    constructor() {
        DOMAIN_SEPARATOR = keccak256(
            abi.encode(
                DOMAIN_TYPE_HASH,
                keccak256("Proof of Reserve"),  // name
                keccak256("1"),                 // version
                block.chainid,                  // chainId
                address(this)                   // verifyingContract
            )
        );
    }

    /**
     * @notice Configure a new reserve wallet
     * @param token The ERC20 token address
     * @param wallet The wallet address holding the reserves and authorized to sign proofs
     */
    function configureReserve(
        address token,
        address wallet
    ) external {
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
     * @notice Hash the proof data according to EIP-712
     * @param token The ERC20 token address
     * @param wallet The reserve wallet address
     * @param balance The balance being proved
     * @param timestamp The timestamp of the proof
     */
    function hashProof(
        address token,
        address wallet,
        uint256 balance,
        uint256 timestamp
    ) public view returns (bytes32) {
        bytes32 structHash = keccak256(
            abi.encode(
                PROOF_TYPE_HASH,
                token,
                wallet,
                balance,
                timestamp
            )
        );
        return keccak256(abi.encodePacked("\x19\x01", DOMAIN_SEPARATOR, structHash));
    }

    /**
     * @notice Verify a proof of reserve
     * @param token The ERC20 token address
     * @param wallet The reserve wallet address
     * @param balance The balance being proved
     * @param timestamp The timestamp of the proof
     * @param v The recovery byte of the signature
     * @param r The first 32 bytes of the signature
     * @param s The second 32 bytes of the signature
     */
    function verifyProof(
        address token,
        address wallet,
        uint256 balance,
        uint256 timestamp,
        uint8 v,
        bytes32 r,
        bytes32 s
    ) external view returns (bool) {
        require(isReserveWallet[token][wallet], "Reserve not active");

        bytes32 digest = hashProof(token, wallet, balance, timestamp);
        address signer = ecrecover(digest, v, r, s);
        
        return signer == wallet;  // Wallet is both the holder and signer
    }

    function getDomainSeparator() external view returns (bytes32) {
        return DOMAIN_SEPARATOR;
    }
}