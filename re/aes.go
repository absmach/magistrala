package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

var (
	errTooShort             = errors.New("ciphertext too short")
	errNotMultipleBlockSize = errors.New("ciphertext is not a multiple of the block size")
)

func pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func unpad(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}

func encryptAES128(plaintext, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	plaintext = pad(plaintext)

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, iv, nil
}

func decryptAES128(ciphertext, iv,  key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errTooShort
	}

	//iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errNotMultipleBlockSize
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	return unpad(ciphertext), nil
}

func main(){
	encText := "2e4409073708130007047a46002005165B780f12898B235425Bc7a69e3e6B9f14f0ef3c618754c635332efd3d64659"
	ciphertext := []byte(encText)
	keys := "CB6ABFAA8D2247B59127D3B839CF34B4"
	key := []byte(keys)
	iv := "2e4409073708130007047a460020052f"
	//iv := []byte(initV)

	data, err := decryptAES128(ciphertext, iv,  key []byte)
	fmt.Println(data, err)

}