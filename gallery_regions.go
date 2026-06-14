package main

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"
)

// region is one navigable diagram object, bbox in source fractions (0..1).
type region struct {
	path           string
	x0, y0, x1, y1 float64
}

func (r region) cx() float64 { return (r.x0 + r.x1) / 2 }
func (r region) cy() float64 { return (r.y0 + r.y1) / 2 }

var (
	svgGroupRe    = regexp.MustCompile(`<g class="([A-Za-z0-9+/=]+)"[^>]*>`)
	svgInnerSVGRe = regexp.MustCompile(`<svg[^>]+class="[^"]*d2-svg[^"]*"[^>]+viewBox="(-?[\d.]+) (-?[\d.]+) (-?[\d.]+) (-?[\d.]+)"`)
	svgViewBoxRe  = regexp.MustCompile(`viewBox="(-?[\d.]+) (-?[\d.]+) (-?[\d.]+) (-?[\d.]+)"`)
	svgDAttrRe    = regexp.MustCompile(`d="([^"]*)"`)
	connPathRe    = regexp.MustCompile(`^\(.*\)\[\d+\]$`)
	diagramIDRe   = regexp.MustCompile(`^d2-\d+`)
)

// parseRegions extracts each container/shape group as a region with a normalized
// bbox. Groups whose class isn't a decodable dotted object path, connection
// groups, and the diagram-id class are skipped. Returns nil if nothing parses.
func parseRegions(data []byte) []region {
	s := string(data)

	// D2 wraps all diagram content in an inner <svg class="d2-... d2-svg" viewBox="minX minY w h">.
	// That viewBox defines the coordinate space used by all path geometry.
	// We must not use the outer <svg> viewBox (0 0 w h) or any <marker> viewBox.
	var minX, minY, vbW, vbH float64
	if vb := svgInnerSVGRe.FindStringSubmatch(s); vb != nil {
		minX, _ = strconv.ParseFloat(vb[1], 64)
		minY, _ = strconv.ParseFloat(vb[2], 64)
		vbW, _ = strconv.ParseFloat(vb[3], 64)
		vbH, _ = strconv.ParseFloat(vb[4], 64)
	} else {
		// Fallback: use the first viewBox found (plain D2, no sketch overlay).
		if vb2 := svgViewBoxRe.FindStringSubmatch(s); vb2 != nil {
			minX, _ = strconv.ParseFloat(vb2[1], 64)
			minY, _ = strconv.ParseFloat(vb2[2], 64)
			vbW, _ = strconv.ParseFloat(vb2[3], 64)
			vbH, _ = strconv.ParseFloat(vb2[4], 64)
		}
	}
	if vbW <= 0 || vbH <= 0 {
		return nil
	}

	// Only base64-decodable groups are real d2 nodes/connections; style groups
	// like class="shape" are skipped. These bound each region's geometry segment.
	type grp struct {
		path       string
		contentEnd int // end of the opening <g ...> tag
		matchStart int // start of the <g ...> tag
	}
	var groups []grp
	for _, loc := range svgGroupRe.FindAllStringSubmatchIndex(s, -1) {
		dec, err := base64.StdEncoding.DecodeString(s[loc[2]:loc[3]])
		if err != nil {
			continue
		}
		groups = append(groups, grp{
			path:       string(dec),
			contentEnd: loc[1],
			matchStart: loc[0],
		})
	}

	var out []region
	for i, g := range groups {
		if g.path == "" || connPathRe.MatchString(g.path) || diagramIDRe.MatchString(g.path) || !isObjectPath(g.path) {
			continue
		}
		segEnd := len(s)
		if i+1 < len(groups) {
			segEnd = groups[i+1].matchStart
		}
		x0, y0, x1, y1, ok := groupBBox(s[g.contentEnd:segEnd])
		if !ok {
			continue
		}
		out = append(out, region{
			path: g.path,
			x0:   (x0 - minX) / vbW,
			y0:   (y0 - minY) / vbH,
			x1:   (x1 - minX) / vbW,
			y1:   (y1 - minY) / vbH,
		})
	}
	return out
}

// isObjectPath rejects junk classes; a d2 object path is dot-separated non-empty
// segments. (Connections/diagram-id are filtered separately by the caller.)
func isObjectPath(p string) bool {
	for _, seg := range strings.Split(p, ".") {
		if seg == "" {
			return false
		}
	}
	return true
}

// groupBBox unions the bbox of every <path d>, <rect>, <ellipse>, <circle> in
// the fragment.
func groupBBox(frag string) (minX, minY, maxX, maxY float64, ok bool) {
	minX, minY = 1e18, 1e18
	maxX, maxY = -1e18, -1e18
	merge := func(a, b, c, d float64, has bool) {
		if !has {
			return
		}
		ok = true
		if a < minX {
			minX = a
		}
		if b < minY {
			minY = b
		}
		if c > maxX {
			maxX = c
		}
		if d > maxY {
			maxY = d
		}
	}
	for _, m := range svgDAttrRe.FindAllStringSubmatch(frag, -1) {
		a, b, c, d, has := pathBBox(m[1])
		merge(a, b, c, d, has)
	}
	merge(rectBBox(frag))
	return
}

var (
	rectElemRe    = regexp.MustCompile(`<rect\b([^>]*)>`)
	ellipseElemRe = regexp.MustCompile(`<(?:ellipse|circle)\b([^>]*)>`)
	numAttrRe     = regexp.MustCompile(`\b([\w-]+)="(-?[\d.]+)"`)
)

// parseAttrs returns a map of attribute name → float64 for a tag's attribute string.
func parseAttrs(attrs string) map[string]float64 {
	m := make(map[string]float64)
	for _, match := range numAttrRe.FindAllStringSubmatch(attrs, -1) {
		v, _ := strconv.ParseFloat(match[2], 64)
		m[match[1]] = v
	}
	return m
}

// rectBBox unions any <rect>/<ellipse>/<circle> bbox found in the fragment.
// Attribute lookup is scoped to each element tag to avoid matching unrelated
// elements (e.g. <text x="..."> in the same fragment).
func rectBBox(frag string) (minX, minY, maxX, maxY float64, ok bool) {
	minX, minY = 1e18, 1e18
	maxX, maxY = -1e18, -1e18
	merge := func(x0, y0, x1, y1 float64) {
		ok = true
		if x0 < minX {
			minX = x0
		}
		if y0 < minY {
			minY = y0
		}
		if x1 > maxX {
			maxX = x1
		}
		if y1 > maxY {
			maxY = y1
		}
	}
	for _, m := range rectElemRe.FindAllStringSubmatch(frag, -1) {
		a := parseAttrs(m[1])
		if _, hasX := a["x"]; hasX {
			merge(a["x"], a["y"], a["x"]+a["width"], a["y"]+a["height"])
		}
	}
	for _, m := range ellipseElemRe.FindAllStringSubmatch(frag, -1) {
		a := parseAttrs(m[1])
		if _, hasCX := a["cx"]; hasCX {
			rx := a["rx"]
			if rx == 0 {
				rx = a["r"]
			}
			ry := a["ry"]
			if ry == 0 {
				ry = rx
			}
			merge(a["cx"]-rx, a["cy"]-ry, a["cx"]+rx, a["cy"]+ry)
		}
	}
	if !ok {
		return 0, 0, 0, 0, false
	}
	return
}
