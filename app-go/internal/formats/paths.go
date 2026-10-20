package formats

import (
	"regexp"
	"runtime"
	"strings"

	"orcaton/internal/third_party/norm"
)

var driveRe = regexp.MustCompile(`^/?([A-Za-z]):[/\\]`)

func NFC(s string) string { return norm.NFC.String(s) }

func Casefold(s string) string {
	s = strings.ToLower(s)
	if strings.ContainsAny(s, "ßςſ") {
		s = strings.NewReplacer("ß", "ss", "ς", "σ", "ſ", "s").Replace(s)
	}
	return s
}

func URLToPath(u string) string {
	if !strings.HasPrefix(strings.ToLower(u), "file:") {
		return u
	}
	netloc, p := splitURL(u)
	p = unquote(p)
	if driveRe.MatchString(p) {
		p = strings.TrimLeft(p, "/")
	} else if netloc != "" && strings.ToLower(netloc) != "localhost" {
		p = "//" + netloc + p
	}
	return p
}

func splitURL(u string) (string, string) {
	rest := u[strings.Index(u, ":")+1:]
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	if strings.HasPrefix(rest, "//") {
		rest = rest[2:]
		i := strings.Index(rest, "/")
		if i < 0 {
			return rest, ""
		}
		return rest[:i], rest[i:]
	}
	return "", rest
}

func unquote(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s)+0 && i+2 <= len(s)-1 && isHex(s[i+1]) && isHex(s[i+2]) {
			b = append(b, unhex(s[i+1])<<4|unhex(s[i+2]))
			i += 2
			continue
		}
		b = append(b, s[i])
	}
	return strings.ToValidUTF8(string(b), "\ufffd")
}

func isHex(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	default:
		return c - 'A' + 10
	}
}

type PathRule struct{ Src, Dst string }

func ParsePathMap(spec string) []PathRule {
	var rules []PathRule
	for _, part := range strings.Split(spec, ";") {
		src, dst, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		if s := strings.TrimSpace(src); s != "" {
			rules = append(rules, PathRule{strings.ReplaceAll(s, `\`, "/"), strings.TrimSpace(dst)})
		}
	}
	return rules
}

func RemapPath(p string, rules []PathRule) string {
	q := NFC(strings.ReplaceAll(p, `\`, "/"))
	for _, r := range rules {
		s := NFC(r.Src)
		if strings.HasPrefix(strings.ToLower(q), strings.ToLower(s)) {
			return r.Dst + string([]rune(q)[len([]rune(s)):])
		}
	}
	return p
}

func PathToURL(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	if strings.HasPrefix(p, "//") {
		host, rest, _ := strings.Cut(p[2:], "/")
		return "file://" + host + "/" + quote(rest, "/:")
	}
	if driveRe.MatchString(p) {
		p = "/" + strings.TrimLeft(p, "/")
	}
	return "file://localhost" + quote(p, "/:")
}

func quote(s, safe string) string {
	const hexd = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') ||
			c == '_' || c == '.' || c == '-' || c == '~' || strings.IndexByte(safe, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hexd[c>>4])
		b.WriteByte(hexd[c&15])
	}
	return b.String()
}

func SlashPath(p string) string { return strings.ReplaceAll(p, `\`, "/") }

func BaseName(p string) string {
	_, parts := posixParts(SlashPath(p))
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func Stem(p string) string {
	n := BaseName(p)
	if i := strings.LastIndex(n, "."); i > 0 && i < len(n)-1 {
		return n[:i]
	}
	return n
}

func Suffix(p string) string {
	n := BaseName(p)
	if i := strings.LastIndex(n, "."); i > 0 && i < len(n)-1 {
		return n[i:]
	}
	return ""
}

func ParentName(p string) string {
	_, parts := posixParts(SlashPath(p))
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-2]
}

func posixParts(p string) (string, []string) {
	root := ""
	if strings.HasPrefix(p, "//") && !strings.HasPrefix(p, "///") {
		root = "//"
	} else if strings.HasPrefix(p, "/") {
		root = "/"
	}
	var parts []string
	for _, x := range strings.Split(p, "/") {
		if x != "" && x != "." {
			parts = append(parts, x)
		}
	}
	return root, parts
}

func ParentPath(p string) string {
	root, parts := posixParts(SlashPath(p))
	if len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}
	s := root + strings.Join(parts, "/")
	if s == "" {
		return "."
	}
	return s
}

func winSplit(p string) (string, string, []string) {
	s := strings.ReplaceAll(p, "/", `\`)
	drive, root, rest := "", "", s
	switch {
	case strings.HasPrefix(s, `\\`) && !strings.HasPrefix(s, `\\\`):
		segs := strings.SplitN(s[2:], `\`, 3)
		if len(segs) >= 2 && segs[0] != "" && segs[1] != "" {
			drive = `\\` + segs[0] + `\` + segs[1]
			rest = ""
			if len(segs) == 3 {
				rest = `\` + segs[2]
			}
			root = `\`
		}
	case len(s) >= 2 && s[1] == ':' && ((s[0] >= 'A' && s[0] <= 'Z') || (s[0] >= 'a' && s[0] <= 'z')):
		drive, rest = s[:2], s[2:]
		if strings.HasPrefix(rest, `\`) {
			root = `\`
		}
	case strings.HasPrefix(s, `\`):
		root = `\`
	}
	var parts []string
	for _, x := range strings.Split(rest, `\`) {
		if x != "" && x != "." {
			parts = append(parts, x)
		}
	}
	return drive, root, parts
}

func ParentPathOS(p string) string {
	if runtime.GOOS != "windows" {
		return ParentPath(p)
	}
	return WinParentPath(p)
}

func WinParentPath(p string) string {
	drive, root, parts := winSplit(p)
	if len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}
	s := drive + root + strings.Join(parts, `\`)
	if s == "" {
		return "."
	}
	return s
}

func ParentNameOS(p string) string {
	if runtime.GOOS != "windows" {
		return ParentName(p)
	}
	return WinParentName(p)
}

func WinParentName(p string) string {
	_, _, parts := winSplit(p)
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-2]
}
