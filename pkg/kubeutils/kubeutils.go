package kubeutils

import (
	"bufio"
	"io"

	multierror "github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

// Deserialize is a function that parses a buffer reader
// for Kubernetes YAML/JSON manifests and returns a slice
// of Kubernetes runtime objects
func Deserialize(m io.Reader) ([]runtime.Object, error) {
	var merror error
	var rto []runtime.Object
	br := bufio.NewReader(m)
	yr := yaml.NewYAMLReader(br)

	// Iterate over input until EOF
	// parse YAML/JSON and convert to Runtime Object
	for {
		y, err := yr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			merror = multierror.Append(merror, err)
		}
		obj, err := deserialize(y)
		if err != nil {
			merror = multierror.Append(merror, err)
		}

		if obj != nil {
			rto = append(rto, obj)
		}
	}
	return rto, merror
}

// deserialize is a function that parses YAML
// or JSON and returns a Kubernetes Runtime Object
func deserialize(l []byte) (runtime.Object, error) {
	des := scheme.Codecs.UniversalDeserializer()
	obj, _, err := des.Decode(l, nil, nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
