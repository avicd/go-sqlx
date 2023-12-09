package scan

import (
	"database/sql"
	"github.com/avicd/go-utilx/conv"
	"github.com/avicd/go-utilx/refx"
	"reflect"
)

func intoObject(rows *sql.Rows, el reflect.Value, colMap map[string][]string) error {
	var dest []any
	cols, _ := rows.Columns()
	colTypes, _ := rows.ColumnTypes()
	mapsToSet := map[string]reflect.Value{}
	mapValKeys := map[string]string{}
	mapValues := map[string]reflect.Value{}
	sliceToSet := map[string]reflect.Value{}
	sliceValues := map[string]reflect.Value{}
	for ci, col := range cols {
		colType := colTypes[ci]
		var chain []string
		if keys, hit := colMap[col]; hit && len(keys) > 0 {
			chain = keys
		} else if el.Kind() == reflect.Struct {
			chain = []string{conv.BigCamelCase(col)}
		} else if el.Kind() == reflect.Map {
			chain = []string{col}
		}
		buf := el
		var pf reflect.Value
		hit := true
		for ni, name := range chain {
			if buf.Kind() == reflect.Struct {
				pf = buf.FieldByName(name)
			} else if buf.Kind() == reflect.Map {
				pf = buf
			}
			if pf.Kind() == reflect.Map {
				if pf.IsNil() {
					pf.Set(refx.NewOf(pf.Type()))
				}
				if len(chain)-ni > 1 {
					if pf.Type().Elem().Kind() == reflect.Interface {
						val := refx.NewOf(reflect.TypeOf(refx.TMapStrAny))
						pf.SetMapIndex(reflect.ValueOf(name), val)
						buf = refx.Indirect(val)
						continue
					} else {
						hit = false
						break
					}
				} else {
					var val reflect.Value
					if pf.Type().Elem().Kind() == reflect.Interface {
						val = refx.NewOf(GetDfType(colType.DatabaseTypeName()))
					} else {
						val = refx.NewOf(pf.Type().Elem())
					}
					mapsToSet[col] = pf
					mapValKeys[col] = name
					mapValues[col] = val
					buf = refx.Indirect(val)
					break
				}
			} else if pf.Kind() == reflect.Slice {
				if ni+1 < len(chain) {
					hit = false
					break
				} else {
					var val reflect.Value
					if pf.Type().Elem().Kind() == reflect.Interface {
						val = refx.NewOf(GetDfType(colType.DatabaseTypeName()))
					} else {
						val = refx.NewOf(pf.Type().Elem())
					}
					sliceToSet[col] = pf
					sliceValues[col] = val
					buf = refx.Indirect(val)
					break
				}
			} else if pf.Kind() == reflect.Pointer && pf.IsNil() {
				val := refx.NewOf(pf.Type())
				pf.Set(val)
			}

			if pf.IsValid() && pf.CanAddr() {
				buf = refx.Indirect(pf)
			} else {
				hit = false
				break
			}
		}

		if hit && buf.IsValid() && buf.CanAddr() {
			dest = append(dest, buf.Addr().Interface())
		} else {
			dest = append(dest, new(any))
		}
	}
	err := rows.Scan(dest...)
	if err != nil {
		return err
	}
	if len(mapsToSet) > 0 {
		for k, mapToSet := range mapsToSet {
			mapToSet.SetMapIndex(refx.ValueOf(mapValKeys[k]), mapValues[k])
		}
	}
	if len(sliceToSet) > 0 {
		for k, list := range sliceToSet {
			tmp := reflect.Append(list, sliceValues[k])
			list.Set(tmp)
		}
	}
	return nil
}

func intoBasic(rows *sql.Rows, el reflect.Value) error {
	var dest []any
	cols, _ := rows.Columns()
	for idx := range cols {
		if idx == 0 && el.CanAddr() {
			dest = append(dest, el.Addr().Interface())
		} else {
			dest = append(dest, new(any))
		}
	}
	err := rows.Scan(dest...)
	if err != nil {
		return err
	}
	return nil
}

func Read(rows *sql.Rows, tp reflect.Type, colMap map[string][]string) (reflect.Value, error) {
	return scanRows(rows, tp, colMap, false)
}

func scanRows(rows *sql.Rows, target reflect.Type, colMap map[string][]string, mul bool) (reflect.Value, error) {
	vl := refx.NewOf(target)
	el := refx.Indirect(vl)
	hasRs := false
	switch el.Kind() {
	case reflect.Slice:
		for rows.Next() {
			hasRs = true
			rs, err := scanRows(rows, el.Type().Elem(), colMap, true)
			if err != nil {
				return refx.ZeroOf(target), err
			}
			nel := reflect.Append(el, rs)
			el.Set(nel)
		}
	case reflect.Struct, reflect.Map:
		if mul || !mul && rows.Next() {
			hasRs = true
			err := intoObject(rows, el, colMap)
			if err != nil {
				return refx.ZeroOf(target), err
			}
		}
	default:
		if mul || !mul && rows.Next() {
			hasRs = true
			err := intoBasic(rows, el)
			if err != nil {
				return refx.ZeroOf(target), err
			}
		}
	}
	if !hasRs && refx.IsPointer(target) {
		return refx.ZeroOf(target), nil
	}
	return vl, nil
}
