package script

import (
	"github.com/avicd/go-utilx/evalx"
	"strings"
)

type SqlBuilder struct {
	*evalx.Scope
	uid int64
	buf strings.Builder
}

func NewSqlBuilder() *SqlBuilder {
	return &SqlBuilder{
		Scope: evalx.NewScope(),
	}
}

func (it *SqlBuilder) Build(sqlNode SqlNode) string {
	it.Reset()
	sqlNode.Render(it)
	return strings.TrimSpace(it.buf.String())
}

func (it *SqlBuilder) Eval(text string) any {
	val, _ := it.Scope.Eval(text)
	return val
}

func (it *SqlBuilder) GetUid() int64 {
	it.uid++
	return it.uid
}

func (it *SqlBuilder) Append(sql string) {
	it.buf.WriteString(" " + sql)
}

func (it *SqlBuilder) Reset() {
	it.buf.Reset()
}
