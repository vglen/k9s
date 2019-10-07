package resource

import (
	"fmt"
	"time"

	"github.com/derailed/k9s/internal/k8s"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CustomResourceDefinition tracks a kubernetes resource.
type CustomResourceDefinition struct {
	*Base
	instance *unstructured.Unstructured
}

// NewCustomResourceDefinitionList returns a new resource list.
func NewCustomResourceDefinitionList(c Connection, ns string) List {
	return NewList(
		NotNamespaced,
		"crd",
		NewCustomResourceDefinition(c),
		CRUDAccess|DescribeAccess,
	)
}

// NewCustomResourceDefinition instantiates a new CustomResourceDefinition.
func NewCustomResourceDefinition(c Connection) *CustomResourceDefinition {
	crd := &CustomResourceDefinition{&Base{Connection: c, Resource: k8s.NewCustomResourceDefinition(c)}, nil}
	crd.Factory = crd

	return crd
}

// New builds a new CustomResourceDefinition instance from a k8s resource.
func (r *CustomResourceDefinition) New(i interface{}) Columnar {
	c := NewCustomResourceDefinition(r.Connection)
	switch instance := i.(type) {
	case *unstructured.Unstructured:
		c.instance = instance
	case unstructured.Unstructured:
		c.instance = &instance
	default:
		log.Fatal().Msgf("unknown CustomResourceDefinition type %#v", i)
	}
	meta := c.instance.Object["metadata"].(map[string]interface{})
	c.path = meta["name"].(string)

	return c
}

// Marshal a resource.
func (r *CustomResourceDefinition) Marshal(path string) (string, error) {
	ns, n := Namespaced(path)
	i, err := r.Resource.Get(ns, n)
	if err != nil {
		return "", err
	}

	raw, err := yaml.Marshal(i)
	if err != nil {
		return "", err
	}

	// BOZO!! Need to figure out apiGroup+Version
	// return r.marshalObject(i.(*unstructured.Unstructured))
	return string(raw), nil
}

// Header return the resource header.
func (*CustomResourceDefinition) Header(ns string) Row {
	return Row{"NAME", "AGE"}
}

// Fields retrieves displayable fields.
func (r *CustomResourceDefinition) Fields(ns string) Row {
	ff := make(Row, 0, len(r.Header(ns)))

	i := r.instance
	meta := i.Object["metadata"].(map[string]interface{})
	t, err := time.Parse(time.RFC3339, meta["creationTimestamp"].(string))
	if err != nil {
		log.Error().Msgf("Fields timestamp %v", err)
	}

	return append(ff, meta["name"].(string), toAge(metav1.Time{t}))
}

// ExtFields returns extended fields.
func (r *CustomResourceDefinition) ExtFields1(m *TypeMeta) {
	i := r.instance
	spec, ok := i.Object["spec"].(map[string]interface{})
	if !ok {
		return
	}

	if meta, ok := i.Object["metadata"].(map[string]interface{}); ok {
		m.Name = meta["name"].(string)
	}
	m.Group, m.Version = spec["group"].(string), spec["version"].(string)
	m.Namespaced = isNamespaced(spec["scope"].(string))
	names, ok := spec["names"].(map[string]interface{})
	if !ok {
		return
	}
	m.Kind = names["kind"].(string)
	m.Singular, m.Plural = names["singular"].(string), names["plural"].(string)
	if names["shortNames"] != nil {
		for _, s := range names["shortNames"].([]interface{}) {
			m.ShortNames = append(m.ShortNames, s.(string))
		}
	} else {
		m.ShortNames = nil
	}
}

// ExtFields returns extended fields.
func (r *CustomResourceDefinition) ExtFields(m *TypeMeta) error {
	i := r.instance
	spec, ok := i.Object["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to locate `spec field on CRD")
	}

	meta, ok := i.Object["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to locate CRD `metadata field")
	}

	var err error
	if m.Name, err = mapStr(meta, "name"); err != nil {
		return err
	}
	if m.Group, err = mapStr(spec, "group"); err != nil {
		return err
	}
	if m.Version, err = mapStr(spec, "version"); err != nil {
		return err
	}
	n, err := mapStr(spec, "scope")
	if err != nil {
		return err
	}
	m.Namespaced = isNamespaced(n)

	names, ok := spec["names"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to locate `names on CRD %s:%s", m.Group, m.Version)
	}
	if m.Kind, err = mapStr(names, "kind"); err != nil {
		return err
	}
	if m.Singular, err = mapStr(names, "singular"); err != nil {
		return err
	}
	if m.Plural, err = mapStr(names, "plural"); err != nil {
		return err
	}

	aa, ok := names["shortNames"].([]interface{})
	if !ok {
		m.ShortNames = nil
		return nil
	}
	for _, a := range aa {
		if v, ok := a.(string); ok {
			m.ShortNames = append(m.ShortNames, v)
		}
	}

	return nil
}

func isNamespaced(scope string) bool {
	return scope == "Namespaced"
}

func mapStr(m map[string]interface{}, s string) (string, error) {
	f, ok := m[s]
	if !ok {
		return "", fmt.Errorf("Unable to locate CRD field %s", s)
	}

	v, ok := f.(string)
	if !ok {
		return "", fmt.Errorf("No string type for CRD field %s", s)
	}

	return v, nil
}
