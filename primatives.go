package jsonapi

import (
	"encoding/json"
	"fmt"
)

// Intger

// JSONInt a struct to aide representation of int in json
type JSONInt struct {
	Null  bool
	Set   bool
	Value int
}

// UnmarshalJSON unmarshalls an integer jsonfield into a JSONInt
func (i *JSONInt) UnmarshalJSON(data []byte) error {
	// If this method was called, the value was set.
	i.Set = true

	if data == nil {
		// The key was set to null
		i.Null = true
		return nil
	}

	// The key isn't set to null
	var temp int
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	i.Value = temp
	return nil
}

// JSONInt8 a struct to aide representation of int8 in json
type JSONInt8 struct {
	Null  bool
	Set   bool
	Value int8
}

// UnmarshalJSON unmarshalls an integer jsonfield into a JSONInt8
func (i *JSONInt8) UnmarshalJSON(data []byte) error {
	// If this method was called, the value was set.
	i.Set = true

	if data == nil {
		// The key was set to null
		i.Null = true
		return nil
	}

	// The key isn't set to null
	var temp int8
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	i.Value = temp
	return nil
}

// JSONInt32 a struct to aide representation of int32 in json
type JSONInt32 struct {
	Null  bool
	Set   bool
	Value int32
}

// UnmarshalJSON unmarshalls an integer jsonfield into a JSONInt32
func (i *JSONInt32) UnmarshalJSON(data []byte) error {
	// If this method was called, the value was set.
	i.Set = true

	if data == nil {
		// The key was set to null
		i.Null = true
		return nil
	}

	// The key isn't set to null
	var temp int32
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	i.Value = temp
	return nil
}

// JSONInt64 a struct to aide representation of int64 in json
type JSONInt64 struct {
	Null  bool
	Set   bool
	Value int64
}

// UnmarshalJSON unmarshalls an integer jsonfield into a JSONInt64
func (i *JSONInt64) UnmarshalJSON(data []byte) error {
	// If this method was called, the value was set.
	i.Set = true

	if data == nil {
		// The key was set to null
		i.Null = true
		return nil
	}

	// The key isn't set to null
	var temp int64
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	i.Value = temp
	return nil
}

// Unsigned Integer

// JSONUInt a struct to aide representation of uint in json
type JSONUInt struct {
	Null  bool
	Set   bool
	Value uint
}

// UnmarshalJSON unmarshalls an unsigned integer jsonfield into a JSONUInt
func (i *JSONUInt) UnmarshalJSON(data []byte) error {
	// If this method was called, the value was set.
	i.Set = true

	if data == nil {
		// The key was set to null
		i.Null = true
		return nil
	}

	// The key isn't set to null
	var temp uint
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	i.Value = temp
	return nil
}

// JSONUInt32 a struct to aide representation of uint32 in json
type JSONUInt32 struct {
	Null  bool
	Set   bool
	Value uint32
}

// UnmarshalJSON unmarshalls an unsigned integer jsonfield into a JSONUInt32
func (i *JSONUInt32) UnmarshalJSON(data []byte) error {
	// If this method was called, the value was set.
	i.Set = true

	if data == nil {
		// The key was set to null
		i.Null = true
		return nil
	}

	// The key isn't set to null
	var temp uint32
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	i.Value = temp
	return nil
}

// JSONInt64 a struct to aide representation of uint64 in json
type JSONUInt64 struct {
	Null  bool
	Set   bool
	Value uint64
}

// UnmarshalJSON unmarshalls an unsigned integer jsonfield into a JSONInt64
func (i *JSONUInt64) UnmarshalJSON(data []byte) error {
	// If this method was called, the value was set.
	i.Set = true

	if data == nil {
		// The key was set to null
		i.Null = true
		return nil
	}

	// The key isn't set to null
	var temp uint64
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	i.Value = temp
	return nil
}

// JSONString a struct to aide representation of string in json
type JSONString struct {
	Null  bool
	Set   bool
	Value string
}

// UnmarshalJSON unmarshalls an integer jsonfield into a JSONInt
func (i *JSONString) UnmarshalJSON(data []byte) error {
	// If this method was called, the value was set.
	i.Set = true

	if data == nil {
		// The key was set to null
		i.Null = true
		return nil
	}

	// The key isn't set to null
	var temp string
	if err := json.Unmarshal(data, &temp); err != nil {
		fmt.Printf("error decoding sakura response: %v", err)
		if e, ok := err.(*json.SyntaxError); ok {
			fmt.Printf("syntax error at byte offset %d", e.Offset)
		}
		fmt.Printf("sakura response: %q", data)
		return err
	}
	i.Value = temp
	return nil
}