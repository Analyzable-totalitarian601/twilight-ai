package types

type Source struct {
	SourceType       string
	ID               string
	URL              string
	Title            string
	ProviderMetadata map[string]any
}
