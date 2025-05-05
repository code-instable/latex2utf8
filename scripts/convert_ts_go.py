# Script to transcript 
# 
# https://github.com/ojsheikh/unicode-latex/blob/master/src/latex.ts
# 
import re
import pyperclip
from pathlib import Path
import os

if not Path("latex.ts").exists():
        os.system(command="wget https://raw.githubusercontent.com/ojsheikh/unicode-latex/master/src/latex.ts")

with open('latex.ts', 'r') as ts_file:
    lines = ts_file.readlines()

go_map_lines = ['var latexSymbols = map[string]string{\n']
for line in lines:
    line = line.strip()
    if line.startswith("'"):
        key_value = line.rstrip(',').split(':', 1)
        if len(key_value) == 2:
            key = key_value[0].strip().strip("'").replace('\\', '')  # Remove '\\' from keys
            value = key_value[1].strip().strip("'")
            go_map_lines.append(f'    "{key}": "{value}",\n')
go_map_lines.append('}\n')

go_code = ''.join(go_map_lines)
pyperclip.copy(go_code)
print("Go map code copied to clipboard.")
