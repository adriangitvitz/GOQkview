package utils

//NOTE: This is used to set default values in a struct

import (
	"fmt"
	"reflect"
)

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func Setfield(field reflect.Value, defaultval string) error {
	if !field.CanSet() {
		return &Error{
			Message: "Cant set value",
		}
	}
	switch field.Kind() {
	case reflect.String:
		field.Set(reflect.ValueOf(defaultval).Convert(field.Type()))
	default:
		return &Error{
			Message: fmt.Sprintf("Unimplemented type: %s", field.Type()),
		}
	}
	return nil
}

func Setdefault(p interface{}, tag string) error {
	if reflect.TypeOf(p).Kind() != reflect.Ptr {
		return &Error{
			Message: "Bad structure pointer",
		}
	}
	v := reflect.ValueOf(p).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		if defaultval := t.Field(i).Tag.Get(tag); defaultval != "" {
			if err := Setfield(v.Field(i), defaultval); err != nil {
				return err
			}
		}
	}
	return nil
}
