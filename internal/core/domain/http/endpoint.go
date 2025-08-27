package httpdomain

// BackendEndpoint describes the single backend target used by the CLI.
type BackendEndpoint struct {
	BaseURL   string
	UserAgent string
}
