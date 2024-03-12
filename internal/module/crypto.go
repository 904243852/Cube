package module

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"cube/internal/builtin"
	"encoding/pem"
	"errors"
	"strings"
)

func init() {
	register("crypto", func(worker Worker, db Db) interface{} {
		return &CryptoClient{}
	})
}

//#region Cipher

type AesCipherClient struct {
	block   cipher.Block
	padding string
}

func (c *AesCipherClient) pad(input []byte, blockSize int, padType string) ([]byte, error) {
	switch strings.ToLower(padType) {
	case "none":
		return input, nil
	case "pkcs5": // pkcs5 填充模式：为 pkcs7 的子集，方式与 pkcs7 相同，不同的是 pkcs5 的 blockSize 固定为 8，而 pkcs7 的 blockSize 为 1 - 255
		fallthrough
	case "pkcs7": // pkcs7 填充模式：在原文末尾填充 padSize（其中 1 ≤ padSize ≤ blockSize）个字节 padByte（值为 padSize），使得总长度为 blockSize 的整数倍
		padSize := blockSize - (len(input) % blockSize)                      // 需要填充的长度
		padByte := byte(padSize)                                             // 需要填充的字节
		return append(input, bytes.Repeat([]byte{padByte}, padSize)...), nil // 在原文末尾填充 padSize 个字节 padByte
	default:
		return nil, errors.New("padding " + padType + " is not supported")
	}
}

func (c *AesCipherClient) unpad(input []byte, blockSize int, padType string) ([]byte, error) {
	switch strings.ToLower(padType) {
	case "none":
		return input, nil
	case "pkcs5":
		fallthrough // 同 pkcs7
	case "pkcs7":
		padByte := input[len(input)-1]         // 最后一个字节，即为填充所使用的字节
		padSize := int(padByte)                // 填充的字节值，也是所填充字节的长度
		return input[:len(input)-padSize], nil // 去除末尾 padSize 个字节
	default:
		return nil, errors.New("padding " + padType + " is not supported")
	}
}

type CryptoCipherClient interface {
	Encrypt(input []byte) (builtin.Buffer, error)
	Decrypt(input []byte) (builtin.Buffer, error)
}

type AesEcbCipherClient struct {
	*AesCipherClient
}

func (c *AesEcbCipherClient) Encrypt(input []byte) (builtin.Buffer, error) {
	blockSize := c.block.BlockSize()
	// 填充
	input, err := c.pad(input, blockSize, c.padding)
	if err != nil {
		return nil, err
	}
	// 分组加密
	output, buffer := make([]byte, 0), make([]byte, blockSize)
	for i, j := 0, len(input); i < j; i += blockSize {
		c.block.Encrypt(buffer, input[i:i+blockSize])
		output = append(output, buffer...)
	}
	return output, nil

}

func (c *AesEcbCipherClient) Decrypt(input []byte) (builtin.Buffer, error) {
	blockSize := c.block.BlockSize()
	// 分组解密
	output, buffer := make([]byte, 0), make([]byte, blockSize)
	for i, j := 0, len(input); i < j; i += blockSize {
		c.block.Decrypt(buffer, input[i:i+blockSize])
		output = append(output, buffer...)
	}
	// 去除填充
	return c.unpad(output, blockSize, c.padding)
}

//#endregion

//#region Hash

type CryptoHashClient struct {
	hash crypto.Hash
}

func (c *CryptoHashClient) Sum(input []byte) builtin.Buffer {
	h := c.hash.New()
	h.Write(input)
	return h.Sum(nil)
}

//#endregion

//#region Hmac

type CryptoHmacClient struct {
	hash crypto.Hash
}

func (c *CryptoHmacClient) Sum(input []byte, key []byte) builtin.Buffer {
	h := hmac.New(c.hash.New, key)
	h.Write(input)
	return h.Sum(nil)
}

//#endregion

//#region Rsa

type CryptoRsaClient struct{}

func (c *CryptoRsaClient) GenerateKey(length int) (*map[string]builtin.Buffer, error) {
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
	return &map[string]builtin.Buffer{
		"privateKey": prvkey,
		"publicKey":  pubKey,
	}, nil
}

func (c *CryptoRsaClient) Encrypt(input []byte, key []byte, padding string) (builtin.Buffer, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("public key is invalid")
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	if padding == "oaep" {
		return rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, input, nil)
	}
	return rsa.EncryptPKCS1v15(rand.Reader, publicKey, input)
}

func (c *CryptoRsaClient) Decrypt(input []byte, key []byte, padding string) (builtin.Buffer, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("private key is invalid")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	if padding == "oaep" {
		return rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, input, nil)
	}
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, input)
}

func (c *CryptoRsaClient) Sign(input []byte, key []byte, algorithm string, padding string) (builtin.Buffer, error) {
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
	if padding == "pss" {
		return rsa.SignPSS(rand.Reader, privateKey, hash, digest, &rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
		})
	}
	return rsa.SignPKCS1v15(nil, privateKey, hash, digest)
}

func (c *CryptoRsaClient) Verify(input []byte, sign []byte, key []byte, algorithm string, padding string) (bool, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return false, errors.New("public key is invalid")
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
	if padding == "pss" {
		if err = rsa.VerifyPSS(publicKey, hash, digest[:], sign, nil); err != nil {
			return false, nil
		}
	} else {
		if err = rsa.VerifyPKCS1v15(publicKey, hash, digest[:], sign); err != nil {
			return false, nil
		}
	}
	return true, nil
}

//#endregion

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
		return crypto.SHA256, errors.New("hash algorithm " + algorithm + " is not supported")
	}
}

type CryptoClient struct{}

func (c *CryptoClient) CreateCipher(algorithm string, key []byte, options map[string]interface{}) (CryptoCipherClient, error) {
	switch strings.ToLower(algorithm) {
	case "aes-ecb":
		if block, err := aes.NewCipher(key); err != nil { // key 长度必须为 16（128 bits）、24（192 bits）或 32（256 bits）
			return nil, err
		} else {
			padding := options["padding"].(string)
			return &AesEcbCipherClient{&AesCipherClient{block, padding}}, nil
		}
	default:
		return nil, errors.New("cipher algorithm " + algorithm + " is not supported")
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
