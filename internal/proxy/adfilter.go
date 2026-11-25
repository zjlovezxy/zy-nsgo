package proxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	uriAttrRe = regexp.MustCompile(`URI="([^"]+)"`)
	extinfRe  = regexp.MustCompile(`^#EXTINF:([0-9]+(?:\.[0-9]+)?)`)
)

type m3u8Block struct {
	lines       []string
	durationSec float64
	hosts       map[string]int
	segCount    int
}

// CleanM3U8 optionally strips #EXT-X-DISCONTINUITY ad blocks and rewrites
// playlist/segment/key URLs to route through proxyBaseURL.
func CleanM3U8(body, baseURL, proxyBaseURL string, headers map[string]string, filterAds bool) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		base = nil
	}
	encodedHeaders := encodeHeaders(headers)

	rawLines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, l := range rawLines {
		lines = append(lines, strings.TrimRight(l, "\r"))
	}

	if filterAds {
		lines = stripDiscontinuityAds(lines, base)
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if strings.HasPrefix(trimmed, "#EXT-X-KEY:") || strings.HasPrefix(trimmed, "#EXT-X-MAP:") {
				line = uriAttrRe.ReplaceAllStringFunc(line, func(match string) string {
					sub := uriAttrRe.FindStringSubmatch(match)
					if len(sub) < 2 {
						return match
					}
					resolved := resolveURL(base, sub[1])
					return `URI="` + buildProxyURL(proxyBaseURL, "/segment.ts", resolved, encodedHeaders) + `"`
				})
			}
			out = append(out, line)
			continue
		}

		resolved := resolveURL(base, trimmed)
		endpoint := "/segment.ts"
		if isM3U8URL(resolved) {
			endpoint = "/playlist.m3u8"
		}
		out = append(out, buildProxyURL(proxyBaseURL, endpoint, resolved, encodedHeaders))
	}
	return strings.Join(out, "\n")
}

func stripDiscontinuityAds(lines []string, base *url.URL) []string {
	var header []string
	var blocks []m3u8Block
	cur := m3u8Block{hosts: make(map[string]int)}
	inMedia := false
	hasDiscontinuity := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "#EXT-X-DISCONTINUITY" {
			hasDiscontinuity = true
			if inMedia {
				blocks = append(blocks, cur)
				cur = m3u8Block{hosts: make(map[string]int)}
			} else {
				inMedia = true
			}
			continue
		}
		if !inMedia && (strings.HasPrefix(trimmed, "#EXTINF:") || (trimmed != "" && !strings.HasPrefix(trimmed, "#"))) {
			inMedia = true
		}
		if !inMedia {
			header = append(header, line)
			continue
		}
		if trimmed == "#EXT-X-ENDLIST" {
			blocks = append(blocks, cur)
			cur = m3u8Block{hosts: make(map[string]int)}
			blocks = append(blocks, m3u8Block{lines: []string{line}, hosts: make(map[string]int)})
			continue
		}
		cur.lines = append(cur.lines, line)
		if m := extinfRe.FindStringSubmatch(trimmed); len(m) == 2 {
			if d, err := strconv.ParseFloat(m[1], 64); err == nil {
				cur.durationSec += d
			}
		} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			cur.segCount++
			if u, err := url.Parse(resolveURL(base, trimmed)); err == nil && u.Host != "" {
				cur.hosts[u.Host]++
			}
		}
	}
	if len(cur.lines) > 0 {
		blocks = append(blocks, cur)
	}
	if !hasDiscontinuity || len(blocks) <= 1 {
		return lines
	}

	// Identify dominant host across all segments.
	hostCounts := make(map[string]int)
	for _, b := range blocks {
		for h, n := range b.hosts {
			hostCounts[h] += n
		}
	}
	dominantHost := ""
	maxCount := 0
	for h, n := range hostCounts {
		if n > maxCount {
			maxCount = n
			dominantHost = h
		}
	}

	lastMediaIdx := -1
	for i, b := range blocks {
		if b.segCount > 0 {
			lastMediaIdx = i
		}
	}
	multiHost := len(hostCounts) > 1

	result := append([]string{}, header...)
	emittedMedia := false
	for i, b := range blocks {
		if len(b.lines) == 1 && strings.TrimSpace(b.lines[0]) == "#EXT-X-ENDLIST" {
			result = append(result, b.lines[0])
			continue
		}
		if isAdBlock(i, lastMediaIdx, b, dominantHost, multiHost) {
			continue
		}
		if emittedMedia {
			result = append(result, "#EXT-X-DISCONTINUITY")
		}
		result = append(result, b.lines...)
		if b.segCount > 0 {
			emittedMedia = true
		}
	}
	return result
}

func isAdBlock(idx, lastMediaIdx int, b m3u8Block, dominantHost string, multiHost bool) bool {
	if b.segCount == 0 {
		return false
	}
	if multiHost && dominantHost != "" && len(b.hosts) > 0 {
		diffHost := true
		for h := range b.hosts {
			if h == dominantHost {
				diffHost = false
				break
			}
		}
		return diffHost && b.durationSec <= 60
	}
	// Single-host playlist: strip short enclosed blocks between discontinuities.
	return idx > 0 && idx < lastMediaIdx && b.durationSec > 0 && b.durationSec <= 45
}

// StripFakeImageHeader removes fake PNG/JPEG/GIF bytes prepended to an MPEG-TS segment.
func StripFakeImageHeader(data []byte) []byte {
	if !hasFakeImageMagic(data) {
		return data
	}
	const tsPacketSize = 188
	for i := 0; i+tsPacketSize < len(data); i++ {
		if data[i] == 0x47 && data[i+tsPacketSize] == 0x47 {
			if i+2*tsPacketSize >= len(data) || data[i+2*tsPacketSize] == 0x47 {
				return data[i:]
			}
		}
	}
	return data
}

func hasFakeImageMagic(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	return bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) ||
		bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) ||
		bytes.HasPrefix(data, []byte("GIF8"))
}

func resolveURL(base *url.URL, ref string) string {
	u, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ref
	}
	if base != nil {
		return base.ResolveReference(u).String()
	}
	return u.String()
}

func isM3U8URL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return strings.Contains(strings.ToLower(raw), ".m3u8")
	}
	return strings.HasSuffix(strings.ToLower(u.Path), ".m3u8")
}

func buildProxyURL(proxyBase, endpoint, targetURL, encodedHeaders string) string {
	q := url.Values{}
	q.Set("url", targetURL)
	if encodedHeaders != "" {
		q.Set("h", encodedHeaders)
	}
	return strings.TrimRight(proxyBase, "/") + endpoint + "?" + q.Encode()
}

func encodeHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	b, err := json.Marshal(headers)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeHeaders(encoded string) map[string]string {
	if encoded == "" {
		return nil
	}
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}
