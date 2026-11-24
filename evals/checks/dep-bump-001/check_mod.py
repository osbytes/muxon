import pathlib
import sys

text = pathlib.Path("go.mod").read_text()
if "github.com/google/uuid" not in text:
    sys.exit(1)
if "github.com/google/uuid v1.6.0" in text:
    sys.exit(1)
sys.exit(0)
