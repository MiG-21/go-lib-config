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
func (r VaultReader) Read(cfg interface{}) error {
	metaInfo, err := readStructMetadata(cfg)
	if err != nil {
		return err
	}

	keyMap := r.storage.InitMemorisedKvMap()

	var result *multierror.Error
	for _, meta := range metaInfo {
		var val interface{}
		tag, _ := meta.Tag.Lookup(r.tag)
		if tag == "" {
			continue
		}
		vaultTags := strings.Split(tag, ":")
		if len(vaultTags) != 2 {
			result = multierror.Append(result, fmt.Errorf("%s vault secret is invalid", tag))
			continue
		}
		key := vaultTags[0]
		if r.formatter != nil {
			key = r.formatter(key)
		}

		LibLogger(fmt.Sprintf("VAULT: reading %s:%s", key, vaultTags[1]))

		if val, err = keyMap(key, vaultTags[1]); err != nil {
			result = multierror.Append(result, err)
			continue
		}

		if err = parseValue(meta.FieldValue, val.(string), meta.Separator, meta.Layout); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}
