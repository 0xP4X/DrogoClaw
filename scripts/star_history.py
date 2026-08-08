#!/usr/bin/env python3
"""Generate a static Star History SVG for the DrogonClaw repo.

GitHub restricted public access to the stargazers API on 2026-06-30, so the
live star-history.com embed no longer renders in the README. This script fetches
the (owner-only) star timestamps via `gh` and renders a self-contained SVG that
is committed to the repo, so the chart always appears without exposing a token.

Usage:
    GITHUB_REPO=0xP4X/drogonclaw python3 scripts/star_history.py [output.svg]
"""

import json
import os
import subprocess
import sys
from datetime import datetime, timezone

REPO = os.environ.get("GITHUB_REPO", "0xP4X/drogonclaw")
OUT = sys.argv[1] if len(sys.argv) > 1 else "assets/star-history.svg"

W, H = 800, 360
PAD_L, PAD_R, PAD_T, PAD_B = 56, 24, 24, 40


def fetch_stars(repo: str):
    stars = []
    page = 1
    while True:
        cmd = [
            "gh", "api",
            "-H", "Accept: application/vnd.github.star+json",
            f"/repos/{repo}/stargazers?per_page=100&page={page}",
        ]
        out = subprocess.run(cmd, capture_output=True, text=True)
        if out.returncode != 0 or not out.stdout.strip():
            break
        batch = json.loads(out.stdout)
        if not batch:
            break
        for item in batch:
            ts = item.get("starred_at")
            if ts:
                stars.append(datetime.fromisoformat(ts.replace("Z", "+00:00")))
        page += 1
        if len(batch) < 100:
            break
    return sorted(stars)


def build_svg(stars):
    if not stars:
        stars = [datetime.now(timezone.utc)]
    first, last = stars[0], stars[-1]
    if (last - first).total_seconds() < 1:
        last = first
    span = max((last - first).total_seconds(), 1.0)
    total = len(stars)

    plot_w = W - PAD_L - PAD_R
    plot_h = H - PAD_T - PAD_B

    def x(i):
        t = stars[i]
        frac = (t - first).total_seconds() / span
        return PAD_L + frac * plot_w

    def y(c):
        frac = c / max(total, 1)
        return PAD_T + (1 - frac) * plot_h

    pts = " ".join(f"{x(i):.1f},{y(i + 1):.1f}" for i in range(total))

    # x axis ticks (start / end dates)
    fmt = "%Y-%m-%d"
    x0_lab = first.strftime(fmt)
    x1_lab = last.strftime(fmt)

    # y axis ticks
    y_ticks = ""
    for c in (0, max(total // 2, 1), total):
        ty = y(c)
        y_ticks += (
            f'<line x1="{PAD_L}" y1="{ty:.1f}" x2="{W - PAD_R}" y2="{ty:.1f}" '
            f'stroke="#eee" stroke-width="1"/>'
            f'<text x="{PAD_L - 8}" y="{ty + 4:.1f}" text-anchor="end" '
            f'font-size="12" fill="#666">{c}</text>'
        )

    return f'''<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" font-family="sans-serif">
  <rect width="{W}" height="{H}" fill="#fff"/>
  <text x="{W/2:.0f}" y="20" text-anchor="middle" font-size="16" font-weight="bold" fill="#333">Star History \u2014 {REPO}</text>
  <line x1="{PAD_L}" y1="{PAD_T}" x2="{PAD_L}" y2="{H - PAD_B}" stroke="#333" stroke-width="1"/>
  <line x1="{PAD_L}" y1="{H - PAD_B}" x2="{W - PAD_R}" y2="{H - PAD_B}" stroke="#333" stroke-width="1"/>
  {y_ticks}
  <polyline fill="none" stroke="#f1c40f" stroke-width="2.5" points="{pts}"/>
  <circle cx="{x(total-1):.1f}" cy="{y(total):.1f}" r="3.5" fill="#f1c40f"/>
  <text x="{PAD_L}" y="{H - PAD_B + 24}" font-size="12" fill="#666">{x0_lab}</text>
  <text x="{W - PAD_R}" y="{H - PAD_B + 24}" text-anchor="end" font-size="12" fill="#666">{x1_lab}</text>
  <text x="{W - PAD_R}" y="20" text-anchor="end" font-size="12" fill="#999">{total} \u2b50</text>
</svg>
'''


def main():
    stars = fetch_stars(REPO)
    svg = build_svg(stars)
    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    with open(OUT, "w") as f:
        f.write(svg)
    print(f"Wrote {OUT} ({len(stars)} stars)")


if __name__ == "__main__":
    main()
