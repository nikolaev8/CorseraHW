package main

import (
	"fmt"
	"reflect"
)





func i2s(data interface{}, out interface{}) error {

	outVal := reflect.ValueOf(out)
	if reflect.TypeOf(out).Kind() != reflect.Ptr{
		return fmt.Errorf("Out argument must be a pointer, not a %v ", reflect.TypeOf(out).Kind())
	}

	outElem := outVal.Elem()

	if !outElem.CanSet() {
		// Вернуть ошибку если не обладает свойством устанавливаемости
		return fmt.Errorf("Out argument %v must can be changed ", outVal)
	}

	dataVal := reflect.ValueOf(data)

	switch outElem.Kind(){
	case reflect.Struct:

		if reflect.TypeOf(data).Kind() != reflect.Map{
			return fmt.Errorf("Expected map[string]interface{} in input data, not %v ", reflect.TypeOf(data).Kind())
		}

		for i := 0; i < outElem.NumField(); i++ {
			fieldType := outElem.Type().Field(i)
			fieldValue := outElem.Field(i)


			val := dataVal.MapIndex(reflect.ValueOf(fieldType.Name))
			valElem := val.Elem()

			switch fieldType.Type.Kind() {
			case reflect.Struct:
				if valElem.Type().Kind() != reflect.Map{
					return fmt.Errorf("Expected Struct, got %v ", valElem.Type().Kind())
				}
				err := i2s(valElem.Interface(), fieldValue.Addr().Interface())
				if err != nil {
					return err
				}
			case reflect.Float64:
				if valElem.Type().Kind() != reflect.Float64{
					return fmt.Errorf("Expected Float64, got %v ", valElem.Type().Kind())
				}
				fieldValue.SetFloat(valElem.Float())
			case reflect.String:
				if valElem.Type().Kind() != reflect.String{
					return fmt.Errorf("Expected String, got %v ", valElem.Type().Kind())
				}
				fieldValue.SetString(valElem.String())
			case reflect.Bool:
				if valElem.Type().Kind() != reflect.Bool{
					return fmt.Errorf("Expected Bool, got %v ", valElem.Type().Kind())
				}
				fieldValue.SetBool(valElem.Bool())
			case reflect.Int:
				if valElem.Type().Kind() != reflect.Float64{
					return fmt.Errorf("Expected Float64, got %v ", valElem.Type().Kind())
				}
				fieldValue.SetInt(int64(valElem.Float()))
			case reflect.Slice:
				if valElem.Type().Kind() != reflect.Slice{
					return fmt.Errorf("Expected Slice, got %v ", valElem.Type().Kind())
				}

				fieldsSlice := reflect.MakeSlice(reflect.SliceOf(fieldType.Type.Elem()), valElem.Len(), valElem.Cap() )

				for j := 0; j < valElem.Len(); j++ {
					err := i2s(val.Elem().Index(j).Interface(), fieldsSlice.Index(j).Addr().Interface())
					if err != nil {
						return err
					}
				}
				fieldValue.Set(fieldsSlice)
			}

		}
	case reflect.Slice:

		if reflect.TypeOf(data).Kind() != reflect.Slice{
			return fmt.Errorf("Expected slice in input data, not %v ", reflect.TypeOf(data).Kind())
		}

		structSlice := reflect.MakeSlice(outElem.Type(), dataVal.Len(), dataVal.Cap() )

		for i :=0; i < dataVal.Len(); i++ {
			err := i2s(dataVal.Index(i).Interface(), structSlice.Index(i).Addr().Interface())
			if err != nil {
				return err
			}
		}
		outElem.Set(structSlice)
	default:
		return fmt.Errorf("Unknown out type ")
	}
	return nil

}
