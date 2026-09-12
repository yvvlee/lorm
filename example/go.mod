module github.com/yvvlee/lorm/example

go 1.27

tool github.com/yvvlee/lorm/cmd/lormgen

require (
	github.com/mattn/go-sqlite3 v1.14.52
	github.com/stretchr/testify v1.12.1
	github.com/yvvlee/lorm v0.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/tools v0.50.0 // indirect
)

replace github.com/yvvlee/lorm => ..
