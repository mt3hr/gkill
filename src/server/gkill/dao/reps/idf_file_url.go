package reps

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

func buildIDFFileURL(targetRepName, targetFile string) string {
	rep := cleanRelativeURLPath(targetRepName)
	rel := cleanRelativeURLPath(filepath.ToSlash(targetFile))

	if rep == "" {
		if rel == "" {
			return "/files/"
		}
		return "/files/" + escapePathSegments(rel)
	}
	if rel == "" {
		return "/files/" + escapePathSegments(rep) + "/"
	}
	return "/files/" + escapePathSegments(rep) + "/" + escapePathSegments(rel)
}

func cleanRelativeURLPath(p string) string {
	cleaned := path.Clean("/" + p)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func escapePathSegments(rel string) string {
	if rel == "" {
		return ""
	}
	parts := strings.Split(rel, "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/")
}
