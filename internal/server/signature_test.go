package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	btcecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"golang.org/x/crypto/ripemd160"
)

type signatureFixture struct {
	signerAddress   string
	pubkeyBase64    string
	signatureBase64 string
	deriveErr       error
}

func newSignatureFixture(t *testing.T, keyScalar byte, message []byte) signatureFixture {
	t.Helper()

	keyBytes := make([]byte, 32)
	keyBytes[31] = keyScalar

	privKey, pubKey := btcec.PrivKeyFromBytes(keyBytes)
	pubkeyBytes := pubKey.SerializeCompressed()

	signerAddress, deriveErr := deriveSecp256k1Address(pubkeyBytes)
	signatureBytes := signMessageForVerify(privKey, message)

	return signatureFixture{
		signerAddress:   signerAddress,
		pubkeyBase64:    base64.StdEncoding.EncodeToString(pubkeyBytes),
		signatureBase64: base64.StdEncoding.EncodeToString(signatureBytes),
		deriveErr:       deriveErr,
	}
}

func signMessageForVerify(privKey *btcec.PrivateKey, message []byte) []byte {
	msgHash := sha256.Sum256(message)
	sig := btcecdsa.Sign(privKey, msgHash[:])

	r := sig.R()
	s := sig.S()
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	signature := make([]byte, 64)
	copy(signature[:32], rBytes[:])
	copy(signature[32:], sBytes[:])

	return signature
}

func TestSignature_DeriveSecp256k1Address_Deterministic(t *testing.T) {
	keyBytes := make([]byte, 32)
	keyBytes[31] = 1
	_, pubKey := btcec.PrivKeyFromBytes(keyBytes)
	compressedPubkey := pubKey.SerializeCompressed()

	firstAddress, firstErr := deriveSecp256k1Address(compressedPubkey)
	secondAddress, secondErr := deriveSecp256k1Address(compressedPubkey)

	if firstErr != nil {
		t.Fatalf("expected no derive error on first call, got %v", firstErr)
	}
	if secondErr != nil {
		t.Fatalf("expected no derive error on second call, got %v", secondErr)
	}

	if firstAddress != secondAddress {
		t.Fatalf("expected deterministic address, got first=%q second=%q", firstAddress, secondAddress)
	}
	if !strings.HasPrefix(firstAddress, "wevibe1") {
		t.Fatalf("expected wevibe bech32 HRP, got %q", firstAddress)
	}
	if len(firstAddress) != 45 {
		t.Fatalf("expected bech32 address length 45, got %d (%q)", len(firstAddress), firstAddress)
	}
}

func TestSignature_DeriveSecp256k1Address_CosmosBech32RoundTrip(t *testing.T) {
	keyBytes := make([]byte, 32)
	keyBytes[31] = 7
	_, pubKey := btcec.PrivKeyFromBytes(keyBytes)
	compressedPubkey := pubKey.SerializeCompressed()

	address, err := deriveSecp256k1Address(compressedPubkey)
	if err != nil {
		t.Fatalf("expected successful derivation, got %v", err)
	}

	shaDigest := sha256.Sum256(compressedPubkey[1:])
	ripemdHasher := ripemd160.New()
	_, _ = ripemdHasher.Write(shaDigest[:])
	expectedPubkeyHash := ripemdHasher.Sum(nil)

	hrP, fiveBitData, err := bech32.Decode(address)
	if err != nil {
		t.Fatalf("expected valid bech32 address, decode failed: %v", err)
	}
	if hrP != "wevibe" {
		t.Fatalf("expected HRP wevibe, got %q", hrP)
	}

	roundTrippedHash, err := bech32.ConvertBits(fiveBitData, 5, 8, false)
	if err != nil {
		t.Fatalf("convert 5-bit payload back to 8-bit failed: %v", err)
	}
	if !bytes.Equal(roundTrippedHash, expectedPubkeyHash) {
		t.Fatalf("round-tripped hash mismatch: expected %x got %x", expectedPubkeyHash, roundTrippedHash)
	}

	converted, err := bech32.ConvertBits(expectedPubkeyHash, 8, 5, true)
	if err != nil {
		t.Fatalf("convert expected hash to 5-bit failed: %v", err)
	}
	expectedAddress, err := bech32.Encode("wevibe", converted)
	if err != nil {
		t.Fatalf("encode expected address failed: %v", err)
	}
	if address != expectedAddress {
		t.Fatalf("address mismatch: expected %q got %q", expectedAddress, address)
	}
}

