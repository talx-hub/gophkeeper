package hash

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

func GenerateHMAC(data []byte, secret []byte) []byte {
	hasher := hmac.New(sha256.New, secret)
	hasher.Write(data)
	hash := hasher.Sum(nil)

	return hash
}

func GenerateSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

const passwordHashingCompetitionFormat = "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"

func GenerateFromPassword(password []byte) (string, error) {
	const saltSize = 32
	salt, err := GenerateRandom(saltSize)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	const timeCost = 2
	const memoryCost = 64 * 1024
	const threads = 1
	const hashLen = 32
	hash := argon2.IDKey(password, salt, timeCost, memoryCost, threads, hashLen)

	phc := fmt.Sprintf(passwordHashingCompetitionFormat,
		argon2.Version,
		memoryCost,
		timeCost,
		threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return phc, nil
}

func CompareHashAndPassword(phc string, password []byte) error {
	var (
		version                int
		saltBase64, hashBase64 string
		timeCost               uint32
		memoryCost             uint32
		threads                uint8
		hashLen                uint32
	)
	_, err := fmt.Sscanf(phc, passwordHashingCompetitionFormat,
		&version,
		&memoryCost,
		&timeCost,
		&threads,
		&saltBase64,
		&hashBase64)
	if err != nil {
		return fmt.Errorf("failed to parse PHC: %w", err)
	}
	if version != argon2.Version {
		return fmt.Errorf(
			"argon2 version mismatch: have %d want %d", version, argon2.Version)
	}

	salt, err := base64.RawStdEncoding.DecodeString(saltBase64)
	if err != nil {
		return fmt.Errorf("salt base64.Decode: %w", err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(hashBase64)
	if err != nil {
		return fmt.Errorf("hash base64.Decode: %w", err)
	}
	hashLen = uint32(len(hash))

	computed := argon2.IDKey(password, salt, timeCost, memoryCost, threads, hashLen)
	if subtle.ConstantTimeCompare(computed, hash) == 0 {
		return errors.New("password is incorrect")
	}
	return nil
}

func GenerateRandom(size int) ([]byte, error) {
	random := make([]byte, size)
	if _, err := rand.Read(random); err != nil {
		return nil, fmt.Errorf("failed to generate random: %w", err)
	}
	return random, nil
}
