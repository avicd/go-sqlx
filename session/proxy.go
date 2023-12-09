package session

import (
	"fmt"
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-sqlx/scan"
	"github.com/avicd/go-sqlx/script"
	"github.com/avicd/go-utilx/evalx"
	"github.com/avicd/go-utilx/refx"
	"reflect"
	"strings"
)

type Proxy struct {
	config   *Config
	stmt     *Stmt
	target   reflect.Type
	ins      []reflect.Type
	outs     []reflect.Type
	outErrs  int
	rsMaps   []RsMap
	argNames []string
	argSets  map[string][]*selectSet
	outSets  map[string][]*selectSet
	ctxSets  []*selectSet
}

func newPayload(proxy *Proxy, args []reflect.Value) *Payload {
	var results []reflect.Value
	for _, out := range proxy.outs {
		results = append(results, refx.ZeroOf(out))
	}
	return &Payload{
		proxy:   proxy,
		extra:   map[string]any{},
		builder: script.NewSqlBuilder(),
		Stmt:    proxy.stmt,
		Args:    args,
		Result:  results,
	}
}

type selectSet struct {
	Invoker reflect.Value
	Props   []any
	Bind    string
	Basic   bool
	SetMap  map[string][]string
}

type Payload struct {
	canceled     bool
	proxy        *Proxy
	session      *Session
	extra        map[string]any
	builder      *script.SqlBuilder
	Stmt         *Stmt
	Args         []reflect.Value
	Sql          string
	SqlArgs      []any
	Result       []reflect.Value
	LastInsertId int64
}

func (it *Payload) mergeExtra() {
	if len(it.extra) > 0 {
		it.builder.Merge(it.extra)
		it.extra = map[string]any{}
	}
}

func (it *Payload) Cancel() {
	it.canceled = true
}

func (it *Payload) Put(key string, value any) {
	it.extra[key] = value
}

func (it *Payload) PutAll(val map[string]any) {
	refx.Merge(&it.extra, val)
}

func (it *Payload) setError(rc any) {
	for i, out := range it.proxy.outs {
		if refx.IsError(out) {
			newVal := refx.NewOf(out)
			refx.Assign(newVal, refx.AsError(rc))
			it.Result[i] = newVal
		}
	}
}

func InName(i int, names []string) string {
	if i < len(names) && i >= 0 {
		return names[i]
	} else if i == 0 {
		return "value"
	} else {
		return fmt.Sprintf("arg%d", i)
	}
}

func OutName(i int) string {
	if i == 0 {
		return "out"
	} else {
		return fmt.Sprintf("out%d", i)
	}
}

func (it *Proxy) Invoker() reflect.Value {
	return reflect.MakeFunc(it.target, func(args []reflect.Value) (results []reflect.Value) {
		payload := newPayload(it, args)
		it.invoke(payload)
		return payload.Result
	})
}

func (it *Proxy) applySelectSet(argType reflect.Type, list []*selectSet, args []reflect.Value, index int) {
	arg := args[index]
	for _, setProxy := range list {
		if !setProxy.Basic && len(setProxy.Props) < 1 && len(setProxy.SetMap) < 1 {
			continue
		}
		values := setProxy.Invoker.Call(args)
		if len(values) < 1 {
			continue
		}
		result := values[0]
		if setProxy.Basic {
			args[index] = result
		} else {
			if refx.IsNil(arg) {
				args[index] = refx.NewOf(argType)
				arg = args[index]
			}
			if len(setProxy.Props) > 0 {
				refx.Set(arg, result, setProxy.Props...)
			} else {
				for _, keys := range setProxy.SetMap {
					props := refx.AsList(keys)
					if val, exist := refx.PropOf(result, props...); exist {
						refx.Set(arg, val, props...)
					}
				}
			}
		}
	}
}

func (it *Proxy) processArgs(payload *Payload) {
	it.applyHook(payload, ProcessArgs, Before)
	var inArgs []reflect.Value
	inArgs = append(inArgs, payload.Args...)
	inArgs = append(inArgs, refx.ValueOf(payload.builder.Scope))
	argsMap := map[string]any{}
	if len(it.ins) > 0 {
		for i, in := range it.ins {
			if scope, ok := inArgs[i].Interface().(*evalx.Scope); ok {
				payload.builder.Merge(scope)
				continue
			}
			name := InName(i, it.argNames)
			if list, ok := it.argSets[name]; ok {
				it.applySelectSet(in, list, inArgs, i)
			}
			argsMap[name] = inArgs[i].Interface()
			payload.Args[i] = inArgs[i]
		}
	}
	for _, setProxy := range it.ctxSets {
		values := setProxy.Invoker.Call(inArgs)
		if len(values) > 0 {
			vl := values[0]
			if setProxy.Bind != "" {
				payload.builder.Bind(setProxy.Bind, vl)
			} else {
				payload.builder.Merge(vl)
			}
		}
	}
	it.applyHook(payload, ProcessArgs, After)
	if len(payload.Args) < 1 {
		return
	}
	if len(payload.Args) == 1 && !refx.IsBasic(payload.Args[0]) {
		payload.builder.Bind(InName(0, it.argNames), payload.Args[0])
		payload.builder.Link(payload.Args[0])
	} else {
		payload.builder.Link(argsMap)
	}
}

