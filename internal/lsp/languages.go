package lsp

// LanguageID returns the LSP language identifier for filename.
// It delegates to the registry so the mapping stays in one place.
func LanguageID(filename string) string {
	return LanguageIDForFile(filename)
}
