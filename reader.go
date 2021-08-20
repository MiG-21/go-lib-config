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
	// Reader should be implemented by custom reader
	Reader interface {
		Read(metas []StructMeta) error
		Stop()
	}

	// Setter gives an ability to implement custom setter for a field or struct
	Setter interface {
		SetValue(string) error
	}

	// Updater gives an ability to implement custom update function for a config structure
	Updater interface {
		Update() error
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
		Provider         string
	}
)

// ReadStructMetadata reads structure metadata (types, tags, etc.)
func ReadStructMetadata(cfgRoot interface{}) ([]StructMeta, error) {
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
				Provider:         "-",
			})
		}
	}

	return metas, nil
}

// parseValue parses value into the corresponding field.
// In case of maps and slices it uses provided Separator to split raw value string
func parseValue(field reflect.Value, value, sep, layout string) error {
	if field.CanInterface() {
		if cs, ok := field.Interface().(Setter); ok {
			return cs.SetValue(value)
		} else if csp, ok := field.Addr().Interface().(Setter); ok {
			return csp.SetValue(value)
		}
	}

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

	// Experimental
	case reflect.Ptr:
		field = indirect(field)
		// ... not sure that this case is possible,
		// but we should prevent infinite recursion
		valueType = field.Type()
		if valueType.Kind() == reflect.Ptr {
			return fmt.Errorf("unsupported type %s.%s", valueType.PkgPath(), valueType.Name())
		}
		return parseValue(field, value, sep, layout)

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
func setDefaults(metas []StructMeta) error {
	errCollector := errorCollector()
	var cErr, err error
	for k, meta := range metas {
		if meta.DefValueProvided {
			if err = parseValue(meta.FieldValue, meta.DefValue, meta.Separator, meta.Layout); err != nil {
				cErr = errCollector(err)
			} else {
				metas[k].Provider = "default"
			}
		}
	}
	return cErr
}

// dumpMetas by logger
func dumpMetas(metas []StructMeta) {
	for _, meta := range metas {
		if meta.NotLogging {
			LibLogger(fmt.Sprintf("%s = ********** [%s]", meta.FieldName, meta.Provider))
		} else {
			LibLogger(fmt.Sprintf("%s = %v [%s]", meta.FieldName, meta.FieldValue, meta.Provider))
		}
	}
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

// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
func indirect(v reflect.Value) reflect.Value {
	// The logic below effectively does this when it first addresses the value
	// (to satisfy possible pointer methods) and continues to dereference
	// subsequent pointers as necessary.
	//
	// After the first round-trip, we set v back to the original value to
	// preserve the original RW flags contained in reflect.Value.
	v0 := v
	haveAddr := false

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		haveAddr = true
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && e.Elem().Kind() == reflect.Ptr {
				haveAddr = false
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		// Prevent infinite loop if v is an interface pointing to its own address:
		//     var v interface{}
		//     v = &v
		if v.Elem().Kind() == reflect.Interface && v.Elem().Elem() == v {
			v = v.Elem()
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 && v.CanInterface() {
			return reflect.Value{}
		}

		if haveAddr {
			v = v0 // restore original value after round-trip Value.Addr().Elem()
			haveAddr = false
		} else {
			v = v.Elem()
		}
	}

	return v
}
