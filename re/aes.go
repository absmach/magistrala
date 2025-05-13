package re

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
)


func hexToBytes(s string) []byte {
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		log.Fatalf("hex decode error: %v", err)
	}
	return b
}


func reverseBytes(b []byte) []byte {
	for i := 0; i < len(b)/2; i++ {
		b[i], b[len(b)-1-i] = b[len(b)-1-i], b[i]
	}
	return b
}


func generateIV(accessNumber byte, deviceID string) []byte {
	idBytes := reverseBytes(hexToBytes(deviceID)) 
	if len(idBytes) != 4 {
		log.Fatalf("Device ID must be 4 bytes")
	}
	iv := make([]byte, 16)
	copy(iv[4:], append(idBytes, []byte{accessNumber}...)) 
	return iv
}



func decryptAES128CBC(key, iv, encrypted []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("NewCipher error: %v", err)
	}
	if len(encrypted)%aes.BlockSize != 0 {
		log.Fatalf("Encrypted data is not a multiple of the block size")
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)
	return decrypted
}

