#!/usr/bin/env python
"""
Generate INSERT statements for the acos table from a CSV with columns: cms_id, name, and optional termination_details.

Usage:
  python scripts/csv_to_acos_insert.py path/to/your.csv

Output: SQL statements to stdout. Pipe to psql or save to a file:
  python scripts/csv_to_acos_insert.py acos.csv > acos_insert.sql
  psql $DATABASE_URL -f acos_insert.sql

Requirements: CSV with header row containing cms_id and name (order does not matter).
Optional header: termination_details (supports M/DD/YY, M/DD/YYYY, and YYYY-MM-DD formats).
Works with Python 3.
"""

from __future__ import print_function

import csv
import datetime
import io
import sys
import uuid


# Strip BOM and whitespace from header names (Excel often saves CSV with UTF-8 BOM).
def _norm_header(name):
    if name is None:
        return ""
    if isinstance(name, bytes):
        name = name.decode("utf-8", "replace")
    return name.strip().lstrip("\ufeff").strip()


def escape_sql(s):
    """Escape single quotes for SQL literal."""
    if isinstance(s, bytes):
        s = s.decode("utf-8", "replace")
    return s.replace("'", "''")


def parse_date(date_str):
    """Parse date from string supporting M/DD/YY, M/DD/YYYY, and YYYY-MM-DD."""
    date_str = date_str.strip()
    if not date_str:
        return None
    for fmt in ("%m/%d/%y", "%m/%d/%Y", "%Y-%m-%d"):
        try:
            return datetime.datetime.strptime(date_str, fmt)
        except ValueError:
            continue
    raise ValueError("Could not parse date: {}".format(date_str))


def main():
    if len(sys.argv) != 2:
        print(__doc__, file=sys.stderr)
        sys.exit(1)

    path = sys.argv[1]
    rows = []
    # utf-8-sig strips BOM so headers are "cms_id" / "name" and avoids Python 2 encode errors
    with io.open(path, "r", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        raw_fields = reader.fieldnames or []
        # Normalize: strip BOM and whitespace so "\ufeffcms_id" or " cms_id " still matches
        norm_to_raw = {_norm_header(f): f for f in raw_fields}
        if "cms_id" not in norm_to_raw or "name" not in norm_to_raw:
            print("Error: CSV must have headers 'cms_id' and 'name' (got: {0})".format(raw_fields), file=sys.stderr)
            sys.exit(1)
        cms_id_key = norm_to_raw["cms_id"]
        name_key = norm_to_raw["name"]
        term_details_key = norm_to_raw.get("termination_details")

        for row in reader:
            cms_id = (row.get(cms_id_key) or "").strip()
            name = (row.get(name_key) or "").strip()
            if not cms_id or not name:
                continue
            term_details_val = ""
            if term_details_key:
                term_details_val = (row.get(term_details_key) or "").strip()
            rows.append((cms_id, name, term_details_val))

    if not rows:
        print("Error: No data rows found", file=sys.stderr)
        sys.exit(1)

    # acos: uuid, cms_id, client_id, name, termination_details (id/created_at/updated_at are DB defaults)
    # client_id is set to the same value as uuid (string), matching admin_create_aco Lambda behavior
    print("-- Direct insert into acos from CSV. Run with: psql $DATABASE_URL -f thisfile.sql")
    print("INSERT INTO acos (uuid, cms_id, client_id, name, termination_details) VALUES")

    values = []
    for cms_id, name, term_details_val in rows:
        u = str(uuid.uuid4())
        # cms_id is type text in schema
        cms_id_esc = escape_sql(cms_id)
        name_esc = escape_sql(name)

        term_json_literal = "NULL"
        if term_details_val:
            try:
                date_obj = parse_date(term_details_val)
                if date_obj:
                    # Format as ISO 8601 string (end of the day)
                    iso_str = date_obj.strftime("%Y-%m-%dT23:59:59Z")
                    term_json_literal = "'{{\"TerminationDate\": \"{0}\", \"CutoffDate\": \"{0}\", \"DenylistType\": 0, \"BlacklistType\": 0}}'".format(iso_str)
            except ValueError as e:
                print("Warning: {}".format(e), file=sys.stderr)

        values.append("  ('{0}', '{1}', '{0}', '{2}', {3})".format(u, cms_id_esc, name_esc, term_json_literal))

    print(",\n".join(values))
    print(";")


if __name__ == "__main__":
    main()
