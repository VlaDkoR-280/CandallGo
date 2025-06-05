package localization

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Local struct {
	Data map[string]map[string]string
}

func (l *Local) Update() error {
	data, err := os.ReadFile("loc.yaml")
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &l.Data)

	return err
}

func (l *Local) Get(lang, key string) string {
	return l.Data[lang][key]
}
