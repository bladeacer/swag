#!/bin/sh
# Write a SubRip fixture with ten thousand cues. The external performance
# tools use it, so every one of them measures the same document.
#
# Usage: sh scripts/bench-fixture.sh [path]
set -eu

out=${1:-.bench/bench.srt}
mkdir -p "$(dirname "$out")"

awk 'BEGIN{
  for (i = 1; i <= 10000; i++) {
    start = i * 2 - 2;
    end = start + 1;
    printf "%d\n", i;
    printf "%02d:%02d:%02d,000 --> %02d:%02d:%02d,000\n",
      int(start/3600), int(start%3600/60), start%60,
      int(end/3600), int(end%3600/60), end%60;
    printf "A subtitle line with some words in it.\n\n";
  }
}' > "$out"

echo "wrote $out"
