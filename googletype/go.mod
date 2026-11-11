module github.com/altessa-s/go-convert/googletype

go 1.27.0

toolchain go1.27.4

require (
	github.com/altessa-s/go-atlas-core v0.3.3
	github.com/altessa-s/go-convert v0.1.0
	github.com/stretchr/testify v1.12.1
	google.golang.org/genproto v0.0.0-20261026165753-2ffab32bce7e
	google.golang.org/protobuf v1.36.12
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace github.com/altessa-s/go-convert => ..
