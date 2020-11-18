package kubeutils

import (
	"bufio"
	"io"

	multierror "github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
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
			if obj.GetObjectKind().GroupVersionKind().Kind == "List" {
				// Runtime Object is of `Kind: List`
				// Lists may contain RawExtensions which are stored as
				// JSON, and require additional pass through Deserializer
				// Ref: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#rawextension
				// https://github.com/kubernetes/apimachinery/blob/master/pkg/runtime/types.go#L94
				// Convert the RuntimeObject to a List Object before deserializing
				listObj := obj.(*corev1.List)
				for _, i := range listObj.Items {
					// Deserialize the ListObject JSON
					lo, err := deserialize(i.Raw)
					if err != nil {
						merror = multierror.Append(merror, err)
					}
					if lo != nil {
						rto = append(rto, lo)
					}
				}
			} else {
				rto = append(rto, obj)
			}
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
