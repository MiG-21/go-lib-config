package config

import (
	"fmt"
	"os"
)

type EnvReader struct {
	abstractReader
	tag string
}

// reads environment variables to the provided configuration structure
func (r EnvReader) Read(cfg interface{}) error {
	metaInfo, err := readStructMetadata(cfg)
	if err != nil {
		return err
	}

	errCollector := errorCollector()
	var cErr error
	for _, meta := range metaInfo {
		tag, _ := meta.Tag.Lookup(r.tag)
		if tag == "" {
			continue
		}

		var rawValue *string

		if value, ok := os.LookupEnv(tag); ok {
			rawValue = &value
		} else {
			cErr = errCollector(fmt.Errorf("ENV: %s is not set", tag))
			continue
		}

		if err = parseValue(meta.FieldValue, *rawValue, meta.Separator, meta.Layout); err != nil {
			cErr = errCollector(err)
		}
	}

	return cErr
}
