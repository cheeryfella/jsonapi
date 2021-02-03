package jsonapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	unsupportedStructTagMsg = "Unsupported jsonapi tag annotation, %s"
)

var (
	// ErrInvalidTime is returned when a struct has a time.Time type field, but
	// the JSON value was not a unix timestamp integer.
	ErrInvalidTime = errors.New("only numbers can be parsed as dates, unix timestamps")
	// ErrInvalidISO8601 is returned when a struct has a time.Time type field and includes
	// "iso8601" in the tag spec, but the JSON value was not an ISO8601 timestamp string.
	ErrInvalidISO8601 = errors.New("only strings can be parsed as dates, ISO8601 timestamps")
	// ErrUnknownFieldNumberType is returned when the JSON value was a float
	// (numeric) but the Struct field was a non numeric type (i.e. not int, uint,
	// float, etc)
	ErrUnknownFieldNumberType = errors.New("the struct field was not of a known number type")
	// ErrInvalidType is returned when the given type is incompatible with the expected type.
	ErrInvalidType = errors.New("invalid type provided") // I wish we used punctuation.

)

// ErrUnsupportedPtrType is returned when the Struct field was a pointer but
// the JSON value was of a different type
type ErrUnsupportedPtrType struct {
	rf          reflect.Value
	t           reflect.Type
	structField reflect.StructField
}

func (eupt ErrUnsupportedPtrType) Error() string {
	typeName := eupt.t.Elem().Name()
	kind := eupt.t.Elem().Kind()
	if kind.String() != "" && kind.String() != typeName {
		typeName = fmt.Sprintf("%s (%s)", typeName, kind.String())
	}
	return fmt.Sprintf(
		"jsonapi: Can't unmarshal %+v (%s) to struct field `%s`, which is a pointer to `%s`",
		eupt.rf, eupt.rf.Type().Kind(), eupt.structField.Name, typeName,
	)
}

func newErrUnsupportedPtrType(rf reflect.Value, t reflect.Type, structField reflect.StructField) error {
	return ErrUnsupportedPtrType{rf, t, structField}
}

// UnmarshalPayload converts an io into a struct instance using jsonapi tags on
// struct fields. This method supports single request payloads only, at the
// moment. Bulk creates and updates are not supported yet.
//
// Will Unmarshal embedded and sideloaded payloads.  The latter is only possible if the
// object graph is complete.  That is, in the "relationships" data there are type and id,
// keys that correspond to records in the "included" array.
//
// For example you could pass it, in, req.Body and, model, a BlogPost
// struct instance to populate in an http handler,
//
//   func CreateBlog(w http.ResponseWriter, r *http.Request) {
//   	blog := new(Blog)
//
//   	if err := jsonapi.UnmarshalPayload(r.Body, blog); err != nil {
//   		http.Error(w, err.Error(), 500)
//   		return
//   	}
//
//   	// ...do stuff with your blog...
//
//   	w.Header().Set("Content-Type", jsonapi.MediaType)
//   	w.WriteHeader(201)
//
//   	if err := jsonapi.MarshalPayload(w, blog); err != nil {
//   		http.Error(w, err.Error(), 500)
//   	}
//   }
//
//
// Visit https://github.com/google/jsonapi#create for more info.
//
// model interface{} should be a pointer to a struct.
func UnmarshalPayload(in io.Reader, model interface{}) error {
	payload := new(OnePayload)

	if err := json.NewDecoder(in).Decode(payload); err != nil {
		return err
	}

	if payload.Included != nil {
		includedMap := make(map[string]*ResourceObj)
		for _, included := range payload.Included {
			key := fmt.Sprintf("%s,%s", included.Type, included.ID)
			includedMap[key] = included
		}

		return unmarshalNode(payload.Data, reflect.ValueOf(model), &includedMap)
	}
	return unmarshalNode(payload.Data, reflect.ValueOf(model), nil)
}

