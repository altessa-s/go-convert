module github.com/altessa-s/go-convert/proto

go 1.27.0

toolchain go1.27.4

require (
	github.com/altessa-s/go-atlas-core v0.3.3
	github.com/altessa-s/go-convert v0.1.0
	github.com/altessa-s/go-convert/codec/optionalcodec v0.1.0
	github.com/stretchr/testify v1.12.1
	google.golang.org/protobuf v1.36.12
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace github.com/altessa-s/go-convert => ..

replace github.com/altessa-s/go-convert/codec/optionalcodec => ../codec/optionalcodec
