package inventory

import (
	"k8s.io/apimachinery/pkg/types"
)

// SearchResult contains a Inventory result item.
type SearchResult struct {
	BuildName  types.NamespacedName // build name maching criteria
	SecretName types.NamespacedName // respective secret coordinates (for webhook)
}

// HasSecret assert if the SecretName is defined.
func (s *SearchResult) HasSecret() bool {
	return s.SecretName.Namespace != "" && s.SecretName.Name != ""
}

// ExtractBuildNames picks the build names from informed SearchResult slice.
func ExtractBuildNames(results ...SearchResult) []string {
	var names []string
	for _, entry := range results {
		names = append(names, entry.BuildName.Name)
	}
	return names
}
