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
func (r EnvReader) Read(metas []StructMeta) error {
	var err error

	var result *multierror.Error
	for k, meta := range metas {
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
				result = multierror.Append(result, fmt.Errorf("%s is not set", tag))
			}
			continue
		}

		if err = parseValue(meta.FieldValue, *rawValue, meta.Separator, meta.Layout); err != nil {
			result = multierror.Append(result, err)
		} else {
			metas[k].Provider = r.tag
		}
	}

	return result
}
