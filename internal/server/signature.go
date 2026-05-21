package server

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"golang.org/x/crypto/ripemd160"
)

func verifyCosmosArbitrarySignature(signerAddress string, message []byte, pubkeyBase64, signatureBase64 string) error {
	pubkeyBytes, err := base64.StdEncoding.DecodeString(pubkeyBase64)
	if err != nil {
		return errors.New("wallet_pubkey must be valid base64")
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return errors.New("wallet_signature must be valid base64")
	}

	if len(pubkeyBytes) != 33 {
		return errors.New("pubkey must be 33 bytes (compressed secp256k1)")
	}
	if len(signatureBytes) != 64 {
		return errors.New("signature must be 64 bytes (r||s)")
	}

	derivedAddress, err := deriveSecp256k1Address(pubkeyBytes)
	if err != nil {
		return fmt.Errorf("derive address from pubkey: %w", err)
	}
	if derivedAddress != signerAddress {
		return errors.New("signer address does not match pubkey")
	}

	msgHash := sha256.Sum256(message)

	pubkey, err := btcec.ParsePubKey(pubkeyBytes)
	if err != nil {
		return errors.New("invalid secp256k1 pubkey")
	}

	r := new(big.Int).SetBytes(signatureBytes[:32])
	s := new(big.Int).SetBytes(signatureBytes[32:])
	ok := ecdsa.Verify(pubkey.ToECDSA(), msgHash[:], r, s)
	if !ok {
		return errors.New("signature verification failed")
	}

	return nil
}

func deriveSecp256k1Address(pubkeyBytes []byte) (string, error) {
	if len(pubkeyBytes) != 33 {
		return "", errors.New("invalid compressed pubkey size")
	}

	hasherSHA := sha256.New()
	_, _ = hasherSHA.Write(pubkeyBytes[1:])
	shaDigest := hasherSHA.Sum(nil)

	hasherRIPEMD := ripemd160.New()
	_, _ = hasherRIPEMD.Write(shaDigest)
	pubkeyHash := hasherRIPEMD.Sum(nil)

	address, err := bech32.Encode("wevibe", pubkeyHash)
	if err != nil {
		return "", fmt.Errorf("bech32 encode: %w", err)
	}
	return address, nil
}
