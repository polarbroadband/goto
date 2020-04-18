package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	log "github.com/sirupsen/logrus"
)

// Written in 2015 by George Tankersley <george.tankersley@gmail.com>

/* ****************************************
Encryption - 256-bit AES-GCM with random 96-bit nonces
**************************************** */

// NewEncryptionKey generates a random 256-bit key for Encrypt() and
// Decrypt(). It panics if the source of randomness fails.
func NewEncryptionKey() *[32]byte {
	key := [32]byte{}
	_, err := io.ReadFull(rand.Reader, key[:])
	if err != nil {
		panic(err)
	}
	return &key
}

// Encrypt encrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Output takes the
// form nonce|ciphertext|tag where '|' indicates concatenation.
func Encrypt(plaintext []byte, key *[32]byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		log.WithError(err).Warn("erroneous cipher block")
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.WithError(err).Warn("erroneous GCM")
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		log.WithError(err).Warn("erroneous random reader")
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input
// form nonce|ciphertext|tag where '|' indicates concatenation.
func Decrypt(ciphertext []byte, key *[32]byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		log.WithError(err).Warn("erroneous cipher block")
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.WithError(err).Warn("erroneous GCM")
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		log.Warn("malformed ciphertext")
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
}
