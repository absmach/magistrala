function aes_cbc_decrypt(key_hex, iv, encrypted_hex)

    local openssl = require("openssl")
    local cipher = openssl.cipher

    local function bin_to_hex(s)
        return (s:gsub('.', function(c)
        return string.format('%02X', string.byte(c))
        end))
    end
    
    local function hex_to_bin(hex)
        return (hex:gsub('..', function(byte)
        return string.char(tonumber(byte, 16))
        end))
    end

    
    local key = hex_to_bin(key_hex)
    local encrypted = hex_to_bin(encrypted_hex)
    local i_v = hex_to_bin(iv)

    local dec = cipher.get("aes-128-cbc"):new(key, iv, false)
    local decrypted = dec:update(encrypted)
    decrypted = decrypted .. dec:final()
    return tostring(decrypted)
end