package module

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
)

func init() {
	register("crypto", func(worker Worker, db Db) interface{} {
		return &CryptoClient{}
	})
}

type CryptoHashClient struct {
	hash crypto.Hash
}

func (c *CryptoHashClient) Sum(input []byte) []byte {
	h := c.hash.New()
	h.Write(input)
	return h.Sum(nil)
}

type CryptoHmacClient struct {
	hash crypto.Hash
}

func (c *CryptoHmacClient) Sum(input []byte, key []byte) []byte {
	h := hmac.New(c.hash.New, key)
	h.Write(input)
	return h.Sum(nil)
}

type CryptoRsaClient struct{}

func (c *CryptoRsaClient) GenerateKey(length int) (*map[string][]byte, error) {
	if length == 0 {
		length = 2048
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, length)
	if err != nil {
		return nil, err
	}
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	prvkey := pem.EncodeToMemory(block)
	publicKey := &privateKey.PublicKey
	derPubStream := x509.MarshalPKCS1PublicKey(publicKey)
	pubKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPubStream,
	})
	return &map[string][]byte{
		"privateKey": prvkey,
		"publicKey":  pubKey,
	}, nil
}

func (c *CryptoRsaClient) Encrypt(input []byte, key []byte) ([]byte, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("the public key is invalid")
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.EncryptPKCS1v15(rand.Reader, publicKey, input)
}

func (c *CryptoRsaClient) Decrypt(input []byte, key []byte) ([]byte, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("the private key is invalid")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, input)
}

func (c *CryptoRsaClient) Sign(input []byte, key []byte, algorithm string) ([]byte, error) {
	hash, err := GetHash(algorithm)
	if err != nil {
		return nil, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	block, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.SignPKCS1v15(nil, privateKey, hash, digest)
}

func (c *CryptoRsaClient) SignPss(input []byte, key []byte, algorithm string) ([]byte, error) {
	hash, err := GetHash(algorithm)
	if err != nil {
		return nil, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	block, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.SignPSS(rand.Reader, privateKey, hash, digest, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
	})
}

func (c *CryptoRsaClient) Verify(input []byte, sign []byte, key []byte, algorithm string) (bool, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return false, errors.New("the public key is invalid")
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return false, err
	}
	hash, err := GetHash(algorithm)
	if err != nil {
		return false, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	if err = rsa.VerifyPKCS1v15(publicKey, hash, digest[:], sign); err != nil {
		return false, nil
	}
	return true, nil
}

func (c *CryptoRsaClient) VerifyPss(input []byte, sign []byte, key []byte, algorithm string) (bool, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return false, errors.New("the public key is invalid")
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return false, err
	}
	hash, err := GetHash(algorithm)
	if err != nil {
		return false, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	if err = rsa.VerifyPSS(publicKey, hash, digest[:], sign, nil); err != nil {
		return false, nil
	}
	return true, nil
}

type CryptoClient struct{}

func GetHash(algorithm string) (crypto.Hash, error) {
	switch strings.ToLower(algorithm) {
	case "md5":
		return crypto.MD5, nil
	case "sha1":
		return crypto.SHA1, nil
	case "sha256":
		return crypto.SHA256, nil
	case "sha512":
		return crypto.SHA512, nil
	default:
		return crypto.SHA256, errors.New("Hash algorithm " + algorithm + " is not supported.")
	}
}

func (c *CryptoClient) CreateHash(algorithm string) (*CryptoHashClient, error) {
	if hash, err := GetHash(algorithm); err != nil {
		return nil, err
	} else {
		return &CryptoHashClient{
			hash: hash,
		}, nil
	}
}

func (c *CryptoClient) CreateHmac(algorithm string) (*CryptoHmacClient, error) {
	if hash, err := GetHash(algorithm); err != nil {
		return nil, err
	} else {
		return &CryptoHmacClient{
			hash: hash,
		}, nil
	}
}

func (c *CryptoClient) CreateRsa() *CryptoRsaClient {
	return &CryptoRsaClient{}
}
