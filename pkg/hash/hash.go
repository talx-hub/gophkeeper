package hash

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const TimeCost = 2
const MemoryCost = 64 * 1024
const Threads = 1
const Len = 32
const SaltSize = 32
const AlgVersion = argon2.Version

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
	salt, err := GenerateRandom(SaltSize)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey(password, salt, TimeCost, MemoryCost, Threads, Len)

	return GeneratePHC(
		hash,
		salt,
		AlgVersion,
		MemoryCost,
		TimeCost,
		Threads,
	), nil
}

func CompareHashAndPassword(phc string, password []byte) error {
	var (
		version                int
		saltBase64, hashBase64 string
		tCost                  uint32
		mCost                  uint32
		threadCount            uint8
	)
	parts := strings.Split(phc, "$")

	versionStr := strings.TrimPrefix(parts[2], "v=")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return fmt.Errorf("parse PHC argon2 version: %w", err)
	}
	if version != AlgVersion {
		return fmt.Errorf(
			"argon2 version mismatch: have %d want %d", version, AlgVersion)
	}

	params := parts[3]
	saltBase64 = parts[4]
	hashBase64 = parts[5]
	_, err = fmt.Sscanf(params, "m=%d,t=%d,p=%d", &mCost, &tCost, &threadCount)
	if err != nil {
		return fmt.Errorf("parse PHC parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(saltBase64)
	if err != nil {
		return fmt.Errorf("salt base64.Decode: %w", err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(hashBase64)
	if err != nil {
		return fmt.Errorf("hash base64.Decode: %w", err)
	}

	computed := argon2.IDKey(password, salt, tCost, mCost, threadCount, uint32(len(hash)))
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

func GeneratePHC(
	passwordHash []byte,
	salt []byte,
	algVersion int,
	timeCost uint32,
	memoryCost uint32,
	threads uint8,
) string {
	return fmt.Sprintf(
		passwordHashingCompetitionFormat,
		algVersion,
		memoryCost,
		timeCost,
		threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(passwordHash),
	)
}
