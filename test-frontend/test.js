const { ethers } = require("ethers");

function recoverPublicKeyFromSignature(signature, nonce, options = {}) {
  const ethPrefixed = !!options.ethPrefixed;
  const returnCompressed = !!options.returnCompressed;

  const sig = signature.startsWith("0x") ? signature : "0x" + signature;

  let msgBytes;
  if (typeof nonce === "string") {
    msgBytes = ethers.utils.toUtf8Bytes(nonce);
  } else {
    msgBytes = nonce;
  }

  let digest;
  if (ethPrefixed) {
    digest = ethers.utils.hashMessage(msgBytes);
  } else {
    digest = ethers.utils.keccak256(msgBytes);
  }

  const pubkey = ethers.utils.recoverPublicKey(digest, sig);

  if (!returnCompressed) return pubkey;

  const pubBytes = ethers.utils.arrayify(pubkey);
  const x = pubBytes.slice(1, 33);
  const yLast = pubBytes[64];
  const prefix = (yLast % 2 === 0) ? 0x02 : 0x03;
  return ethers.utils.hexlify(Uint8Array.from([prefix, ...x]));
}

function publicKeyToAddress(uncompressedPubKey) {
  const pubBytes = ethers.utils.arrayify(uncompressedPubKey);
  const hash = ethers.utils.keccak256(pubBytes.slice(1));
  return "0x" + hash.slice(-40);
}


// =====================
// 示例调用（改为真实数据）
// =====================

const signature = "24c33710f4a30d8370f9882938e3ac916c25191cf0f77b59f13267045a9d7db12d4c9447d05d6c60f82f855b996e8292d447501c8945e4114186a75a1de2e9a01c";
const nonce = "91ad171ee0f2afb51baf8cea8c3fb4c1";

const pubkey = recoverPublicKeyFromSignature(signature, nonce, { ethPrefixed: false });
console.log("Recovered public key:", pubkey);

const address = publicKeyToAddress(pubkey);
console.log("Address:", address);
