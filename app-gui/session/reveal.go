package session

import "path/filepath"

func RevealCommand(goos, path string) (string, []string) {
	switch goos {
	case "windows":
		return "explorer", []string{"/select," + path}
	case "darwin":
		return "open", []string{"-R", path}
	default:
		return "xdg-open", []string{filepath.Dir(path)}
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
