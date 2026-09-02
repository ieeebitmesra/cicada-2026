package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

// cryptRandIntn returns a cryptographically random int in [0,n).
func cryptRandIntn(n int) int {
	max := big.NewInt(int64(n))
	v, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand failure: %v", err))
	}
	return int(v.Int64())
}

var (
	ErrNoSignature  = errors.New("no PGP signature found")
	ErrBadPublicKey = errors.New("cannot parse PGP public key")
	ErrBadSignature = errors.New("PGP signature verification failed")
)

// ParsePublicKey parses an armored PGP public key and returns the primary entity.
func ParsePublicKey(armored string) (*openpgp.Entity, error) {
	armored = strings.TrimSpace(armored)
	if armored == "" {
		return nil, ErrBadPublicKey
	}
	if !strings.Contains(armored, "-----BEGIN PGP PUBLIC KEY BLOCK-----") {
		return nil, ErrBadPublicKey
	}
	keys, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armored))
	if err != nil || len(keys) == 0 {
		return nil, fmt.Errorf("%w: %v", ErrBadPublicKey, err)
	}
	return keys[0], nil
}
// PublicKeyID returns the 16-hex key ID (e.g. "0x1234567890ABCDEF") for gpg CLI commands.
func PublicKeyID(armored string) string {
	e, err := ParsePublicKey(armored)
	if err != nil || e == nil || e.PrimaryKey == nil {
		return ""
	}
	return fmt.Sprintf("0x%016X", e.PrimaryKey.KeyId)
}

// PublicKeyFingerprint returns a clean 16-hex key identifier for UI displays.
func PublicKeyFingerprint(armored string) string {
	e, err := ParsePublicKey(armored)
	if err != nil || e == nil || e.PrimaryKey == nil {
		return "(none)"
	}
	return fmt.Sprintf("%016X", e.PrimaryKey.KeyId)
}

// VerifyClearsign verifies an armored clearsigned message against the team's
// armored public key and returns the signed plaintext body.
func VerifyClearsign(armoredPubkey, armoredSigned string) (string, error) {
	entity, err := ParsePublicKey(armoredPubkey)
	if err != nil {
		return "", err
	}

	block, _ := clearsign.Decode([]byte(armoredSigned))
	if block == nil || block.ArmoredSignature == nil {
		return "", ErrNoSignature
	}

	_, err = openpgp.CheckDetachedSignature(
		openpgp.EntityList{entity},
		bytes.NewReader(block.Bytes),
		block.ArmoredSignature.Body,
		&packet.Config{},
	)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBadSignature, err)
	}
	return string(bytes.TrimSpace(block.Plaintext)), nil
}

// RandomChallengeToken is unused today but handy for tests/tools.
func RandomChallengeToken() string {
	buf := make([]byte, 16)
	io.ReadFull(rand.Reader, buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}
