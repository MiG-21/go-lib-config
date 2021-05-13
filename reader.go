package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	TagDataLayout      = "data-layout"
	TagDataSeparator   = "data-separator"
	TagDataDefault     = "data-default"
	TagDataDescription = "data-description"
	TagDataNotLogging  = "data-not-logging"

	// DefaultSeparator is a default list and map Separator character
	DefaultSeparator = ","
)

type (
	Reader interface {
		Read(cfg interface{}) error
	}

	// StructMeta is a structure metadata entity
	StructMeta struct {
		FieldName        string
		FieldValue       reflect.Value
		Tag              *reflect.StructTag
		Layout           string
		Separator        string
		DefValue         string
		DefValueProvided bool
		Description      string
		NotLogging       bool
	}
)

// isFieldValueZero determines if FieldValue empty or not
func (sm *StructMeta) isFieldValueZero() bool {
	return sm.FieldValue.IsZero()
}

// readStructMetadata reads structure metadata (types, tags, etc.)
func readStructMetadata(cfgRoot interface{}) ([]StructMeta, error) {
	cfgStack := []interface{}{cfgRoot}
	metas := make([]StructMeta, 0)

	for i := 0; i < len(cfgStack); i++ {
		s := reflect.ValueOf(cfgStack[i])

		// unwrap pointer
		if s.Kind() == reflect.Ptr {
			s = s.Elem()
		}

		// process only structures
		if s.Kind() != reflect.Struct {
			return nil, fmt.Errorf("wrong type %v", s.Kind())
		}
		typeInfo := s.Type()

		// read tags
		for idx := 0; idx < s.NumField(); idx++ {
			fType := typeInfo.Field(idx)

			var (
				layout    string
				separator string
			)

			// process nested structure (except of time.Time)
			if fld := s.Field(idx); fld.Kind() == reflect.Struct {
				// add structure to parsing stack
				if fld.Type() != reflect.TypeOf(time.Time{}) {
					cfgStack = append(cfgStack, fld.Addr().Interface())
					continue
				}
				// process time.Time
				if l, ok := fType.Tag.Lookup(TagDataLayout); ok {
					layout = l
				}
			}

			// check is the field value can be changed
			if !s.Field(idx).CanSet() {
				continue
			}

			defValue, defValueProvided := fType.Tag.Lookup(TagDataDefault)
			dataDescription, _ := fType.Tag.Lookup(TagDataDescription)
			_, dataNotLogging := fType.Tag.Lookup(TagDataNotLogging)

			if sep, ok := fType.Tag.Lookup(TagDataSeparator); ok {
				separator = sep
			} else {
				separator = DefaultSeparator
			}

			metas = append(metas, StructMeta{
				FieldName:        s.Type().Field(idx).Name,
				FieldValue:       s.Field(idx),
				Tag:              &fType.Tag,
				Layout:           layout,
				Separator:        separator,
				DefValue:         defValue,
				DefValueProvided: defValueProvided,
				Description:      dataDescription,
				NotLogging:       dataNotLogging,
			})
		}
	}

	return metas, nil
}

// parseValue parses value into the corresponding field.
// In case of maps and slices it uses provided Separator to split raw value string
func parseValue(field reflect.Value, value, sep, layout string) error {
	valueType := field.Type()

	switch valueType.Kind() {
	// parse string value
	case reflect.String:
		field.SetString(value)

	// parse boolean value
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)

	// parse integer (or time) value
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Kind() == reflect.Int64 && valueType.PkgPath() == "time" && valueType.Name() == "Duration" {
			// try to parse time
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(d))

		} else {
			// parse regular integer
			number, err := strconv.ParseInt(value, 0, valueType.Bits())
			if err != nil {
				return err
			}
			field.SetInt(number)
		}

	// parse unsigned integer value
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		number, err := strconv.ParseUint(value, 0, valueType.Bits())
		if err != nil {
			return err
		}
		field.SetUint(number)

	// parse floating point value
	case reflect.Float32, reflect.Float64:
		number, err := strconv.ParseFloat(value, valueType.Bits())
		if err != nil {
			return err
		}
		field.SetFloat(number)

	// parse sliced value
	case reflect.Slice:
		sliceValue, err := parseSlice(valueType, value, sep, layout)
		if err != nil {
			return err
		}

		field.Set(*sliceValue)

	// parse mapped value
	case reflect.Map:
		mapValue, err := parseMap(valueType, value, sep, layout)
		if err != nil {
			return err
		}

		field.Set(*mapValue)

	case reflect.Struct:
		// process time.Time only
		if valueType.PkgPath() == "time" && valueType.Name() == "Time" {
			var l string
			if layout != "" {
				l = layout
			} else {
				l = time.RFC3339
			}
			val, err := time.Parse(l, value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(val))
		}

	default:
		return fmt.Errorf("unsupported type %s.%s", valueType.PkgPath(), valueType.Name())
	}

	return nil
}

// parseSlice parses value into a slice of given type
func parseSlice(valueType reflect.Type, value, sep, layout string) (*reflect.Value, error) {
	sliceValue := reflect.MakeSlice(valueType, 0, 0)
	if valueType.Elem().Kind() == reflect.Uint8 {
		sliceValue = reflect.ValueOf([]byte(value))
	} else if len(strings.TrimSpace(value)) != 0 {
		values := strings.Split(value, sep)
		sliceValue = reflect.MakeSlice(valueType, len(values), len(values))

		for i, val := range values {
			if err := parseValue(sliceValue.Index(i), val, sep, layout); err != nil {
				return nil, err
			}
		}
	}
	return &sliceValue, nil
}

// parseMap parses value into a map of given type
func parseMap(valueType reflect.Type, value, sep, layout string) (*reflect.Value, error) {
	mapValue := reflect.MakeMap(valueType)
	if len(strings.TrimSpace(value)) != 0 {
		pairs := strings.Split(value, sep)
		for _, pair := range pairs {
			kvPair := strings.SplitN(pair, ":", 2)
			if len(kvPair) != 2 {
				return nil, fmt.Errorf("invalid map item: %q", pair)
			}
			k := reflect.New(valueType.Key()).Elem()
			err := parseValue(k, kvPair[0], sep, layout)
			if err != nil {
				return nil, err
			}
			v := reflect.New(valueType.Elem()).Elem()
			err = parseValue(v, kvPair[1], sep, layout)
			if err != nil {
				return nil, err
			}
			mapValue.SetMapIndex(k, v)
		}
	}
	return &mapValue, nil
}

// setDefaults data after populating
func setDefaults(cfg interface{}) error {
	metaInfo, err := readStructMetadata(cfg)
	if err != nil {
		return err
	}
	errCollector := errorCollector()
	var cErr error
	for _, meta := range metaInfo {
		if meta.isFieldValueZero() {
			if meta.DefValue != "" {
				if err = parseValue(meta.FieldValue, meta.DefValue, meta.Separator, meta.Layout); err != nil {
					cErr = errCollector(err)
				} else if !meta.NotLogging {
					LibLogger(fmt.Sprintf("DEFAULT: %s = %v", meta.FieldName, meta.FieldValue))
				}
			}
		}
	}
	return cErr
}

// errorCollector for populate errors without break read loop
func errorCollector() func(err error) error {
	var collectedErr error
	return func(err error) error {
		if collectedErr == nil {
			collectedErr = err
		} else {
			collectedErr = errors.Wrap(collectedErr, err.Error())
		}
		return collectedErr
	}
}
