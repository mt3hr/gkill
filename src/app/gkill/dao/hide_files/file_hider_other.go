//go:build !windows

package hide_files

func hideFolder(path string) error   { return nil }
func unhideFolder(path string) error { return nil }