func TestSignature_DeriveSecp256k1Address_InvalidPubkeys(t *testing.T) {
	tests := []struct {
		name   string
		pubkey []byte
	}{
		{name: "empty", pubkey: []byte{}},
		{name: "wrong length 32", pubkey: bytes.Repeat([]byte{0x01}, 32)},
		{name: "wrong length 34", pubkey: bytes.Repeat([]byte{0x01}, 34)},
		{name: "non curve garbage", pubkey: append([]byte{0x02}, bytes.Repeat([]byte{0xFF}, 32)...)},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			panicked := false
			var (
				address string
				err     error
			)

			func() {
				defer func() {
					if recover() != nil {
						panicked = true
					}
				}()
				address, err = deriveSecp256k1Address(tc.pubkey)
			}()

			if panicked {
				t.Fatalf("deriveSecp256k1Address panicked for input %q", tc.name)
			}
			if err == nil {
				t.Fatalf("expected error for input %q, got nil", tc.name)
			}
			if address != "" {
				t.Fatalf("expected empty address on error for input %q, got %q", tc.name, address)
			}
		})
	}
}

func TestSignature_VerifyCosmosArbitrarySignature_HappyPath(t *testing.T) {
	message := []byte("wevibe-social-graph signature happy path")
	fixture := newSignatureFixture(t, 2, message)

	if fixture.deriveErr != nil {
		t.Fatalf("expected fixture address derivation to succeed, got %v", fixture.deriveErr)
	}

	if err := verifyCosmosArbitrarySignature(fixture.signerAddress, message, fixture.pubkeyBase64, fixture.signatureBase64); err != nil {
		t.Fatalf("expected successful signature verification, got %v", err)
	}
}

func TestSignature_VerifyCosmosArbitrarySignature_TamperedMessage(t *testing.T) {
	originalMessage := []byte("wevibe-social-graph signature original")
	tamperedMessage := []byte("wevibe-social-graph signature tampered")
	fixture := newSignatureFixture(t, 3, originalMessage)
	if fixture.deriveErr != nil {
		t.Fatalf("expected fixture address derivation to succeed, got %v", fixture.deriveErr)
	}

	err := verifyCosmosArbitrarySignature(fixture.signerAddress, tamperedMessage, fixture.pubkeyBase64, fixture.signatureBase64)
	if err == nil {
		t.Fatalf("expected verification error for tampered message, got nil")
	}
	if !strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("expected signature verification failure, got %v", err)
	}
}

func TestSignature_VerifyCosmosArbitrarySignature_WrongPubkey(t *testing.T) {
	message := []byte("wevibe-social-graph signature wrong pubkey")
	correctSigner := newSignatureFixture(t, 4, message)
	wrongPubkey := newSignatureFixture(t, 5, message)
	if correctSigner.deriveErr != nil {
		t.Fatalf("expected correct signer fixture derivation to succeed, got %v", correctSigner.deriveErr)
	}
	if wrongPubkey.deriveErr != nil {
		t.Fatalf("expected wrong pubkey fixture derivation to succeed, got %v", wrongPubkey.deriveErr)
	}

	err := verifyCosmosArbitrarySignature(correctSigner.signerAddress, message, wrongPubkey.pubkeyBase64, correctSigner.signatureBase64)
	if err == nil {
		t.Fatalf("expected verification error with wrong pubkey, got nil")
	}
	if !strings.Contains(err.Error(), "signer address does not match pubkey") {
		t.Fatalf("expected signer/pubkey mismatch error, got %v", err)
	}
}

func TestSignature_VerifyCosmosArbitrarySignature_MalformedBase64(t *testing.T) {
	validPubkeyBase64 := base64.StdEncoding.EncodeToString(append([]byte{0x02}, bytes.Repeat([]byte{0x01}, 32)...))
	validSignatureBase64 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x02}, 64))

	tests := []struct {
		name            string
		pubkeyBase64    string
		signatureBase64 string
		expectedError   string
	}{
		{
			name:            "malformed pubkey base64",
			pubkeyBase64:    "%%%not-base64%%%",
			signatureBase64: validSignatureBase64,
			expectedError:   "wallet_pubkey must be valid base64",
		},
		{
			name:            "malformed signature base64",
			pubkeyBase64:    validPubkeyBase64,
			signatureBase64: "%%%not-base64%%%",
			expectedError:   "wallet_signature must be valid base64",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := verifyCosmosArbitrarySignature("wevibe1ignored", []byte("message"), tc.pubkeyBase64, tc.signatureBase64)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tc.name)
			}
			if err.Error() != tc.expectedError {
				t.Fatalf("expected %q, got %q", tc.expectedError, err.Error())
			}
		})
	}
}

func TestSignature_VerifyCosmosArbitrarySignature_InvalidSignatureLength(t *testing.T) {
	fixture := newSignatureFixture(t, 6, []byte("signature length validation"))

	tests := []struct {
		name      string
		signature string
	}{
		{
			name:      "63 bytes",
			signature: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 63)),
		},
		{
			name:      "65 bytes",
			signature: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 65)),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := verifyCosmosArbitrarySignature(fixture.signerAddress, []byte("signature length validation"), fixture.pubkeyBase64, tc.signature)
			if err == nil {
				t.Fatalf("expected signature length error, got nil")
			}
			if err.Error() != "signature must be 64 bytes (r||s)" {
				t.Fatalf("expected signature length error, got %q", err.Error())
			}
		})
	}
}
