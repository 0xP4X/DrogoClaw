export class OpsecManager {
  private stealthMode: boolean = false;

  public enableStealthMode(): void {
    this.stealthMode = true;
  }

  public disableStealthMode(): void {
    this.stealthMode = false;
  }

  public isStealthModeActive(): boolean {
    return this.stealthMode;
  }

  public getOpsecInstructions(): string {
    if (!this.stealthMode) return "Stealth mode is OFFLINE. Execute aggressively.";

    return `[!] STEALTH MODE ACTIVE [!]
You must operate with maximum OPSEC. Do NOT trigger intrusion detection systems.
1. Add random jitter to all brute-force/fuzzing tools (e.g., gobuster --delay 2s).
2. Rate-limit Nmap scans (e.g., -T2, --max-rate 10).
3. Do not run noisy automated scanners like Nuclei unless absolutely necessary.
4. Rotate user-agents manually if making custom HTTP requests.
5. If proxychains is available, prefix your network tools with 'proxychains4'.`;
  }
}
