package file

import "reflect"

type ExcelData struct {
	Columns []string
	Data map[string][]interface{}
}

func (excelData *ExcelData) SetData(column string, data interface{}) {
	val, ok := CreateAnyTypeSlice(data)
	if !ok {
		panic("数据设置错误")
	}
	excelData.Data[column] = val
}

func CreateAnyTypeSlice(slice interface{}) ([]interface{}, bool) {
	val, ok := isSlice(slice)

	if !ok {
		return nil, false
	}

	sliceLen := val.Len()

	out := make([]interface{}, sliceLen)

	for i := 0; i < sliceLen; i++ {
		out[i] = val.Index(i).Interface()
	}

	return out, true
}

// 判断是否为slice数据
func isSlice(arg interface{}) (val reflect.Value, ok bool) {
	val = reflect.ValueOf(arg)

	if val.Kind() == reflect.Slice {
		ok = true
	}

	return
}

