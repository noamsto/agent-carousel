#!/usr/bin/env bash
# Rewrite d2's embedded @font-face family names in an SVG to an installed
# family, and inject weight/style so bold/italic labels pick the right face.
# resvg ignores @font-face, so without this every text label is dropped.
set -euo pipefail

svg="$1"
family="${AGENT_CAROUSEL_D2_FONT:-Noto Sans}"

sed -E \
	-e "s/d2-[0-9]+-font-[a-z]+/${family}/g" \
	-e 's/(\.text-bold \{)/\1font-weight:bold;/g' \
	-e 's/(\.text-italic \{)/\1font-style:italic;/g' \
	"$svg" >"$svg.tmp"
mv "$svg.tmp" "$svg"
