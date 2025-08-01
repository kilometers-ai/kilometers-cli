module github.com/kilometers-ai/kilometers-cli

go 1.24.5

require (
	github.com/kilometers-ai/kilometers-cli-plugins v0.1.0
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// For local development, use local plugin source
replace github.com/kilometers-ai/kilometers-cli-plugins => ../kilometers-cli-plugins
