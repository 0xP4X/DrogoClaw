# Contributing to DrogonClaw

Thanks for taking the time to contribute to DrogonClaw! Before you send a PR, there are three architectural invariants you need to understand.

## 0. Where to find work

We track open work on the GitHub Issues page. 

If you're a first-time contributor, filter by the **`good first issue`** label. These are generally scoped to a single LangChain tool module (e.g., adding a new tool adapter in `src/skills/pentest/`, ~50 LOC each).

### How to claim a task

1. Pick an open issue from the backlog.
2. Drop a comment saying you are picking it up. This prevents two people from implementing the same exploit or tool wrapper in parallel.
3. Fork, branch off `main`, implement, and open a PR.

---

## 1. The "Unrestricted Shell" Invariant (Rule 0x01)

**DrogonClaw is an offensive weapon, not a corporate scanner. Do not build cages.**

Many AI pentesting tools strip shell metacharacters (`|`, `>`, `&`, `$()`) out of fear that the LLM will break something. DrogonClaw trusts the LLM. 

If your PR introduces an execution environment, tool, or command adapter:
- **Do not sanitize valid bash logic.** If the AI wants to pipe `curl` into `bash`, or write a Python script via `EOF`, it must be allowed to.
- **Do not hardcode allowlists.** We do not block `rm`, `kill`, or `drop`. We rely on the Human-in-the-Loop (HitL) checkpoint to protect the operator, not artificial sandbox constraints.

If your PR introduces artificial guardrails that stop the AI from acting like a real hacker, it will be rejected.

---

## 2. The "OPSEC Cleanup" Invariant (Rule 0x02)

**If your feature leaves a footprint, you must wipe it.**

DrogonClaw is designed for stealth during Red Team engagements. We recently implemented the `CleanupRegistry` to guarantee zero-trace exits.

If your PR adds a feature that:
- Drops a payload on disk
- Modifies a firewall rule
- Opens a proxy (e.g., Ligolo/Chisel)
- Spawns a background listener

You **MUST** register a cleanup hook for it.

```typescript
// ✅ Good - Leaves no trace
CleanupRegistry.getInstance().register("rm -f /tmp/payload.bin", "Remove payload");

// ❌ Bad - Leaves artifacts behind for Blue Teams to find
fs.writeFileSync("/tmp/payload.bin", exploitData);
```

PRs that introduce persistent artifacts on the target or host without a cleanup hook will not pass review.

---

## 3. The "Decoupled Memory" Invariant (Rule 0x03)

**Do not flood the LLM Context Window.**

DrogonClaw operates autonomously. If it runs an Nmap scan that returns 10,000 open ports, pushing that into the LangChain memory graph will instantly burn the user's API budget and cause the model to hallucinate.

If you are writing a new Reconnaissance tool:
- **Do not** return massive JSON blobs or text dumps directly to the AI.
- **Do** parse the output and use the `LootDB` (SQLite) to store the data asynchronously.
- Return a summary to the AI: `"[Success] Discovered 500 open ports. They have been saved to the Loot Database."`

---

## Reporting security issues

Do not open public issues for zero-days or vulnerabilities you find in the underlying infrastructure of this project. Email the maintainers directly.

Happy hacking.
