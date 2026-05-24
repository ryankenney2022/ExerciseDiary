#!/usr/bin/env python3
"""Parse Guide-5A PRT score tables into a structured JSON the Go app embeds.

Uses pdfplumber's table extraction, which preserves the visual row/column
layout — unlike pdftotext -layout which scrambles tables that have header
rows spanning multiple printed rows.

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
from pathlib import Path

import pdfplumber

HERE = Path(__file__).parent
PDF = HERE / "Guide-5A_PRT.pdf"
OUT = HERE / "score_tables.json"

# Pages in the PDF (1-indexed). Tables for the low-altitude section live on
# pages 21-32; high-altitude tables live on pages 33-44. We just iterate every
# page in the document and let the table-header text tell us which bracket
# we're looking at — that's more robust than pinning page numbers.
SEX_AGE_RE = re.compile(
    r"(Males|Females):\s*Age\s*(\d+)\s*(?:-\s*(\d+)|(\+))", re.IGNORECASE
)

ALT_LOW_MARKER = "Altitudes Less Than"
ALT_HIGH_MARKER = "Altitudes Greater Than"


def mmss_to_seconds(s: str) -> int | None:
    s = (s or "").strip()
    if not s:
        return None
    m = re.match(r"^(\d+):(\d{2})$", s)
    if not m:
        return None
    return int(m.group(1)) * 60 + int(m.group(2))


def int_or_none(s: str) -> int | None:
    s = (s or "").strip()
    if not s:
        return None
    try:
        return int(s)
    except ValueError:
        return None


def parse_one_subtable(rows: list[list[str]], sex_age_header: str) -> dict:
    """Parse one (sex × age) sub-table from a pdfplumber-extracted page table.

    rows is the slice of full-page rows belonging to one sex×age block,
    starting with the header rows and ending before the next header (or
    end of page).

    Each data row has columns:
      [Category, Level, Points, Push-ups, Plank, Run, Row, 500yd swim, 450m swim]
    """
    # Identify sex
    m = SEX_AGE_RE.search(sex_age_header)
    if not m:
        return None
    sex = "M" if m.group(1).lower().startswith("m") else "F"
    age_min = int(m.group(2))
    age_max = int(m.group(3)) if m.group(3) else 999

    pushups_rows = []
    plank_rows = []
    run_rows = []

    for r in rows:
        # Pad to expected width
        cells = [(c or "").strip() for c in r]
        if len(cells) < 6:
            continue
        cat, level, points_s, pushups_s, plank_s, run_s = cells[:6]

        # Skip header rows like ["Category", "Level", "", ...] or the title row
        if cat.lower() == "category" or cat.lower() == "performance":
            continue
        points = int_or_none(points_s)
        if points is None:
            continue
        if cat == "" and level == "":
            continue  # blank row

        # Probationary has no Level cell.
        category = cat
        lvl = level if level else ""

        if (pu := int_or_none(pushups_s)) is not None:
            pushups_rows.append({
                "score": points, "category": category, "level": lvl, "raw": pu,
            })
        if (pl := mmss_to_seconds(plank_s)) is not None:
            plank_rows.append({
                "score": points, "category": category, "level": lvl, "raw": pl,
            })
        if (rn := mmss_to_seconds(run_s)) is not None:
            run_rows.append({
                "score": points, "category": category, "level": lvl, "raw": rn,
            })

    return {
        "sex": sex, "age_min": age_min, "age_max": age_max,
        "events": {
            "pushups": pushups_rows,
            "plank": plank_rows,
            "run": run_rows,
        },
    }


def split_subtables(rows: list[list[str]]) -> list[tuple[str, list[list[str]]]]:
    """A page may contain TWO sub-tables (Males then Females, same age bracket).
    Split into a list of (header_text, sub_rows)."""
    out = []
    current_header = None
    current_rows = []

    for r in rows:
        joined = " ".join((c or "") for c in r)
        if SEX_AGE_RE.search(joined):
            if current_header is not None:
                out.append((current_header, current_rows))
            current_header = joined
            current_rows = []
        elif current_header is not None:
            current_rows.append(r)

    if current_header is not None:
        out.append((current_header, current_rows))
    return out


def main():
    out_records = []
    current_altitude = None

    with pdfplumber.open(PDF) as pdf:
        for page in pdf.pages:
            text = page.extract_text() or ""
            if ALT_LOW_MARKER in text and not text.lower().startswith("section 4-2"):
                # Update altitude based on first marker found on the page.
                if ALT_HIGH_MARKER in text:
                    # Both markers — high comes later in the document.
                    current_altitude = "high" if current_altitude == "high" else "low"
                else:
                    current_altitude = "low"
            elif ALT_HIGH_MARKER in text:
                current_altitude = "high"

            if current_altitude is None:
                continue

            tables = page.extract_tables() or []
            for tbl in tables:
                for header, sub in split_subtables(tbl):
                    parsed = parse_one_subtable(sub, header)
                    if not parsed:
                        continue
                    for event, rows in parsed["events"].items():
                        if not rows:
                            continue
                        out_records.append({
                            "sex": parsed["sex"],
                            "age_min": parsed["age_min"],
                            "age_max": parsed["age_max"],
                            "altitude": current_altitude,
                            "event": event,
                            "rows": rows,
                        })

    OUT.write_text(json.dumps(out_records, indent=2))
    print(f"Wrote {len(out_records)} (sex/age/altitude/event) tables to {OUT}")

    # Sanity: dump the M 45-49 low altitude pushups table
    for t in out_records:
        if t["sex"] == "M" and t["age_min"] == 45 and t["altitude"] == "low" and t["event"] == "pushups":
            print("\nM 45-49 low altitude pushups:")
            for r in t["rows"]:
                print(f"  {r['score']:3d} pts ({r['category']} {r['level']}): {r['raw']} reps")


if __name__ == "__main__":
    main()
