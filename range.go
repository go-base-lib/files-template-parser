package templateparser

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

func rangeInterface(rangeData []interface{}, data map[string]interface{}, callBack func() error, keyIndex int) error {
	if len(rangeData) == 0 {
		return callBack()
	}

	keyIndexStr := strconv.Itoa(keyIndex)
	dataKey := "k" + keyIndexStr
	valKey := "v" + keyIndexStr

	defer func() {
		delete(data, dataKey)
		delete(data, valKey)
	}()
	keyIndex += 1

	rd := rangeData[0]
	rangeData = rangeData[1:]

	if orderFieldMap, ok := rd.(*OrderFieldMap); ok {
		keys := orderFieldMap.Keys()
		if keys == nil || len(keys) == 0 {
			return errors.New(fmt.Sprintf("第%s个range变量为空: ", keyIndexStr))
		}
		for _, k := range keys {
			val, _ := orderFieldMap.Get(k)
			data[dataKey] = k
			data[valKey] = val
			if err := rangeInterface(rangeData, data, callBack, keyIndex); err != nil {
				return err
			}
		}
		return nil
	}

	t := reflect.TypeOf(rd)
	v := reflect.ValueOf(rd)
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		valKeyLen := v.Len()
		for i := 0; i < valKeyLen; i++ {
			data[dataKey] = i
			data[valKey] = v.Index(i).Interface()
			if err := rangeInterface(rangeData, data, callBack, keyIndex); err != nil {
				return err
			}
		}
	case reflect.Map:
		mapRange := v.MapRange()
		for mapRange.Next() {
			key := mapRange.Key()
			val := mapRange.Value()
			data[dataKey] = key.Interface()
			data[valKey] = val.Interface()
			if err := rangeInterface(rangeData, data, callBack, keyIndex); err != nil {
				return err
			}
		}
	case reflect.Struct:
		valKeyLen := v.NumField()
		for i := 0; i < valKeyLen; i++ {
			field := v.Field(i)
			fieldName := t.Field(i).Name
			data[dataKey] = fieldName
			data[valKey] = field.Interface()
			if err := rangeInterface(rangeData, data, callBack, keyIndex); err != nil {
				return err
			}
		}
	default:
		return errors.New(fmt.Sprintf("第%s个range变量为空: ", keyIndexStr))
	}
	return nil
}
