package session

import posixpath "path"

func RevealCommand(goos, path string) (string, []string) {
	switch goos {
	case "windows":
		return "explorer", []string{"/select," + path}
	case "darwin":
		return "open", []string{"-R", path}
	default:
		return "xdg-open", []string{posixpath.Dir(path)}
	}
}

func OpenCommand(goos, path string) (string, []string) {
	switch goos {
	case "windows":
		return "explorer", []string{path}
	case "darwin":
		return "open", []string{path}
	default:
		return "xdg-open", []string{path}
	}
}
