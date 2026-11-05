#!/usr/bin/env python3
"""
compile_names.py compiles name_ranking.parquet (~330,000 records) into
internal/anonymizer/detector/name_catalog.gz for HadesGate's deterministic PERSON detector.

Binary file layout (HGNM format):
  - Magic (4 bytes): 'HGNM'
  - Version (1 byte): 0x01
  - Total records (4 bytes uint32 little-endian)
  For each record (sorted alphabetically):
    - Name length L (1 byte uint8, max 18)
    - Name type flags (1 byte uint8: 1=given, 2=surname, 3=both)
    - Name ASCII bytes (L bytes)
    - Frequency given (4 bytes uint32 little-endian)
    - Frequency surname (4 bytes uint32 little-endian)
"""

import gzip
import os
import struct
import sys
import pyarrow.parquet as pq

def compile_catalog(parquet_path: str, output_path: str):
    if not os.path.exists(parquet_path):
        print(f"Error: Parquet file not found: {parquet_path}", file=sys.stderr)
        sys.exit(1)

    print(f"Reading {parquet_path}...")
    table = pq.read_table(parquet_path)
    pyd = table.to_pydict()
    names = pyd["name_text"]
    types = pyd["name_type"]
    freqs = pyd["frequency"]

    catalog = {}
    for n, t, f in zip(names, types, freqs):
        n = n.strip().lower()
        if not n:
            continue
        if n not in catalog:
            catalog[n] = [0, 0, 0]  # [flags, freq_given, freq_surname]
        if t == "given":
            catalog[n][0] |= 1
            catalog[n][1] = int(f)
        else:
            catalog[n][0] |= 2
            catalog[n][2] = int(f)

    sorted_names = sorted(catalog.keys())
    os.makedirs(os.path.dirname(output_path), exist_ok=True)

    print(f"Compiling {len(sorted_names)} unique names to {output_path}...")
    with gzip.open(output_path, "wb", compresslevel=9) as gz:
        gz.write(b"HGNM\x01")
        gz.write(struct.pack("<I", len(sorted_names)))
        for n in sorted_names:
            flags, fg, fs = catalog[n]
            nb = n.encode("ascii")
            gz.write(struct.pack("<BB", len(nb), flags))
            gz.write(nb)
            gz.write(struct.pack("<II", fg, fs))

    size_mb = os.path.getsize(output_path) / (1024 * 1024)
    print(f"Success! Output size: {size_mb:.2f} MB ({os.path.getsize(output_path)} bytes)")

if __name__ == "__main__":
    src = "name_ranking.parquet"
    dst = "internal/anonymizer/detector/name_catalog.gz"
    if len(sys.argv) > 1:
        src = sys.argv[1]
    if len(sys.argv) > 2:
        dst = sys.argv[2]
    compile_catalog(src, dst)

