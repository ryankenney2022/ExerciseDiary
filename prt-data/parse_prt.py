#!/usr/bin/env python3
"""Parse the pdftotext -layout output of the Navy PRT Guide-5A into structured JSON.

Output schema (one object per (sex, age_bracket, altitude, event)):
{
  "sex": "M" | "F",
  "age_min": int,
  "age_max": int,    # 999 for "65+"
  "altitude": "low" | "high",
  "event": "pushups" | "plank" | "run",
  "rows": [{"score": int, "category": str, "level": str, "raw": int}, ...]
}

For pushups: raw is the count.
For plank/run: raw is seconds (mm:ss converted).
"""

import json
import re
import sys
from pathlib import Path

TXT = Path(__file__).parent / "Guide-5A_PRT.txt"
OUT = Path(__file__).parent / "score_tables.json"

AGE_RE = re.compile(r"(Males|Females): Age (\d+)\s*-\s*(\d+)|(Males|Females): Age (\d+)\s*\+", re.IGNORECASE)

CATEGORIES = [
    ("Outstanding", "High"),
    ("Outstanding", "Medium"),
    ("Outstanding", "Low"),
    ("Excellent", "High"),
    ("Excellent", "Medium"),
    ("Excellent", "Low"),
    ("Good", "High"),
    ("Good", "Medium"),
    ("Good", "Low"),
    ("Satisfactory", "High"),
    ("Satisfactory", "Medium"),
    ("Probationary", ""),
]


def mmss_to_seconds(s: str) -> int:
    s = s.strip()
    if not s:
        return None
    m = re.match(r"^(\d+):(\d{2})$", s)
    if not m:
        return None
    return int(m.group(1)) * 60 + int(m.group(2))


def parse_table_block(lines: list[str]) -> dict:
    """Given the lines for one (sex × age) block, parse out per-event score rows.

    Approach: walk the lines; track current category/level as it appears in the
    leftmost columns. Each row has:
      [Category] [Level] [Points] [Pushups] [Plank mm:ss] [1.5mi mm:ss] [2km row] [500yd swim] [450m swim]
    Intermediate-points rows (e.g. 95, 85) only have Points + Pushups + (sometimes) 450m swim.
    """
    rows_pushups = []  # [{score, category, level, raw}]
    rows_plank = []
    rows_run = []

    cur_category = None
    cur_level = None

    # Strip well-known column-header words so the "Category Level 100 76 Planks run row swim 6:43"
    # header line still yields parseable data (the 100-pts row data is embedded there).
    HEADER_NOISE = re.compile(r"\b(Category|Level|Points|Push-\s*ups|Pushups|Forearm|Planks?|1\.5\s*-?\s*mile|2-?km|500\s*-?\s*yd|450\s*-?\s*m|row|swim|run|years)\b", re.IGNORECASE)

    for line in lines:
        if not line.strip():
            continue

        # Try to identify a category-level row first
        cat_match = None
        for cat, level in CATEGORIES:
            if level and re.match(rf"^\s*{cat}\s+{level}\b", line):
                cat_match = (cat, level)
                break
            if not level and re.match(rf"^\s*{cat}\b", line):
                cat_match = (cat, "")
                break

        body = line
        if cat_match:
            cur_category, cur_level = cat_match
            pat = rf"^\s*{cur_category}\s+{cur_level}\s*" if cur_level else rf"^\s*{cur_category}\s*"
            body = re.sub(pat, "", line, count=1)

        # Scrub the column-header words so they don't get parsed as tokens.
        body = HEADER_NOISE.sub(" ", body)

        # Split remaining body into whitespace-separated tokens. Order *should* be:
        #   [Points] [Pushups] [Plank] [Run] [Row] [500swim] [450swim]
        # but intermediate-points rows lack most of these.
        toks = body.split()

        # Find a numeric token at the start that looks like Points (in {45,50,55,...,100})
        points = None
        if toks and re.match(r"^\d+$", toks[0]):
            n = int(toks[0])
            if n in {45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100}:
                points = n
                toks = toks[1:]

        if points is None and not cat_match:
            # Neither a category row nor an intermediate-points row — skip
            continue

        # Points-only rows that precede the first category label fall into Outstanding.
        if points is not None and cur_category is None:
            cur_category = "Outstanding"
            cur_level = ""

        # For category rows without an explicit Points number, infer from category:
        if points is None and cat_match:
            cat_to_points = {
                ("Outstanding", "High"): 90,
                ("Outstanding", "Medium"): 80,
                ("Outstanding", "Low"): 70,
                ("Excellent", "High"): 60,
                ("Excellent", "Medium"): 50,
                ("Excellent", "Low"): 45,
                ("Good", "High"): 40,
                ("Good", "Medium"): 35,
                ("Good", "Low"): 30,
                ("Satisfactory", "High"): 25,
                ("Satisfactory", "Medium"): 20,
                ("Probationary", ""): 15,
            }
            points = cat_to_points.get((cur_category, cur_level))
            if points is None:
                continue

        # Now interpret remaining tokens.
        #  - integer (<200) → pushup count
        #  - mm:ss tokens are ONLY plank/run on category-level rows (Outstanding High, etc.);
        #    on intermediate-point rows (95, 85, ...), mm:ss values are always 450m swim — skip.
        pushup_raw = None
        plank_raw = None
        run_raw = None

        extract_mmss = cat_match is not None
        mmss_idx = 0
        for tok in toks:
            if re.match(r"^\d+$", tok):
                if pushup_raw is None and int(tok) < 200:
                    pushup_raw = int(tok)
            elif re.match(r"^\d+:\d{2}$", tok) and extract_mmss:
                if mmss_idx == 0:
                    plank_raw = mmss_to_seconds(tok)
                elif mmss_idx == 1:
                    run_raw = mmss_to_seconds(tok)
                # idx 2+ would be row/swim — skip
                mmss_idx += 1

        if pushup_raw is not None:
            rows_pushups.append({
                "score": points,
                "category": cur_category or "",
                "level": cur_level or "",
                "raw": pushup_raw,
            })
        if plank_raw is not None:
            rows_plank.append({
                "score": points,
                "category": cur_category or "",
                "level": cur_level or "",
                "raw": plank_raw,
            })
        if run_raw is not None:
            rows_run.append({
                "score": points,
                "category": cur_category or "",
                "level": cur_level or "",
                "raw": run_raw,
            })

    return {"pushups": rows_pushups, "plank": rows_plank, "run": rows_run}


