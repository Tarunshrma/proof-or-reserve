// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IERC20 {
    function balanceOf(address account) external view returns (uint256);
}

contract ProofOfReserve {
    address public immutable token;           // ERC-20 Token
    address public immutable reserveWallet;   // Controlled wallet

    // EIP-712 type hashes
    bytes32 public constant DOMAIN_TYPE_HASH = keccak256(
        "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
    );

    bytes32 public constant PROOF_TYPE_HASH = keccak256(
        "Proof(uint256 balance,uint256 timestamp)"
    );

    // EIP-712 domain separator
    bytes32 public immutable DOMAIN_SEPARATOR;

    constructor(address _token, address _wallet) {
        token = _token;
        reserveWallet = _wallet;

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

    function getReserveBalance() external view returns (uint256) {
        return IERC20(token).balanceOf(reserveWallet);
    }

    function hashProof(uint256 balance, uint256 timestamp) public view returns (bytes32) {
        bytes32 structHash = keccak256(
            abi.encode(
                PROOF_TYPE_HASH,
                balance,
                timestamp
            )
        );
        return keccak256(abi.encodePacked("\x19\x01", DOMAIN_SEPARATOR, structHash));
    }

    function verifyProof(
        uint256 balance, 
        uint256 timestamp, 
        uint8 v, 
        bytes32 r, 
        bytes32 s
    ) external view returns (bool) {
        bytes32 digest = hashProof(balance, timestamp);
        address signer = ecrecover(digest, v, r, s);
        return signer == reserveWallet;
    }

    function getDomainSeparator() external view returns (bytes32) {
        return DOMAIN_SEPARATOR;
    }
}