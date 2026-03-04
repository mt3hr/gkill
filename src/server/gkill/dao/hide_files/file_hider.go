package hide_files

// HideFolder: Windowsではフォルダを「隠し」属性にする。
// Windows以外では何もしない（no-op）。
func HideFolder(path string) error {
	return hideFolder(path)
}

// UnhideFolder: Windowsでは「隠し」属性を外す。
// Windows以外では何もしない（no-op）。
func UnhideFolder(path string) error {
	return unhideFolder(path)
}