// UnmarshalManyPayload converts an io into a set of struct instances using
// jsonapi tags on the type's struct fields.
func UnmarshalManyPayload(in io.Reader, t reflect.Type) ([]interface{}, error) {
	payload := new(ManyPayload)

	if err := json.NewDecoder(in).Decode(payload); err != nil {
		return nil, err
	}

	models := []interface{}{}                // will be populated from the "data"
	includedMap := map[string]*ResourceObj{} // will be populate from the "included"

	if payload.Included != nil {
		for _, included := range payload.Included {
			key := fmt.Sprintf("%s,%s", included.Type, included.ID)
			includedMap[key] = included
		}
	}

	for _, data := range payload.Data {
		model := reflect.New(t.Elem())
		err := unmarshalNode(data, model, &includedMap)
		if err != nil {
			return nil, err
		}
		models = append(models, model.Interface())
	}

	return models, nil
}

func unmarshalNode(data *ResourceObj, model reflect.Value, included *map[string]*ResourceObj) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("data is not a jsonapi representation of '%v'", model.Type())
		}
	}()

	modelValue := model.Elem()
	modelType := modelValue.Type()

	var er error

	for i := 0; i < modelValue.NumField(); i++ {
		fieldType := modelType.Field(i)
		tag := fieldType.Tag.Get("jsonapi")
		if tag == "" {
			continue
		}

		fieldValue := modelValue.Field(i)

		args := strings.Split(tag, ",")
		if len(args) < 2 {
			er = ErrBadJSONAPIStructTag
			break
		}

		annotation := args[0]

		switch {
		case annotation == annotationPrimary:
			if data.ID == "" {
				continue
			}

			// Check the JSON API Type
			if data.Type != args[1] {
				er = fmt.Errorf(
					"Trying to Unmarshal an object of type %#v, but %#v does not match",
					data.Type,
					args[1],
				)
				break
			}

			// ID will have to be transmitted as astring per the JSON API spec
			v := reflect.ValueOf(data.ID)

			// Deal with PTRS
			var kind reflect.Kind
			if fieldValue.Kind() == reflect.Ptr {
				kind = fieldType.Type.Elem().Kind()
			} else {
				kind = fieldType.Type.Kind()
			}

			// Handle String case
			if kind == reflect.String {
				assign(fieldValue, v)
				continue
			}

			// Value was not a string... only other supported type was a numeric,
			// which would have been sent as a float value.
			floatValue, err := strconv.ParseFloat(data.ID, 64)
			if err != nil {
				// Could not convert the value in the "id" attr to a float
				er = ErrBadJSONAPIID
				break
			}

			// Convert the numeric float to one of the supported ID numeric types
			// (int[8,16,32,64] or uint[8,16,32,64])
			idValue, err := handleNumeric(floatValue, fieldType.Type, fieldValue)
			if err != nil {
				// We had a JSON float (numeric), but our field was not one of the
				// allowed numeric types
				er = ErrBadJSONAPIID
				break
			}

			assign(fieldValue, idValue)
		case annotation == annotationAttribute:
			attributes := data.Attributes

			if attributes == nil || len(data.Attributes) == 0 {
				continue
			}

			attribute := attributes[args[1]]

			// continue if the attribute was not included in the request
			if attribute == nil {
				continue
			}

			structField := fieldType
			value, err := unmarshalAttribute(attribute, args, structField, fieldValue)
			if err != nil {
				er = err
				break
			}

			assign(fieldValue, value)
		case annotation == annotationRelation:
			if data.Relationships == nil || data.Relationships[args[1]] == nil {
				continue
			}

			isSlice := fieldValue.Type().Kind() == reflect.Slice

			if isSlice {
				// to-many relationship
				relationship := new(RelationshipManyNode)

				buf := bytes.NewBuffer(nil)

				json.NewEncoder(buf).Encode(data.Relationships[args[1]])
				json.NewDecoder(buf).Decode(relationship)

				data := relationship.Data
				models := reflect.New(fieldValue.Type()).Elem()

				for _, n := range data {
					m := reflect.New(fieldValue.Type().Elem().Elem())

					if err := unmarshalNode(
						fullNode(n, included),
						m,
						included,
					); err != nil {
						er = err
						break
					}

					models = reflect.Append(models, m)
				}

				fieldValue.Set(models)
			} else {
				// to-one relationships
				relationship := new(RelationshipOneNode)

				buf := bytes.NewBuffer(nil)

				json.NewEncoder(buf).Encode(
					data.Relationships[args[1]],
				)
				json.NewDecoder(buf).Decode(relationship)

				/*
					http://jsonapi.org/format/#document-resource-object-relationships
					http://jsonapi.org/format/#document-resource-object-linkage
					relationship can have a data node set to null (e.g. to disassociate the relationship)
					so unmarshal and set fieldValue only if data obj is not null
				*/
				if relationship.Data == nil {
					continue
				}

				m := reflect.New(fieldValue.Type().Elem())
				if err := unmarshalNode(
					fullNode(relationship.Data, included),
					m,
					included,
				); err != nil {
					er = err
					break
				}

				fieldValue.Set(m)

			}
		default:
			er = fmt.Errorf(unsupportedStructTagMsg, annotation)
		}
	}

	return er
}

