// Api versions allow the api contract for a resource to be changed while keeping
// backward compatibility by support multiple concurrent versions
// of the same resource

//go:generate go run ../../../../vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy -i . -h ../../../../boilerplate.go.txt
//go:generate go run ../../../../vendor/k8s.io/code-generator/cmd/defaulter-gen/main.go -O zz_generated.defaults -i . -h ../../../../boilerplate.go.txt
//go:generate go run ../../../../vendor/k8s.io/code-generator/cmd/conversion-gen/main.go -O zz_generated.conversion -i . -h ../../../../boilerplate.go.txt

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package
// +k8s:defaulter-gen=TypeMeta
// +groupName=storage.loft.sh
package v1 // import "github.com/loft-sh/api/v4/apis/storage/v1"