func (it *Proxy) processResults(payload *Payload) {
	it.applyHook(payload, ProcessResults, Before)
	if len(it.outs) > 0 {
		var outArgs []reflect.Value
		outArgs = append(outArgs, payload.Result...)
		outArgs = append(outArgs, payload.Args...)
		outArgs = append(outArgs, refx.ValueOf(payload.builder.Scope))
		for i, out := range it.outs {
			name := OutName(i)
			if list, ok := it.outSets[name]; ok {
				it.applySelectSet(out, list, outArgs, i)
			}
			payload.Result[i] = outArgs[i]
		}
	}
	it.applyHook(payload, ProcessResults, After)
}

func (it *Proxy) applyHook(payload *Payload, hook Hook, order Order) {
	if len(it.config.plugins) < 1 {
		return
	}
	for _, plugin := range it.config.plugins {
		if (plugin.Hook() == Always || plugin.Hook() == hook) && (plugin.Order() == order) {
			if !plugin.Intercept(payload) {
				break
			}
		}
	}
	payload.mergeExtra()
}

func (it *Proxy) processKeyProps(payload *Payload) {
	if payload.Stmt.StmtType != Insert {
		return
	}
	if len(payload.Stmt.KeyProps) < 1 {
		return
	}
	if payload.LastInsertId < 1 {
		return
	}
	for _, arg := range payload.Args {
		if refx.IsBasic(arg) {
			continue
		}
		if refx.SetById(arg, payload.LastInsertId, payload.Stmt.KeyProps[0]) {
			break
		}
	}
}

func (it *Proxy) invoke(payload *Payload) {
	factory := it.config.Factory()
	defer func() {
		if rc := recover(); rc != nil {
			factory.Keeper().Rollback()
			if it.outErrs > 0 {
				payload.setError(rc)
			} else {
				logger.Error(rc)
			}
		}
	}()
	it.processArgs(payload)
	payload.Sql, payload.SqlArgs = it.stmt.EvalSql(payload.builder)
	it.applyHook(payload, SqlQuery, Before)
	if !payload.canceled {
		payload.session = factory.Open()
		if logger.IsDebug() {
			format := fmt.Sprintf("--> %s:\n", it.stmt.Id)
			format += strings.ReplaceAll(payload.Sql, "?", "%v")
			logger.Debugf(format, payload.SqlArgs...)
		}
		switch payload.Stmt.StmtType {
		case Select, SelectSet:
			it.doQuery(payload)
		default:
			it.doExec(payload)
		}
		payload.session.Commit()
		it.applyHook(payload, SqlQuery, After)
	}
	it.processKeyProps(payload)
	it.processResults(payload)
}

func (it *Proxy) doQuery(payload *Payload) {
	rows, err := payload.session.Query(payload.Stmt, payload.Sql, payload.SqlArgs)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			panic(err)
		}
	}()
	var results []reflect.Value
	offset := 0
	for i, out := range it.outs {
		if refx.IsError(out) {
			offset++
			continue
		}
		rsMap := it.rsMaps[i-offset]
		if i > 0 {
			rows.NextResultSet()
		}
		newVal, err := scan.Read(rows, out, rsMap)
		if err != nil {
			panic(err)
		}
		results = append(results, newVal)
	}
	payload.Result = results
}

func (it *Proxy) doExec(payload *Payload) {
	result, err := payload.session.Exec(payload.Stmt, payload.Sql, payload.SqlArgs)
	if err != nil {
		panic(err)
	}
	if payload.Stmt.StmtType == Insert {
		lastId, err := result.LastInsertId()
		if err != nil {
			panic(err)
		}
		payload.LastInsertId = lastId
	}
	rows, err := result.RowsAffected()
	if err != nil {
		panic(err)
	}
	for i, out := range it.outs {
		if refx.IsGeneralInt(out) {
			newVal := refx.NewOf(out)
			refx.Assign(newVal.Addr(), rows)
			payload.Result[i] = newVal
		}
	}
}