func fullNode(n *ResourceObj, included *map[string]*ResourceObj) *ResourceObj {
	includedKey := fmt.Sprintf("%s,%s", n.Type, n.ID)

	if included != nil && (*included)[includedKey] != nil {
		return (*included)[includedKey]
	}

	return n
}

// assign will take the value specified and assign it to the field; if
// field is expecting a ptr assign will assign a ptr.
func assign(field, value reflect.Value) {
	value = reflect.Indirect(value)

	if field.Kind() == reflect.Ptr {
		// initialize pointer so it's value
		// can be set by assignValue
		field.Set(reflect.New(field.Type().Elem()))
		field = field.Elem()
	}

	assignValue(field, value)
}

// assign assigns the specified value to the field,
// expecting both values not to be pointer types.
func assignValue(field, value reflect.Value) {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		field.SetInt(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		field.SetUint(value.Uint())
	case reflect.Float32, reflect.Float64:
		field.SetFloat(value.Float())
	case reflect.String:
		field.SetString(value.String())
	case reflect.Bool:
		field.SetBool(value.Bool())
	default:
		field.Set(value)
	}
}

// unmarshalAttribute will unmarshall each attribute field.
func unmarshalAttribute(
	attribute interface{},
	args []string,
	structField reflect.StructField,
	fieldValue reflect.Value) (value reflect.Value, err error) {

	value = reflect.ValueOf(attribute)
	fieldType := structField.Type

	value, err = handleField(attribute, args, fieldType, fieldValue)
	switch{
	case err == ErrInvalidType:
		return reflect.Value{}, ErrInvalidType
	case err == ErrInvalidISO8601:
		return reflect.Value{}, ErrInvalidISO8601
	case err != nil:
		return reflect.Value{},
		newErrUnsupportedPtrType(reflect.ValueOf(attribute), fieldType, structField)
	}
	return
}

// handleField parses each individual field given its type and value. The method allows for recursion when unmarshalling
// so we can traverse to primitive types.
func handleField(
	attribute interface{},
	args []string,
	fieldType reflect.Type,
	fieldValue reflect.Value) (value reflect.Value, err error) {

	value = reflect.ValueOf(attribute)
	switch fieldType.Kind() {
	case reflect.Bool:
		val, err := handleBool(attribute)
		return reflect.ValueOf(val), err
	case reflect.Int:
		val, err := handleInt(attribute)
		return reflect.ValueOf(val), err
	case reflect.Int8:
		val, err := handleInt8(attribute)
		return reflect.ValueOf(val), err
	case reflect.Int16:
		val, err := handleInt16(attribute)
		return reflect.ValueOf(val), err
	case reflect.Int32:
		val, err := handleInt32(attribute)
		return reflect.ValueOf(val), err
	case reflect.Int64:
		val, err := handleInt64(attribute)
		return reflect.ValueOf(val), err
	case reflect.Uint:
		val, err := handleUint(attribute)
		return reflect.ValueOf(val), err
	case reflect.Uint8:
		val, err := handleUint8(attribute)
		return reflect.ValueOf(val), err
	case reflect.Uint16:
		val, err := handleUint16(attribute)
		return reflect.ValueOf(val), err
	case reflect.Uint32:
		val, err := handleUint32(attribute)
		return reflect.ValueOf(val), err
	case reflect.Uint64:
		val, err := handleUint64(attribute)
		return reflect.ValueOf(val), err
	case reflect.Float32:
		val, err := handleFloat32(attribute)
		return reflect.ValueOf(val), err
	case reflect.Float64:
		val, err := handleFloat64(attribute)
		return reflect.ValueOf(val), err
	case reflect.String:
		val, err := handleString(attribute, fieldType, fieldValue)
		return reflect.ValueOf(val), err
	case reflect.Slice:
		switch reflect.TypeOf(fieldValue.Interface()).Elem().Kind() {
		case reflect.Struct:
			return handleStructSlice(attribute, fieldValue)
		default:
			return handleSlice(attribute, args, fieldType, fieldValue)
		}

	case reflect.Ptr:
		return handlePointer(attribute, args, fieldType, fieldValue)
	case reflect.Struct:
		if fieldType.ConvertibleTo(reflect.TypeOf(time.Time{})) {
			return handleTime(attribute, args, fieldValue)
		}
		return handleStruct(attribute,fieldValue)
	}

	return
}

