package sqlx

import (
	"database/sql"
	"github.com/avicd/go-sqlx/builder"
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-sqlx/session"
	"github.com/avicd/go-utilx/refx"
	"reflect"
	"strings"
)

type Sqlx struct {
	Config *session.Config
}

func New(config *session.Config, builders ...builder.Builder) *Sqlx {
	ins := &Sqlx{Config: config}
	if len(builders) > 0 {
		for _, bdl := range builders {
			bdl.Build(config)
		}
	} else {
		bdl := &builder.XmlBuilder{}
		bdl.Build(config)
	}
	return ins
}

func (it *Sqlx) AsMapper(dest any, nss ...string) {
	if !refx.IsStruct(dest) {
		logger.Fatal("dest is not a struct")
	}
	var el reflect.Value
	if target, ok := refx.AddrAbleOf(dest); ok {
		target.Set(refx.NewOf(target))
		el = refx.Indirect(target)
	} else {
		logger.Fatal("dest is unaddressable")
	}
	var ns string
	if len(nss) > 0 {
		ns = nss[0]
		logger.Warnf("namespace provided is empty, fallback to PkgPath")
	}
	if ns == "" {
		ns = strings.ReplaceAll(el.Type().PkgPath(), "/", ".") + "." + el.Type().Name()
	}
	if !it.Config.HasNs(ns) {
		logger.Fatalf("missing mapper namespace='%s'", ns)
	}
	for i := 0; i < el.Type().NumField(); i++ {
		tf := el.Type().Field(i)
		if !tf.IsExported() || tf.Type.Kind() != reflect.Func {
			continue
		}
		id := ns + "." + tf.Name
		stmt := it.Config.GetStmt(id)
		if stmt == nil {
			logger.Warnf("missing statement id='%s', ignored", id)
			continue
		}
		el.Field(i).Set(stmt.ProxyOfField(tf).Invoker())
	}
}

func (it *Sqlx) AsProxy(dest any, id string) {
	stmt := it.Config.GetStmt(id)
	if stmt == nil {
		logger.Fatalf("missing statement id='%s'", id)
	}
	stmt.Scan(dest)
}

func (it *Sqlx) Tx(fn func()) {
	it.TxWith(nil, fn)
}

func (it *Sqlx) TxWith(txOpts *sql.TxOptions, fn func()) {
	sess := it.Config.Factory().OpenTxWith(txOpts)
	defer func() {
		err := recover()
		if err != nil {
			sess.Rollback()
			logger.Error(err)
		} else {
			sess.Commit()
		}
	}()
	fn()
}

func (it *Sqlx) StmtOf(script string, nss ...string) *session.Stmt {
	stNs := session.NameSpace
	if len(nss) > 0 {
		stNs = nss[0]
	}
	bdl := &builder.RawBuilder{Script: script, Ns: stNs}
	bdl.Build(it.Config)
	return bdl.Stmt
}
