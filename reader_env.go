package config

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"
)

type EnvReader struct {
	tag string
}

// reads environment variables to the provided configuration structure
func (r EnvReader) Read(cfg interface{}) error {
	metaInfo, err := readStructMetadata(cfg)
	if err != nil {
		return err
	}

	var result *multierror.Error
	for _, meta := range metaInfo {
		tag, _ := meta.Tag.Lookup(r.tag)
		if tag == "" {
			continue
		}

		LibLogger(fmt.Sprintf("reading %s", tag))

		var rawValue *string

		if value, ok := os.LookupEnv(tag); ok {
			rawValue = &value
		} else {
			if !meta.DefValueProvided || Verbose {
				result = multierror.Append(result, fmt.Errorf("env var %s is not set", tag))
			}
			continue
		}

		if err = parseValue(meta.FieldValue, *rawValue, meta.Separator, meta.Layout); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}
