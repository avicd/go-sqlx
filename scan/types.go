package scan

import (
	"github.com/avicd/go-utilx/refx"
	"reflect"
)

func GetDfType(dt string) reflect.Type {
	switch dt {
	case "BOOL":
		return reflect.TypeOf(refx.TBool)
	case "BIT":
		return reflect.TypeOf(refx.TByte)
	case "BINARY", "VARBINARY", "LONG VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB":
		return reflect.TypeOf(refx.TBytes)
	case "BIGINT":
		return reflect.TypeOf(refx.TInt64)
	case "INTEGER", "INT", "MEDIUMINT":
		return reflect.TypeOf(refx.TInt32)
	case "SMALLINT":
		return reflect.TypeOf(refx.TInt16)
	case "TINYINT":
		return reflect.TypeOf(refx.TInt8)
	case "BIGINT UNSIGNED":
		return reflect.TypeOf(refx.TUint64)
	case "INTEGER UNSIGNED", "INT UNSIGNED", "MEDIUMINT UNSIGNED":
		return reflect.TypeOf(refx.TUint32)
	case "SMALLINT UNSIGNED":
		return reflect.TypeOf(refx.TUint16)
	case "TINYINT UNSIGNED":
		return reflect.TypeOf(refx.TUint8)
	case "CHAR", "VARCHAR", "LONG VARCHAR", "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT", "ENUM", "SET", "JSON":
		return reflect.TypeOf(refx.TString)
	case "DATE", "TIME", "DATETIME", "TIMESTAMP", "YEAR":
		return reflect.TypeOf(refx.TString)
	case "FLOAT":
		return reflect.TypeOf(refx.TFloat32)
	case "REAL", "DOUBLE", "DOUBLE PRECISION", "NUMERIC", "DECIMAL":
		return reflect.TypeOf(refx.TFloat64)
	}
	return reflect.TypeOf(refx.TAny)
}
