package session

import (
	"database/sql"
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-sqlx/script"
	"github.com/avicd/go-utilx/conv"
	"github.com/avicd/go-utilx/evalx"
	"github.com/avicd/go-utilx/refx"
	"github.com/avicd/go-utilx/tokx"
	"reflect"
	"strings"
)

type StmtType uint

const (
	Select StmtType = iota
	SelectSet
	Insert
	Update
	Delete
)

type Target uint

const (
	ARG Target = iota
	RESULT
	CONTEXT
)

const SelectSetSuffix = "!SelectSet"

type Stmt struct {
	config     *Config
	Id         string
	Ns         string
	DbId       string
	SqlNode    script.SqlNode
	Dynamic    bool
	RsMaps     []RsMap
	StmtType   StmtType
	ArgNames   []string
	KeyProps   []string
	KeyColumns []string
	Target     Target
	Bind       string
	SetStmts   []*Stmt
	staticSql  string
}

type SetProxy struct {
	Invoker reflect.Value
	Props   []any
	Bind    string
	Basic   bool
	SetMap  map[string][]string
}

func StmtIdOf(id string) string {
	if id == "" {
		return id
	}
	if strings.Index(id, ".") < 0 {
		id = NameSpace + "." + id
	}
	return id
}

func (it *Stmt) DB() *sql.DB {
	if it.DbId != "" {
		return it.config.GetDB(it.Id)
	} else {
		return it.config.MainDB()
	}
}

func (it *Stmt) EvalSql(builder *script.SqlBuilder) (string, []any) {
	var sqlStr string
	var sqlArgs []any
	if it.Dynamic || it.staticSql == "" {
		sqlStr = builder.Build(it.SqlNode)
		if !it.Dynamic {
			it.staticSql = sqlStr
		}
	} else {
		sqlStr = it.staticSql
	}
	sqlStr = tokx.NewPair("#{", "}").Map(sqlStr, func(expr string) string {
		val := builder.Eval(expr)
		sqlArgs = append(sqlArgs, val)
		return "?"
	})
	return sqlStr, sqlArgs
}

func (it *Stmt) proxy() *Proxy {
	return &Proxy{
		config:   it.config,
		stmt:     it,
		argNames: it.ArgNames,
	}
}

func (it *Stmt) tagsRsMap(rsMap RsMap, tp reflect.Type) {
	tp = refx.IndirectType(tp)
	for i := 0; i < tp.NumField(); i++ {
		tf := tp.Field(i)
		if !tf.IsExported() {
			continue
		}
		tags := tokx.ParseTags(tf.Tag.Get(NameSpace))
		if cols, ok := tags[tp.Name()]; ok && len(cols) > 0 {
			props := refx.AsListOf[string](tp.Name())
			if refx.IsSlice(tf.Type) {
				for _, column := range cols {
					rsMap[column] = props
				}
			} else {
				rsMap[cols[0]] = props
			}
		}
	}
}

