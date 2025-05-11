module github.com/pascalcheek/subpub

go 1.21

require (
	google.golang.org/grpc v1.62.0
	google.golang.org/protobuf v1.34.1
)

replace (
	google.golang.org/protobuf => google.golang.org/protobuf v1.34.1
	google.golang.org/grpc => google.golang.org/grpc v1.62.0
)