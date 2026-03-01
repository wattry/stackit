package pr

// ResolveUnifiedScope returns a scope only when all non-empty scopes match.
// This avoids mislabeling mixed-scope merge metadata.
func ResolveUnifiedScope(scopes []string) string {
	var selected string
	for _, scope := range scopes {
		if scope == "" {
			continue
		}
		if selected == "" {
			selected = scope
			continue
		}
		if scope != selected {
			return ""
		}
	}
	return selected
}
