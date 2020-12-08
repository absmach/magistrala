// +build !noasm

package meow

// cpu contains feature flags relevant to selecting a Meow implementation.
var cpu struct {
	HasOSXSAVE    bool
	HasAES        bool
	HasVAES       bool
	HasAVX        bool
	HasAVX512F    bool
	HasAVX512VL   bool
	HasAVX512DQ   bool
	EnabledAVX    bool
	EnabledAVX512 bool
}

func init() {
	determineCPUFeatures()

	switch {
	case cpu.HasVAES && cpu.HasAVX512F && cpu.EnabledAVX512:
		implementation = "vaes-512"
		checksum = checksum512
		blocks = blocks512
		finish = finish128
	case cpu.HasAES && cpu.HasAVX && cpu.EnabledAVX:
		// AVX required for VEX-encoded AES instruction, which allows non-aligned memory addresses.
		implementation = "aes-ni"
		checksum = checksum128
		blocks = blocks128
		finish = finish128
	}
}

// AES-NI implementation.
func checksum128(seed uint64, dst, src []byte)
func blocks128(s, src []byte)
func finish128(seed uint64, s, dst, rem, trail []byte, length uint64)

// VAES-256 implementation.
func checksum256(seed uint64, dst, src []byte)
func blocks256(s, src []byte)

// VAES-512 implementation.
func checksum512(seed uint64, dst, src []byte)
func blocks512(s, src []byte)

// determineCPUFeatures populates flags in global cpu variable by querying CPUID.
func determineCPUFeatures() {
	maxID, _, _, _ := cpuid(0, 0)
	if maxID < 1 {
		return
	}

	_, _, ecx1, _ := cpuid(1, 0)
	cpu.HasOSXSAVE = isSet(ecx1, 27)
	cpu.HasAES = isSet(ecx1, 25)
	cpu.HasAVX = isSet(ecx1, 28)

	if cpu.HasOSXSAVE {
		eax, _ := xgetbv()
		cpu.EnabledAVX = (eax & 0x6) == 0x6
		cpu.EnabledAVX512 = (eax & 0xe0) == 0xe0
	}

	if maxID < 7 {
		return
	}
	_, ebx7, ecx7, _ := cpuid(7, 0)
	cpu.HasVAES = isSet(ecx7, 9)
	cpu.HasAVX512F = isSet(ebx7, 16)
	cpu.HasAVX512VL = isSet(ebx7, 31)
	cpu.HasAVX512DQ = isSet(ebx7, 17)
}

// cpuid executes the CPUID instruction with the given EAX, ECX inputs.
func cpuid(eaxArg, ecxArg uint32) (eax, ebx, ecx, edx uint32)

// xgetbv executes the XGETBV instruction.
func xgetbv() (eax, edx uint32)

// isSet determines if bit i of x is set.
func isSet(x uint32, i uint) bool {
	return (x>>i)&1 == 1
}
