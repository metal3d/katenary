""" Fix the markdown files to replace code blocs by lists when the code blocs are lists."""

import re
import sys
from typing import Tuple

# get markdown bloc code
re_code = re.compile(r"```(.*?)```", re.DOTALL)


def fix(text: str) -> Tuple[str, bool]:
    """Fix the markdown text to replace code blocs by lists when the code blocs are lists."""
    # in the text, get the code blocs
    code_blocs = re_code.findall(text)
    # for each code bloc, if lines begin by a "-", this is a list. So,
    # make it a mkdocs list and remove the block code
    fixed = False
    for code in code_blocs:
        lines = code.split("\n")
        lines = [line.strip() for line in lines if line.strip()]
        if all(line.startswith("-") for line in lines):
            fixed = True
            # make a mkdocs list
            lines = [f"- {line[1:]}" for line in lines]
            # replace the code bloc by the list
            text = text.replace(f"```{code}```", "\n".join(lines))
    return text, fixed


def main(filename: str):
    """Fix and rewrite the markdown file."""
    with open(filename, "r", encoding="utf-8") as f:
        text = f.read()
    content, fixed = fix(text)

    if not fixed:
        return

    with open(sys.argv[1], "w", encoding="utf-8") as f:
        f.write(content)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python fix.py <file>")
        sys.exit(1)

    main(sys.argv[1])
