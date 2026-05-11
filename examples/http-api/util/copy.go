package util

import (
	"database/sql"
	"reflect"
	"sync"
	"time"
)

var fieldCache sync.Map // map[reflect.Type]map[string]int

// Copy 复制同名字段从 src 到 dst（dst 必须是指针）
func Copy(dst, src any) {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return
	}
	dstVal = dstVal.Elem()
	srcVal := reflect.ValueOf(src)
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.IsNil() {
			return
		}
		srcVal = srcVal.Elem()
	}

	dstFields := getFieldMap(dstVal.Type())
	srcFields := getFieldMap(srcVal.Type())

	for name, dstIdx := range dstFields {
		srcIdx, ok := srcFields[name]
		if !ok {
			continue
		}
		dstField := dstVal.Field(dstIdx)
		srcField := srcVal.Field(srcIdx)
		if !dstField.CanSet() {
			continue
		}
		copyField(dstField, srcField)
	}
}

func getFieldMap(t reflect.Type) map[string]int {
	if cached, ok := fieldCache.Load(t); ok {
		return cached.(map[string]int)
	}
	m := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		m[t.Field(i).Name] = i
	}
	fieldCache.Store(t, m)
	return m
}

func copyField(dst, src reflect.Value) {
	srcType := src.Type()
	dstType := dst.Type()

	// 类型相同直接赋值
	if srcType == dstType {
		dst.Set(src)
		return
	}

	// sql.Null* -> 基本类型
	if srcType.PkgPath() == "database/sql" {
		switch srcType.Name() {
		case "NullString":
			if dstType.Kind() == reflect.String {
				dst.SetString(src.Interface().(sql.NullString).String)
			}
		case "NullInt64":
			if dstType.Kind() == reflect.Int64 {
				dst.SetInt(src.Interface().(sql.NullInt64).Int64)
			}
		case "NullFloat64":
			if dstType.Kind() == reflect.Float64 {
				dst.SetFloat(src.Interface().(sql.NullFloat64).Float64)
			}
		case "NullBool":
			if dstType.Kind() == reflect.Bool {
				dst.SetBool(src.Interface().(sql.NullBool).Bool)
			}
		case "NullTime":
			if dstType == reflect.TypeOf(time.Time{}) {
				dst.Set(reflect.ValueOf(src.Interface().(sql.NullTime).Time))
			}
		}
		return
	}

	// 基本类型 -> sql.Null*
	if dstType.PkgPath() == "database/sql" {
		switch dstType.Name() {
		case "NullString":
			if srcType.Kind() == reflect.String {
				dst.Set(reflect.ValueOf(sql.NullString{String: src.String(), Valid: src.String() != ""}))
			}
		case "NullInt64":
			if srcType.Kind() == reflect.Int64 {
				dst.Set(reflect.ValueOf(sql.NullInt64{Int64: src.Int(), Valid: true}))
			}
		case "NullFloat64":
			if srcType.Kind() == reflect.Float64 {
				dst.Set(reflect.ValueOf(sql.NullFloat64{Float64: src.Float(), Valid: true}))
			}
		case "NullBool":
			if srcType.Kind() == reflect.Bool {
				dst.Set(reflect.ValueOf(sql.NullBool{Bool: src.Bool(), Valid: true}))
			}
		case "NullTime":
			if srcType == reflect.TypeOf(time.Time{}) {
				t := src.Interface().(time.Time)
				dst.Set(reflect.ValueOf(sql.NullTime{Time: t, Valid: !t.IsZero()}))
			}
		}
		return
	}

	// 可转换类型
	if srcType.ConvertibleTo(dstType) {
		dst.Set(src.Convert(dstType))
	}
}
