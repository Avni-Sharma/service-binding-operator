package binding

import (
	"fmt"
	"strings"

	"github.com/redhat-developer/service-binding-operator/pkg/controller/servicebindingrequest/nested"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// bindingType encodes the medium the binding should deliver the configuration value.
type bindingType string

const (
	// BindingTypeVolumeMount indicates the binding should happen through a volume mount.
	BindingTypeVolumeMount bindingType = "volumemount"
	// BindingTypeEnvVar indicates the binding should happen through environment variables.
	BindingTypeEnvVar bindingType = "env"
)

// result contains data that has been collected by an annotation handler.
type result struct {
	// Data contains the annotation data collected by an annotation handler inside a deep structure
	// with its root being the value specified in the Path field.
	Data map[string]interface{}
	// Type indicates where the Object field should be injected in the application; can be either
	// "env" or "volumemount".
	Type bindingType
	// Path is the nested location the collected data can be found in the Data field.
	Path string
	// RawData contains the annotation data collected by an annotation handler
	// inside a deep structure with its root being composed by the path where
	// the external resource name was extracted and the path within the external
	// resource.
	RawData map[string]interface{}
}

type errHandlerNotFound string

func (e errHandlerNotFound) Error() string {
	return fmt.Sprintf("could not find handler for annotation value %q", string(e))
}

func IsErrHandlerNotFound(err error) bool {
	_, ok := err.(errHandlerNotFound)
	return ok
}

type SpecHandler struct {
	kubeClient      dynamic.Interface
	obj             unstructured.Unstructured
	annotationKey   string
	annotationValue string
	restMapper      meta.RESTMapper
}

func (s *SpecHandler) Handle() (result, error) {
	mapper := &AnnotationToDefinitionMapper{
		KubeClient: s.kubeClient,
	}
	opts := NewAnnotationMapperOptions(s.annotationKey, s.annotationValue)
	d, err := mapper.Map(opts)
	if err != nil {
		return result{}, err
	}

	val, err := d.Apply(&s.obj)
	if err != nil {
		return result{}, err
	}

	v := val.GetValue()

	path := strings.Join(d.GetPath(), ".")

	out := make(map[string]interface{})

	switch t := v.(type) {
	case map[string]string:
		for k, v := range t {
			out[k] = v
		}
	case map[string]interface{}:
		for k, v := range t {
			out[k] = v
		}
	}

	return result{
		Data:    out,
		RawData: nested.ComposeValue(out, nested.NewPath(path)),
	}, nil
}

func NewSpecHandler(
	kubeClient dynamic.Interface,
	annotationKey string,
	annotationValue string,
	obj unstructured.Unstructured,
	restMapper meta.RESTMapper,
) (*SpecHandler, error) {
	return &SpecHandler{
		kubeClient:      kubeClient,
		obj:             obj,
		annotationKey:   annotationKey,
		annotationValue: annotationValue,
		restMapper:      restMapper,
	}, nil
}

func IsSpec(annotationKey string) bool {
	return strings.HasPrefix(annotationKey, AnnotationPrefix)
}