// handleBool
func handleBool(
	attribute interface{}) (bool, error) {
	if val, ok := attribute.(bool); ok {
		return val, nil
	}

	return false, errors.New("invalid value to assign to boolean")
}

func handleNumeric(
	attribute interface{},
	fieldType reflect.Type,
	fieldValue reflect.Value) (reflect.Value, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	var kind reflect.Kind
	if fieldValue.Kind() == reflect.Ptr {
		kind = fieldType.Elem().Kind()
	} else {
		kind = fieldType.Kind()
	}

	var numericValue reflect.Value

	switch kind {
	case reflect.Int:
		n := int(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Int8:
		n := int8(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Int16:
		n := int16(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Int32:
		n := int32(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Int64:
		n := int64(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint:
		n := uint(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint8:
		n := uint8(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint16:
		n := uint16(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint32:
		n := uint32(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint64:
		n := uint64(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Float32:
		n := float32(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Float64:
		n := floatValue
		numericValue = reflect.ValueOf(&n)
	default:
		return reflect.Value{}, ErrUnknownFieldNumberType
	}

	return numericValue, nil
}

// handleInt
func handleInt(attribute interface{}) (int, error) {
	v := reflect.ValueOf(attribute)

	floatValue, ok := v.Interface().(float64)
	if !ok {
		return 0, ErrInvalidType
	}
	return int(floatValue), nil
}

// handleInt8
func handleInt8(attribute interface{}) (int8, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return int8(floatValue), nil
}

// handleInt16
func handleInt16(attribute interface{}) (int16, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return int16(floatValue), nil
}

// handleInt32
func handleInt32(attribute interface{}) (int32, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return int32(floatValue), nil
}

// handleInt64
func handleInt64(attribute interface{}) (int64, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return int64(floatValue), nil
}

// handleUint
func handleUint(attribute interface{}) (uint, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return uint(floatValue), nil
}

// handleUint8
func handleUint8(attribute interface{}) (uint8, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return uint8(floatValue), nil
}

// handleUint16
func handleUint16(attribute interface{}) (uint16, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return uint16(floatValue), nil
}

// handleUint32
func handleUint32(attribute interface{}) (uint32, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return uint32(floatValue), nil
}

// handleUint64
func handleUint64(attribute interface{}) (uint64, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return uint64(floatValue), nil
}

// handleFloat32
func handleFloat32(attribute interface{}) (float32, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return float32(floatValue), nil
}

// handleFloat64
func handleFloat64(attribute interface{}) (float64, error) {
	v := reflect.ValueOf(attribute)
	floatValue := v.Interface().(float64)

	return float64(floatValue), nil
}

// handleString
func handleString(
	attribute interface{},
	fieldType reflect.Type,
	fieldValue reflect.Value) (value string, err error) {
	v := reflect.ValueOf(attribute)
	if v.Kind() != reflect.String {

		return value, errors.New(fmt.Sprintf("can't unmarshal value of type %s to string", v.Kind().String()) )
	}
	value = v.Interface().(string)

	return
}

// handleSlice
func handleSlice(
	attribute interface{},
	args []string,
	fieldType reflect.Type,
	fieldValue reflect.Value) (value reflect.Value, err error) {

	// check passed values is a struct
	submittedValues, ok := attribute.([]interface{})
	if !ok {
		return reflect.Value{}, errors.New("require slice of values to unmarshall into slice")
	}

	// find type tp pass back to handle field - recursively filling the values
	sliceType := fieldType.Elem()

	vals := reflect.MakeSlice(reflect.SliceOf(sliceType), 0, len(submittedValues))

	for _, val := range submittedValues {
		v, tErr := handleField(val, args, sliceType, fieldValue)
		if tErr != nil {
			return reflect.Value{}, tErr
		}

		vals = reflect.Append(vals, v)
	}

	return vals, nil
}

// handlePointer
func handlePointer(
	attribute interface{},
	args []string,
	fieldType reflect.Type,
	fieldValue reflect.Value) (value reflect.Value, err error) {
	t := fieldType.Elem()

	value, err = handleField(attribute, args, t, fieldValue)
	if err != nil {
		return reflect.Value{}, err
	}

	return
}

func handleTime(attribute interface{}, args []string, fieldValue reflect.Value) (reflect.Value, error) {
	var isIso8601 bool
	v := reflect.ValueOf(attribute)

	if len(args) > 2 {
		for _, arg := range args[2:] {
			if arg == annotationISO8601 {
				isIso8601 = true
			}
		}
	}

	if isIso8601 {
		var tm string
		if v.Kind() == reflect.String {
			tm = v.Interface().(string)
		} else {
			return reflect.ValueOf(time.Now()), ErrInvalidISO8601
		}

		t, err := time.Parse(iso8601TimeFormat, tm)
		if err != nil {
			return reflect.ValueOf(time.Now()), ErrInvalidISO8601
		}

		if fieldValue.Kind() == reflect.Ptr {
			return reflect.ValueOf(&t), nil
		}

		return reflect.ValueOf(t), nil
	}

	var at int64

	if v.Kind() == reflect.Float64 {
		at = int64(v.Interface().(float64))
	} else if v.Kind() == reflect.Int {
		at = v.Int()
	} else {
		return reflect.ValueOf(time.Now()), ErrInvalidTime
	}

	t := time.Unix(at, 0)

	return reflect.ValueOf(t), nil
}

func handleStruct(
	attribute interface{},
	fieldValue reflect.Value) (reflect.Value, error) {

	data, err := json.Marshal(attribute)
	if err != nil {
		return reflect.Value{}, err
	}

	node := new(ResourceObj)
	if err := json.Unmarshal(data, &node.Attributes); err != nil {
		return reflect.Value{}, err
	}

	var model reflect.Value
	if fieldValue.Kind() == reflect.Ptr {
		model = reflect.New(fieldValue.Type().Elem())
	} else {
		model = reflect.New(fieldValue.Type())
	}

	if err := unmarshalNode(node, model, nil); err != nil {
		return reflect.Value{}, err
	}

	return model, nil
}

func handleStructSlice(
	attribute interface{},
	fieldValue reflect.Value) (reflect.Value, error) {
	models := reflect.New(fieldValue.Type()).Elem()
	dataMap := reflect.ValueOf(attribute).Interface().([]interface{})
	for _, data := range dataMap {
		model := reflect.New(fieldValue.Type().Elem()).Elem()

		value, err := handleStruct(data, model)

		if err != nil {
			continue
		}

		models = reflect.Append(models, reflect.Indirect(value))
	}

	return models, nil
}

func isCustomStruct(fieldValue reflect.Value) bool {
	t := fieldValue.Type()
	//el := t.Elem()

	targetStruct := reflect.New(t).Interface()

	target, err := json.Marshal(targetStruct)

	var customStruct map[string]interface{}
	err = json.Unmarshal(target, &customStruct)
	if err != nil {
		return false
	}

	return true
}

// attribute interface{},
//	fieldValue reflect.Value) (reflect.Value, error)
func handleCustomStruct(attribute interface{}, fieldValue reflect.Value) (reflect.Value, error) {
	data, err := json.Marshal(attribute)
	if err != nil {
		return reflect.Value{}, err
	}

	t := fieldValue.Type()
	customStruct := reflect.New(t).Interface()
	customErr := json.Unmarshal(data, customStruct)
	if customErr != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(customStruct), nil
}
