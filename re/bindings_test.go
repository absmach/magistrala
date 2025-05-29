package re

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestDecodeParams(t *testing.T) {
	tests := []struct {
		name         string
		keyStr       string
		ivStr        string
		dataStr      string
		expectedKey  []byte
		expectedIV   []byte
		expectedData []byte
		err          bool
		errMsg       string
	}{
		{
			name:         "valid hex strings",
			keyStr:       "0123456789abcdef0123456789abcdef", // 32 chars = 16 bytes
			ivStr:        "fedcba9876543210fedcba9876543210", // 32 chars = 16 bytes
			dataStr:      "deadbeefcafebabe0000111122223333", // 32 chars = 16 bytes
			expectedKey:  []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
			expectedIV:   []byte{0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
			expectedData: []byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe, 0xba, 0xbe, 0x00, 0x00, 0x11, 0x11, 0x22, 0x22, 0x33, 0x33},
			err:          false,
		},
		{
			name:         "empty data",
			keyStr:       "0123456789abcdef0123456789abcdef",
			ivStr:        "fedcba9876543210fedcba9876543210",
			dataStr:      "",
			expectedKey:  []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
			expectedIV:   []byte{0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
			expectedData: []byte{},
			err:          false,
		},
		{
			name:    "invalid key hex",
			keyStr:  "invalid_hex",
			ivStr:   "fedcba9876543210fedcba9876543210",
			dataStr: "deadbeefcafebabe0000111122223333",
			err:     true,
			errMsg:  "failed to decode key",
		},
		{
			name:    "invalid IV hex",
			keyStr:  "0123456789abcdef0123456789abcdef",
			ivStr:   "invalid_hex",
			dataStr: "deadbeefcafebabe0000111122223333",
			err:     true,
			errMsg:  "failed to decode IV",
		},
		{
			name:    "invalid data hex",
			keyStr:  "0123456789abcdef0123456789abcdef",
			ivStr:   "fedcba9876543210fedcba9876543210",
			dataStr: "invalid_hex",
			err:     true,
			errMsg:  "failed to decode data",
		},
		{
			name:    "odd length key",
			keyStr:  "0123456789abcdef0123456789abcde", // 31 chars (odd)
			ivStr:   "fedcba9876543210fedcba9876543210",
			dataStr: "deadbeefcafebabe0000111122223333",
			err:     true,
			errMsg:  "failed to decode key",
		},
		{
			name:    "odd length IV",
			keyStr:  "0123456789abcdef0123456789abcdef",
			ivStr:   "fedcba9876543210fedcba987654321", // 31 chars (odd)
			dataStr: "deadbeefcafebabe0000111122223333",
			err:     true,
			errMsg:  "failed to decode IV",
		},
		{
			name:    "odd length data",
			keyStr:  "0123456789abcdef0123456789abcdef",
			ivStr:   "fedcba9876543210fedcba9876543210",
			dataStr: "deadbeefcafebabe000011112222333", // 31 chars (odd)
			err:     true,
			errMsg:  "failed to decode data",
		},
		{
			name:         "uppercase hex",
			keyStr:       "0123456789ABCDEF0123456789ABCDEF",
			ivStr:        "FEDCBA9876543210FEDCBA9876543210",
			dataStr:      "DEADBEEFCAFEBABE0000111122223333",
			expectedKey:  []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
			expectedIV:   []byte{0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
			expectedData: []byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe, 0xba, 0xbe, 0x00, 0x00, 0x11, 0x11, 0x22, 0x22, 0x33, 0x33},
			err:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			L := lua.NewState()
			defer L.Close()

			L.Push(lua.LString(tt.keyStr))
			L.Push(lua.LString(tt.ivStr))
			L.Push(lua.LString(tt.dataStr))

			key, iv, data, err := decodeParams(L)

			if tt.err {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedKey, key)
				assert.Equal(t, tt.expectedIV, iv)
				assert.Equal(t, tt.expectedData, data)
			}
		})
	}
}

func TestLuaEncrypt(t *testing.T) {
	validKey := "0123456789abcdef0123456789abcdef"  // 16 bytes
	validIV := "fedcba9876543210fedcba9876543210"   // 16 bytes
	validData := "deadbeefcafebabe0000111122223333" // 16 bytes

	tests := []struct {
		name         string
		keyStr       string
		ivStr        string
		dataStr      string
		expectReturn int
		shouldPanic  bool
		panicMsg     string
	}{
		{
			name:         "successful encryption",
			keyStr:       validKey,
			ivStr:        validIV,
			dataStr:      validData,
			expectReturn: 1,
			shouldPanic:  false,
		},
		{
			name:         "successful encryption with empty data",
			keyStr:       validKey,
			ivStr:        validIV,
			dataStr:      "",
			expectReturn: 1,
			shouldPanic:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			L := lua.NewState()
			defer L.Close()

			L.Push(lua.LString(tt.keyStr))
			L.Push(lua.LString(tt.ivStr))
			L.Push(lua.LString(tt.dataStr))

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r != nil {
						assert.Contains(t, r.(string), tt.panicMsg)
					} else {
						t.Error("Expected panic but none occurred")
					}
				}()
				luaEncrypt(L)
			} else {
				result := luaEncrypt(L)
				assert.Equal(t, tt.expectReturn, result)

				// Verify that a valid hex string was pushed to the stack
				encryptedHex := L.ToString(-1)
				_, err := hex.DecodeString(encryptedHex)
				assert.NoError(t, err, "Pushed value should be valid hex")
			}
		})
	}
}

