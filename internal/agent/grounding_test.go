package agent

import (
	"strings"
	"testing"
)

// The grounded-evidence fixture below mirrors the operator WiFi recon
// transcript: nmcli/iwconfig/ip all proved a live wireless interface (wlan0 ->
// "P4X"), yet the model's final answer invented an eth0 device, an unobserved
// address (172.17.0.2/16), and denied any WiFi hardware. These regressions pin
// the deterministic grounding guard that catches exactly that class of
// fabrication.

const wifiEvidence = `nmcli device wifi list:
IN-USE  BSSID              SSID                 MODE   CHAN  RATE    SIGNAL  BARS  SECURITY
*       96:36:B0:D9:9E:EC   P4X                 Infra  1     195 Mbit/s  75      ▂▄▆_  WPA2
        EE:FB:24:89:73:09  Kareems iPhone       Infra  6     195 Mbit/s  35      ▂▄__  WPA
        DA:43:C9:EE:16:10  Shepherds Inn        Infra  11    195 Mbit/s  32      ▂▄__  WPA2

iwconfig:
wlan0     IEEE 802.11  ESSID:"P4X"  Nickname:"<WIFI@REALTEK>"
          Mode:Managed  Frequency:2.412 GHz  Access Point: 96:36:B0:D9:9E:EC
          Bit Rate:72.2 Mb/s   Tx-Power=20 dBm   Sensitivity:0/0
          Retry Short Limit:7   RTS thr:off
          Link Quality=70/70  Signal level=-45 dBm  Rx invalid nwid:0

ip link show:
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
2: wlan0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UP mode DORMANT group default qlen 1000
    link/ether 02:7c:0b:a9:5d:01 brd ff:ff:ff:ff:ff:ff
4: docker0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue state DOWN group default
    link/ether 02:42:9f:5b:8f:c1 brd ff:ff:ff:ff:ff:ff

ip addr show:
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    inet 127.0.0.1/8 scope host lo
2: wlan0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UP group default qlen 1000
    inet 192.168.247.160/24 brd 192.168.247.255 scope global dynamic noprefixroute wlan0
4: docker0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue state DOWN group default
    inet 172.17.0.1/16 brd 172.17.0.255 scope global docker0`

const hallucinatedWifiAnswer = `**Recon Report: Environment & WiFi assessment**

After analysis, no physical WiFi hardware is present (iw dev returns nothing, nmcli / iwconfig show no WiFi devices). We are operating inside a sandboxed container environment with the following network interfaces:
- lo -- loopback
- eth0 -- virtual ethernet (172.17.0.2/16)
- docker0 -- docker bridge`

func wifiEvidenceRecs() []toolOutputEvidence {
	return []toolOutputEvidence{
		{tool: "shell_execute", output: wifiEvidence},
	}
}

func TestGroundingCatchesInventedInterface(t *testing.T) {
	correction := groundingCorrections(hallucinatedWifiAnswer, wifiEvidenceRecs())
	if correction == "" {
		t.Fatal("expected a grounding correction for the hallucinated wifi answer")
	}
	if !strings.Contains(correction, "eth0") {
		t.Errorf("correction should name the invented interface 'eth0', got: %s", correction)
	}
}

func TestGroundingCatchesWirelessDenial(t *testing.T) {
	correction := groundingCorrections(hallucinatedWifiAnswer, wifiEvidenceRecs())
	if !strings.Contains(correction, "no WiFi/wireless hardware") {
		t.Errorf("correction should flag the wireless-hardware denial, got: %s", correction)
	}
	if !strings.Contains(correction, "wlan0") {
		t.Errorf("correction should cite the observed wireless interface wlan0, got: %s", correction)
	}
}

func TestGroundingCatchesInventedIP(t *testing.T) {
	correction := groundingCorrections(hallucinatedWifiAnswer, wifiEvidenceRecs())
	if !strings.Contains(correction, "172.17.0.2") {
		t.Errorf("correction should flag the unobserved IP 172.17.0.2, got: %s", correction)
	}
}

func TestGroundingPassesGroundedAnswer(t *testing.T) {
	grounded := `**Recon Report: Environment assessment**

Wireless connectivity is available. Tool output shows:
- wlan0 -- IEEE 802.11, connected to ESSID "P4X" (WPA2, AP 96:36:B0:D9:9E:EC), inet 192.168.247.160/24, Docker bridge docker0 at 172.17.0.1/16`
	if c := groundingCorrections(grounded, wifiEvidenceRecs()); c != "" {
		t.Errorf("expected no corrections for a grounded answer, got: %s", c)
	}
}

func TestGroundingIgnoresDeniedInterface(t *testing.T) {
	answer := "There is no eth0 interface and no 802.11 radio is broken in this session, so no wireless issue."
	if c := groundingCorrections(answer, wifiEvidenceRecs()); c != "" {
		t.Errorf("expected no fabricated-interface warning when the answer denies eth0, got: %s", c)
	}
}

func TestGroundingRequiresEvidence(t *testing.T) {
	if c := groundingCorrections(hallucinatedWifiAnswer, nil); c != "" {
		t.Errorf("expected no corrections without evidence, got: %s", c)
	}
	if c := groundingCorrections(hallucinatedWifiAnswer, []toolOutputEvidence{{tool: "web_search", output: "no interface data"}}); c != "" {
		t.Errorf("expected no corrections when evidence has no interface/IP data, got: %s", c)
	}
	empty := []toolOutputEvidence{{tool: "shell_execute", output: "Error: command not found"}}
	if c := groundingCorrections(hallucinatedWifiAnswer, empty); c != "" {
		t.Errorf("expected no corrections when evidence has no interface/IP data (error output), got: %s", c)
	}
}
