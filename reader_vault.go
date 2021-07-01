package config

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

type (
	SecretPathFormatter func(secret string) string

	VaultReader struct {
		storage   *StorageVault
		formatter SecretPathFormatter
		tag       string
	}
)

// reads vault variables to the provided configuration structure
func (r VaultReader) Read(metas []StructMeta) error {
	var err error

	keyMap := r.storage.InitMemorisedKvMap()

	var result *multierror.Error
	for k, meta := range metas {
		var val interface{}
		tag, _ := meta.Tag.Lookup(r.tag)
		if tag == "" {
			continue
		}
		vaultTags := strings.Split(tag, ":")
		if len(vaultTags) != 2 {
			result = multierror.Append(result, fmt.Errorf("%s secret is invalid", tag))
			continue
		}
		key := vaultTags[0]
		if r.formatter != nil {
			key = r.formatter(key)
		}

		LibLogger(fmt.Sprintf("reading %s:%s", key, vaultTags[1]))

		if val, err = keyMap(key, vaultTags[1]); err != nil {
			if !meta.DefValueProvided || Verbose {
				result = multierror.Append(result, err)
			}
			continue
		}

		if err = parseValue(meta.FieldValue, val.(string), meta.Separator, meta.Layout); err != nil {
			result = multierror.Append(result, err)
		} else {
			metas[k].Provider = r.tag
		}
	}

	return result
}

func (r VaultReader) Stop() {
	r.storage.Stop()
}