def main():
    text = TXT.read_text(encoding="utf-8", errors="replace")
    lines = text.splitlines()

    # Find every "Males: Age X - Y" / "Females: Age X - Y" header line
    headers = []  # [(line_no, sex, age_min, age_max)]
    for i, line in enumerate(lines):
        m = AGE_RE.search(line)
        if not m:
            continue
        sex_a, age_min_a, age_max_a, sex_b, age_b = m.groups()
        if sex_a:
            sex = "M" if sex_a.lower().startswith("m") else "F"
            age_min = int(age_min_a)
            age_max = int(age_max_a)
        else:
            sex = "M" if sex_b.lower().startswith("m") else "F"
            age_min = int(age_b)
            age_max = 999
        headers.append((i, sex, age_min, age_max))

    # Split: low-altitude tables come before "Section 4-2"; high-altitude tables come after.
    # (TOC entries with "Greater Than 5000" appear earlier and must be skipped.)
    alt_split_line = None
    for i, line in enumerate(lines):
        if "Section 4-2" in line:
            alt_split_line = i
            break
    if alt_split_line is None:
        print("ERROR: could not find 'Section 4-2' header", file=sys.stderr)
        sys.exit(1)

    out = []
    for idx, (line_no, sex, age_min, age_max) in enumerate(headers):
        # Block goes from this header to the next header (or end of file)
        next_line = headers[idx + 1][0] if idx + 1 < len(headers) else len(lines)
        block = lines[line_no:next_line]
        altitude = "high" if line_no > alt_split_line else "low"
        parsed = parse_table_block(block)
        for event in ("pushups", "plank", "run"):
            out.append({
                "sex": sex,
                "age_min": age_min,
                "age_max": age_max,
                "altitude": altitude,
                "event": event,
                "rows": parsed[event],
            })

    OUT.write_text(json.dumps(out, indent=2))
    print(f"Wrote {len(out)} tables to {OUT}")
    # Sanity-print one table
    for t in out:
        if t["sex"] == "M" and t["age_min"] == 35 and t["altitude"] == "low" and t["event"] == "pushups":
            print("\nSample (M 35-39 low altitude, pushups):")
            for r in t["rows"]:
                print(f"  {r['score']} pts ({r['category']} {r['level']}): {r['raw']} reps")


if __name__ == "__main__":
    main()
