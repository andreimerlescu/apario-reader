package main

import (
	`crypto/aes`
	`crypto/cipher`
	`crypto/rand`
	`crypto/sha256`
	`encoding/base64`
	`errors`
	`io`

	`golang.org/x/crypto/bcrypt`
	`golang.org/x/crypto/pbkdf2`
)

func f_s_bcrypt_password(password string) (string, error) {
	hashed_password, hash_err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if hash_err != nil {
		return "", hash_err
	}
	return string(hashed_password), nil
}

func f_encryption_derive_key(password string) []byte {
	salt := []byte("use-a-unique-salt")
	return pbkdf2.Key([]byte(password), salt, 4096, 32, sha256.New)
}

func f_s_encrypt_string(data string, password string) (string, error) {
	sem_concurrent_crypt_actions.Acquire()
	defer sem_concurrent_crypt_actions.Release()
	key := f_encryption_derive_key(password)
	block, block_err := aes.NewCipher(key)
	if block_err != nil {
		return "", block_err
	}
	gcm, gcm_err := cipher.NewGCM(block)
	if gcm_err != nil {
		return "", gcm_err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipher_text := gcm.Seal(nonce, nonce, []byte(data), nil)
	return base64.StdEncoding.EncodeToString(cipher_text), nil
}

func f_s_decrypt_string(base64_cipher_text string, password string) (string, error) {
	sem_concurrent_crypt_actions.Acquire()
	defer sem_concurrent_crypt_actions.Release()
	key := f_encryption_derive_key(password)
	cipher_text, text_err := base64.StdEncoding.DecodeString(base64_cipher_text)
	if text_err != nil {
		return "", text_err
	}
	block, block_err := aes.NewCipher(key)
	if block_err != nil {
		return "", block_err
	}
	gcm, gcm_err := cipher.NewGCM(block)
	if gcm_err != nil {
		return "", gcm_err
	}
	if len(cipher_text) < gcm.NonceSize() {
		return "", errors.New("cipher_text too short")
	}
	nonce := cipher_text[:gcm.NonceSize()]
	cipher_text = cipher_text[gcm.NonceSize():]
	plain_text, decrypt_err := gcm.Open(nil, nonce, cipher_text, nil)
	if decrypt_err != nil {
		return "", decrypt_err
	}
	return string(plain_text), nil
}
