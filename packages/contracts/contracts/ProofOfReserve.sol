// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IERC20 {
    function balanceOf(address account) external view returns (uint256);
    function name() external view returns (string memory);
    function symbol() external view returns (string memory);
}

/**
 * @title ProofOfReserve
 * @notice A minimal contract to manage and verify proof of reserves for tokens and wallets
 */
contract ProofOfReserve {
    // EIP-712 type hashes
    bytes32 public constant DOMAIN_TYPEHASH = keccak256(
        "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
    );
    
    bytes32 public constant PROOF_TYPEHASH = keccak256(
        "Proof(address token,address wallet,uint256 validUntil)"
    );

    string public constant DOMAIN_NAME = "ProofOfReserve";
    string public constant DOMAIN_VERSION = "1";
    uint256 public immutable CHAIN_ID;
    bytes32 private immutable _DOMAIN_SEPARATOR;
    
    // State variables
    address public owner;
    struct ReserveConfig {
        bool isConfigured;
        uint256 target;
        uint256 thresholdPercent;
        uint256 lastVerifiedTimestamp;
    }
    mapping(address => mapping(address => ReserveConfig)) public reserveConfigs; // token => wallet => config

    // Events
    event ReserveConfigured(address indexed token, address indexed wallet);
    event ReserveDeactivated(address indexed token, address indexed wallet);
    event ProofVerified(address indexed token, address indexed wallet, bool success);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    // Struct for returning multiple details
    struct ReserveDetails {
        bool isConfigured;
        string name;
        string symbol;
        uint256 balance;
        uint256 lastVerified;
    }

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
        CHAIN_ID = block.chainid;
        _DOMAIN_SEPARATOR = keccak256(
            abi.encode(
                DOMAIN_TYPEHASH,
                keccak256(bytes(DOMAIN_NAME)),
                keccak256(bytes(DOMAIN_VERSION)),
                block.chainid,
                address(this)
            )
        );
        emit OwnershipTransferred(address(0), msg.sender);
    }

    function DOMAIN_SEPARATOR() public view returns (bytes32) {
        return _DOMAIN_SEPARATOR;
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
     * @param token The ERC20 token address, or address(0) for native XDC token
     * @param wallet The wallet address holding the reserves
     */
    function configureReserve(
        address token,
        address wallet,
        uint256 target,
        uint256 thresholdPercent
    ) external onlyOwner {
        require(wallet != address(0), "Invalid wallet address");
        require(thresholdPercent < 100, "Threshold must be < 100");
        ReserveConfig storage config = reserveConfigs[token][wallet];
        require(!config.isConfigured, "Reserve already configured");
        config.isConfigured = true;
        config.target = target;
        config.thresholdPercent = thresholdPercent;
        config.lastVerifiedTimestamp = 0;
        emit ReserveConfigured(token, wallet);
    }

    /**
     * @notice Deactivate a reserve wallet
     * @param token The ERC20 token address
     * @param wallet The wallet address to deactivate
     */
    function deactivateReserve(address token, address wallet) external onlyOwnerOrWallet(wallet) {
        ReserveConfig storage config = reserveConfigs[token][wallet];
        require(config.isConfigured, "Reserve not active");
        config.isConfigured = false;
        emit ReserveDeactivated(token, wallet);
    }

    /**
     * @notice Get the current balance of a reserve wallet
     * @param token The ERC20 token address, or address(0) for the native chain token (e.g., XDC)
     * @param wallet The reserve wallet address
     */
    function getReserveBalance(address token, address wallet) external view returns (uint256) {
        require(reserveConfigs[token][wallet].isConfigured, "Reserve not active");
        if (token == address(0)) {
            return wallet.balance;
        } else {
            return IERC20(token).balanceOf(wallet);
        }
    }

    /**
     * @notice Submit and verify a proof of reserve using EIP-712 signature
     * @param token The ERC20 token address
     * @param wallet The reserve wallet address
     * @param validUntil The timestamp until which this proof is valid
     * @param signature The EIP-712 signature proving the wallet owns the reserves
     * @return success Whether the proof was valid
     */
    function verifyProof(
        address token,
        address wallet,
        uint256 validUntil,
        bytes memory signature
    ) external returns (bool success) {
        ReserveConfig storage config = reserveConfigs[token][wallet];
        require(config.isConfigured, "Reserve not active");
        require(validUntil > block.timestamp, "Proof has expired");
        uint256 currentBalance = (token == address(0)) ? wallet.balance : IERC20(token).balanceOf(wallet);
        if (config.target > 0) {
            uint256 minRequired = config.target * (100 - config.thresholdPercent) / 100;
            require(currentBalance >= minRequired, "Reserve balance below threshold");
        }
        // Verify EIP-712 signature
        bytes32 digest = keccak256(
            abi.encodePacked(
                "\x19\x01",
                DOMAIN_SEPARATOR(),
                keccak256(
                    abi.encode(
                        PROOF_TYPEHASH,
                        token,
                        wallet,
                        validUntil
                    )
                )
            )
        );
        address signer = recoverSigner(digest, signature);
        success = (signer == wallet);
        if (success) {
            config.lastVerifiedTimestamp = block.timestamp;
        }
        emit ProofVerified(token, wallet, success);
        return success;
    }

    /**
     * @notice Split signature into r, s, v components and recover signer
     */
    function recoverSigner(
        bytes32 digest,
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

        return ecrecover(digest, v, r, s);
    }

    /**
     * @notice Get comprehensive details for a specific reserve
     * @param token The ERC20 token address, or address(0) for the native chain token (e.g., XDC)
     * @param wallet The reserve wallet address
     * @return details The ReserveDetails struct
     */
    function getReserveDetails(
        address token,
        address wallet
    ) external view returns (ReserveDetails memory details) {
        ReserveConfig storage config = reserveConfigs[token][wallet];
        bool configured = config.isConfigured;
        string memory tokenName = "";
        string memory tokenSymbol = "";
        uint256 bal = 0;
        uint256 verifiedTime = config.lastVerifiedTimestamp;

        if (configured) {
            if (token == address(0)) {
                // Handle native chain token (e.g., XDC)
                tokenName = "XinFin XDC";
                tokenSymbol = "XDC";
                bal = wallet.balance;
            } else if (token != address(0)) {
                // Handle ERC20 token
                IERC20 tokenContract = IERC20(token);
                try tokenContract.name() returns (string memory _name) {
                    tokenName = _name;
                } catch { /* Fails silently, tokenName remains empty */ }
                try tokenContract.symbol() returns (string memory _symbol) {
                    tokenSymbol = _symbol;
                } catch { /* Fails silently, tokenSymbol remains empty */ }
                try tokenContract.balanceOf(wallet) returns (uint256 _balance) {
                    bal = _balance;
                } catch { /* Fails silently, bal remains 0 */ }
            }
            verifiedTime = config.lastVerifiedTimestamp;
        }

        details = ReserveDetails({
            isConfigured: configured,
            name: tokenName,
            symbol: tokenSymbol,
            balance: bal,
            lastVerified: verifiedTime
        });
        return details;
    }
}