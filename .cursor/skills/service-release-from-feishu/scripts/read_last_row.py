#!/usr/bin/env python3
import json
import subprocess
import sys


SPREADSHEET_TOKEN = "HEyssqGClhz92ztmE1ucn6yNnti"
SHEET_ID = "4397b9"
DEFAULT_RANGE = f"{SHEET_ID}!A1:J700"


def run_lark_cli():
    cmd = [
        "lark-cli",
        "sheets",
        "+read",
        "--spreadsheet-token",
        SPREADSHEET_TOKEN,
        "--range",
        DEFAULT_RANGE,
        "--as",
        "user",
        "--value-render-option",
        "ToString",
    ]
    result = subprocess.run(cmd, text=True, capture_output=True, check=False)
    if result.returncode != 0:
        raise SystemExit(result.stderr or result.stdout or "lark-cli failed")
    return json.loads(result.stdout)


def main():
    payload = run_lark_cli()
    values = payload["data"]["valueRange"]["values"]
    if not values:
        raise SystemExit("sheet returned no rows")

    header = values[0]
    last = None
    for row_index, row in enumerate(values[1:], start=2):
        if any(cell not in (None, "") for cell in row):
            last = (row_index, row)

    if not last:
        raise SystemExit("no non-empty data row found")

    row_index, row = last
    mapped = {
        header[i]: row[i] if i < len(row) else None
        for i in range(len(header))
    }
    print(json.dumps({"row_index": row_index, "row": mapped}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    try:
        main()
    except KeyError as exc:
        print(f"unexpected lark-cli response shape: missing {exc}", file=sys.stderr)
        raise
