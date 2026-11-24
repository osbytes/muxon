import pathlib
import sys

path = pathlib.Path("answer.txt")
if not path.exists():
    sys.exit(1)
sys.exit(0 if path.read_text().strip() == "5" else 1)