func TestLuaDecrypt(t *testing.T) {
	validKey := "0123456789abcdef0123456789abcdef" // 16 bytes
	validIV := "fedcba9876543210fedcba9876543210"  // 16 bytes

	// Create valid encrypted data by first encrypting some data
	keyBytes, _ := hex.DecodeString(validKey)
	ivBytes, _ := hex.DecodeString(validIV)
	plainData := []byte("1234567890123456") // 16 bytes
	encryptedData, _ := encrypt(keyBytes, ivBytes, plainData)
	validEncryptedStr := hex.EncodeToString(encryptedData)

	tests := []struct {
		name         string
		keyStr       string
		ivStr        string
		dataStr      string
		expectReturn int
		shouldPanic  bool
		panicMsg     string
	}{
		{
			name:         "successful decryption",
			keyStr:       validKey,
			ivStr:        validIV,
			dataStr:      validEncryptedStr,
			expectReturn: 1,
			shouldPanic:  false,
		},
		{
			name:         "successful decryption with empty data",
			keyStr:       validKey,
			ivStr:        validIV,
			dataStr:      "",
			expectReturn: 1,
			shouldPanic:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			L := lua.NewState()
			defer L.Close()

			L.Push(lua.LString(tt.keyStr))
			L.Push(lua.LString(tt.ivStr))
			L.Push(lua.LString(tt.dataStr))

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r != nil {
						assert.Contains(t, r.(string), tt.panicMsg)
					} else {
						t.Error("Expected panic but none occurred")
					}
				}()
				luaDecrypt(L)
			} else {
				result := luaDecrypt(L)
				assert.Equal(t, tt.expectReturn, result)

				// Verify that a valid hex string was pushed to the stack
				decryptedHex := L.ToString(-1)
				_, err := hex.DecodeString(decryptedHex)
				assert.NoError(t, err, "Pushed value should be valid hex")
			}
		})
	}
}

func TestLuaDecryptWithSample(t *testing.T) {
	iv := "0907780613000704d2d2d2d2d2d2d2d2"
	payload := "Ba56dc989e08a76f855ae12ae8B00ef13fae6ad436eBe8e03e97f17B5751c241"
	key := "CB6ABFAA8D2247B59127D3B839CF34B4"
	expected := "2f2f0c0613760100046d27350f380c13555134022f2f2f2f2f2f2f2f2f2f2f2f"

	L := lua.NewState()
	defer L.Close()

	L.Push(lua.LString(key))
	L.Push(lua.LString(iv))
	L.Push(lua.LString(payload))

	result := luaDecrypt(L)
	if result != 1 {
		t.Errorf("luaDecrypt() expected 1 return value, got %d", result)
		return
	}

	decrypted, err := hex.DecodeString(L.ToString(-1))
	require.Nil(t, err, "Failed to decode decrypted payload")
	assert.Equal(t, expected, hex.EncodeToString(decrypted), "Decrypted payload does not match expected")
}

func TestLuaEncryptDecryptRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		keyStr  string
		ivStr   string
		dataStr string
	}{
		{
			name:    "single block round trip",
			keyStr:  "0123456789abcdef0123456789abcdef",
			ivStr:   "fedcba9876543210fedcba9876543210",
			dataStr: "deadbeefcafebabe0000111122223333",
		},
		{
			name:    "multiple block round trip",
			keyStr:  "0123456789abcdef0123456789abcdef",
			ivStr:   "fedcba9876543210fedcba9876543210",
			dataStr: "deadbeefcafebabe0000111122223333cafebabe0123456789abcdef01234567",
		},
		{
			name:    "empty data round trip",
			keyStr:  "0123456789abcdef0123456789abcdef",
			ivStr:   "fedcba9876543210fedcba9876543210",
			dataStr: "",
		},
		{
			name:    "AES-256 round trip",
			keyStr:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			ivStr:   "fedcba9876543210fedcba9876543210",
			dataStr: "deadbeefcafebabe0000111122223333",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption
			L1 := lua.NewState()
			defer L1.Close()

			L1.Push(lua.LString(tt.keyStr))
			L1.Push(lua.LString(tt.ivStr))
			L1.Push(lua.LString(tt.dataStr))

			result := luaEncrypt(L1)
			require.Equal(t, 1, result)

			// Get encrypted result
			encryptedHex := L1.ToString(-1)

			// Test decryption
			L2 := lua.NewState()
			defer L2.Close()

			L2.Push(lua.LString(tt.keyStr))
			L2.Push(lua.LString(tt.ivStr))
			L2.Push(lua.LString(encryptedHex))

			result = luaDecrypt(L2)
			require.Equal(t, 1, result)

			// Verify round trip
			decryptedHex := L2.ToString(-1)
			assert.Equal(t, strings.ToLower(tt.dataStr), strings.ToLower(decryptedHex))
		})
	}
}
