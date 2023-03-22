package api_security

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
)

func Sha1(value string) string {
	hash := sha1.New()
	hash.Write([]byte(value))
	result := hex.EncodeToString(hash.Sum(nil))
	return result
}

func Sha256(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func CodeChallengeToCode(codeChallenge string) string {
	return Sha256(Sha1(codeChallenge))
}
