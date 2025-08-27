package domain

// PluginDescriptor describes a plugin binary to run locally during development
// or testing. It intentionally contains only the minimal, stable attributes
// that identify and locate a plugin implementation.
type PluginDescriptor struct {
	// Name is the logical plugin name (e.g., "console-logger")
	Name string

	// Version is optional and may be empty for local builds
	Version string

	// Path is the absolute path to the plugin binary
	Path string

	// Source indicates where this descriptor originated (e.g., "local")
	Source string
}
