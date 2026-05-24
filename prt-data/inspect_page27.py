"""Render page 27 of the Navy PRT Guide-5A as a structured table to see the
actual visual layout (rather than pdftotext's line-by-line guess)."""

import pdfplumber

PDF = r"C:\Users\rkenn\ClaudeCodeProjects\ExerciseDiary\prt-data\Guide-5A_PRT.pdf"

with pdfplumber.open(PDF) as pdf:
    # pdfplumber pages are 0-indexed; user said "page 27" (likely 1-indexed)
    for page_num in (26, 27, 28):  # 0-indexed → covers what the cover page calls 27
        page = pdf.pages[page_num]
        print(f"\n========== PDF page index {page_num} (printed page label: {page_num + 1}) ==========")
        tables = page.extract_tables()
        print(f"Found {len(tables)} tables")
        for ti, t in enumerate(tables):
            print(f"\n--- Table #{ti} ---")
            for row in t:
                # Filter to non-empty cells
                cells = [c if c else "" for c in row]
                print("  | " + " | ".join(cells) + " |")
