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
    // State variables
    address public owner;
    mapping(address => mapping(address => bool)) public isReserveWallet;    // token => wallet => isActive
    mapping(address => mapping(address => uint256)) public lastVerifiedTimestamp;  // token => wallet => timestamp

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
     * @notice Submit and verify a proof of reserve
     * @param token The ERC20 token address
     * @param wallet The reserve wallet address
     * @param signature The signature proving the wallet owns the reserves
     * @return success Whether the proof was valid
     */
    function verifyProof(
        address token,
        address wallet,
        bytes memory signature
    ) external returns (bool success) {
        require(isReserveWallet[token][wallet], "Reserve not active");
        
        // Create and verify signature of the ownership claim
        bytes32 messageHash = getMessageHash(token, wallet);
        bytes32 ethSignedMessageHash = getEthSignedMessageHash(messageHash);
        address signer = recoverSigner(ethSignedMessageHash, signature);
        
        // Verify the signature matches the wallet
        success = (signer == wallet);
        
        if (success) {
            lastVerifiedTimestamp[token][wallet] = block.timestamp;
        }
        
        emit ProofVerified(token, wallet, success);
        return success;
    }

    /**
     * @notice Creates a message hash from token and wallet
     * @param token The token address
     * @param wallet The wallet address
     */
    function getMessageHash(
        address token,
        address wallet
    ) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(
            "ProofOfReserve:",
            token,
            wallet
        ));
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

    /**
     * @notice Get comprehensive details for a specific reserve
     * @param token The ERC20 token address
     * @param wallet The reserve wallet address
     * @return details The ReserveDetails struct
     */
    function getReserveDetails(
        address token,
        address wallet
    ) external view returns (ReserveDetails memory details) {
        bool configured = isReserveWallet[token][wallet];
        string memory tokenName = "";
        string memory tokenSymbol = "";
        uint256 bal = 0;
        uint256 verifiedTime = 0;

        if (configured) {
            if (token != address(0)) {
                // It's good practice to wrap these calls in a try/catch if they might revert,
                // but for a view function, letting it revert is often acceptable if token is not ERC20 compliant.
                // However, a simple frontend might prefer empty strings over a full revert.
                // For simplicity here, we'll assume valid ERC20 or accept reverts.
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
            verifiedTime = lastVerifiedTimestamp[token][wallet];
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