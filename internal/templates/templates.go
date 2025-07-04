package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"

	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed *.yaml
var templates embed.FS

func ResourceFromTemplate[T any, R any](t *T, name string) (*R, error) {
	resYaml, err := templates.ReadFile(fmt.Sprintf("%s.yaml", name))
	if err != nil {
		return nil, fmt.Errorf("could not read %s template: %v", name, err)
	}
	tmpl, err := template.New(name).Parse(string(resYaml))
	if err != nil {
		return nil, fmt.Errorf("could not parse %s template: %v", name, err)
	}
	var buf bytes.Buffer

	err = tmpl.Execute(&buf, struct {
		Values *T
	}{
		Values: t,
	})
	if err != nil {
		return nil, fmt.Errorf("could not execute %s template: %v", name, err)
	}

	res := new(R)
	decoder := yaml.NewYAMLOrJSONDecoder(&buf, 10)
	if err := decoder.Decode(res); err != nil {
		return nil, fmt.Errorf("could not decode resource %T: %v", res, err)
	}

	return res, nil
}