func (it *Stmt) ProxyOf(target reflect.Type, args ...string) *Proxy {
	if !refx.IsFunc(target) {
		logger.Fatal("target type is not function")
	}
	proxy := it.proxy()
	if len(args) > 0 {
		proxy.argNames = args
	}
	proxy.target = target
	inIndexMap := map[string]int{}
	outIndexMap := map[string]int{}
	for i := 0; i < target.NumIn(); i++ {
		inIndexMap[InName(i, proxy.argNames)] = i
		proxy.ins = append(proxy.ins, target.In(i))
	}
	var rsMaps []RsMap
	for i := 0; i < target.NumOut(); i++ {
		outIndexMap[OutName(i)] = i
		outType := target.Out(i)
		proxy.outs = append(proxy.outs, outType)
		if refx.IsError(outType) {
			proxy.outErrs++
			continue
		}
		rsMap := RsMap{}
		if i < len(it.RsMaps) {
			refx.Clone(&rsMap, it.RsMaps[i-proxy.outErrs])
		}
		if refx.IsStruct(outType) {
			it.tagsRsMap(rsMap, outType)
		}
		rsMaps = append(rsMaps, rsMap)
	}
	proxy.rsMaps = rsMaps
	argIndex := 0
	rsIndex := 0
	proxy.argSets = map[string][]*selectSet{}
	proxy.outSets = map[string][]*selectSet{}
	for _, setStmt := range it.SetStmts {
		var outTypes []reflect.Type
		var indexMap map[string]int
		var proxyMap map[string][]*selectSet
		var outType reflect.Type
		var targetType reflect.Type
		var pIns []reflect.Type
		var pOuts []reflect.Type
		var scope *evalx.Scope
		bind := setStmt.Bind
		bindIndex := -1
		if bind != "" && conv.IsDigit(bind) {
			bindIndex = int(conv.ParseInt(bind))
		}
		switch setStmt.Target {
		case ARG:
			outTypes = proxy.ins
			proxyMap = proxy.argSets
			indexMap = inIndexMap
			pIns = append(pIns, proxy.ins...)
			pIns = append(pIns, refx.TypeOf(scope))
			if bind == "" {
				bind = InName(argIndex, proxy.argNames)
				argIndex++
			} else if bindIndex > -1 {
				bind = InName(bindIndex, proxy.argNames)
			}
		case RESULT:
			outTypes = proxy.outs
			proxyMap = proxy.outSets
			indexMap = outIndexMap
			pIns = append(pIns, proxy.outs...)
			pIns = append(pIns, proxy.ins...)
			pIns = append(pIns, refx.TypeOf(scope))
			if bind == "" {
				bind = OutName(rsIndex)
				rsIndex++
			} else if bindIndex > -1 {
				bind = OutName(bindIndex)
			}
		case CONTEXT:
			outType = reflect.TypeOf(refx.TMapStrAny)
			pIns = append(pIns, proxy.ins...)
			pIns = append(pIns, refx.TypeOf(scope))
			pOuts = []reflect.Type{outType}
			targetType = reflect.FuncOf(pIns, pOuts, false)
			setProxy := setStmt.ProxyOf(targetType)
			proxy.ctxSets = append(proxy.ctxSets, &selectSet{
				Invoker: setProxy.Invoker(),
				Bind:    setStmt.Bind,
			})
			continue
		}
		if bi, ok := indexMap[bind]; !ok {
			logger.Errorf("can't match a target for selectSet statement '%s'", setStmt.Id)
		} else {
			outType = outTypes[bi]
		}
		proxySet := &selectSet{}
		if refx.IsStruct(outType) || refx.IsMap(outType) {
			if len(setStmt.RsMaps) > 0 && len(setStmt.RsMaps[0]) == 1 {
				for _, names := range setStmt.RsMaps[0] {
					props := refx.AsList(names)
					if val, ok := refx.TypeOfField(outType, props...); ok {
						pOuts = refx.AsListOf[reflect.Type](val)
						proxySet.Props = props
					}
				}
			} else if len(setStmt.KeyProps) == 1 {
				if val, ok := refx.TypeOfId(outType, setStmt.KeyProps[0]); ok {
					pOuts = refx.AsListOf[reflect.Type](val)
					proxySet.Props = refx.AsList(conv.StrToArr(setStmt.KeyProps[0], "."))
				}
			}
			if pOuts == nil {
				pOuts = refx.AsListOf[reflect.Type](outType)
				if len(setStmt.RsMaps) > 0 {
					proxySet.SetMap = setStmt.RsMaps[0]
				}
			}
		} else {
			pOuts = refx.AsListOf[reflect.Type](outType)
			proxySet.Basic = true
		}
		targetType = reflect.FuncOf(pIns, pOuts, false)
		setProxy := setStmt.ProxyOf(targetType)
		proxySet.Invoker = setProxy.Invoker()
		proxyMap[bind] = append(proxyMap[bind], proxySet)
	}
	return proxy
}

func (it *Stmt) ProxyOfField(tf reflect.StructField) *Proxy {
	tags := tokx.ParseTags(tf.Tag.Get(NameSpace))
	args := tags["args"]
	return it.ProxyOf(tf.Type, args...)
}

func (it *Stmt) Scan(dest any) {
	if !refx.IsFunc(dest) {
		logger.Fatal("dest is not function")
	}
	if target, ok := refx.AddrAbleOf(dest); ok {
		proxy := it.ProxyOf(refx.TypeOf(target))
		target.Set(proxy.Invoker())
	} else {
		logger.Fatal("dest is unaddressable")
	}
}
