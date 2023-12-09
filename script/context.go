package script

type Context interface {
	Eval(text string) any
	GetUid() int64
	Bind(name string, value interface{})
	UnBind(name string)
	Backup(name string)
	Restore(name string)
	Append(sql string)
	Reset()
}

type SqlNode interface {
	Render(ctx Context) bool
}
