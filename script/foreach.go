package script

import (
	"fmt"
	"github.com/avicd/go-utilx/refx"
	"github.com/avicd/go-utilx/tokx"
)

const foreachPrefix = "_foreach_var_"

type ForeachNode struct {
	Collection string
	Content    SqlNode
	Open       string
	Close      string
	Separator  string
	Item       string
	Index      string
}

type foreachScope struct {
	Context
	*ForeachNode
	itemValue  interface{}
	indexValue interface{}
	isFirst    bool
}

func (ctx *foreachScope) Append(sql string) {
	if ctx.Item != "" {
		ctx.Backup(ctx.Item)
	}
	if ctx.Index != "" {
		ctx.Backup(ctx.Index)
	}
	newScope := tokx.NewPair("#{", "}").Map(sql, func(expr string) string {
		if ctx.Item != "" {
			ctx.Bind(ctx.Item, ctx.itemValue)
		}
		if ctx.Index != "" {
			ctx.Bind(ctx.Index, ctx.indexValue)
		}
		val := ctx.Eval(expr)
		varName := fmt.Sprintf("%s%d", foreachPrefix, ctx.GetUid())
		ctx.Bind(varName, val)
		if ctx.Item != "" {
			ctx.UnBind(ctx.Item)
		}
		if ctx.Index != "" {
			ctx.UnBind(ctx.Index)
		}
		return fmt.Sprintf("#{%s}", varName)
	})
	if ctx.Item != "" {
		ctx.Restore(ctx.Item)
	}
	if ctx.Index != "" {
		ctx.Restore(ctx.Index)
	}
	if ctx.isFirst {
		ctx.isFirst = false
	} else {
		ctx.Context.Append(ctx.Separator)
	}
	ctx.Context.Append(newScope)
}

func (it *ForeachNode) Render(ctx Context) bool {
	result := ctx.Eval(it.Collection)

	if it.Open != "" {
		ctx.Append(it.Open)
	}

	scope := &foreachScope{
		Context:     ctx,
		ForeachNode: it,
		isFirst:     true,
	}

	itrFn := func(key interface{}, val interface{}) {
		scope.itemValue = val
		scope.indexValue = key
		it.Content.Render(scope)
	}

	if refx.IsNumber(result) {
		for i := 0; i < int(refx.AsInt(result)); i++ {
			itrFn(i, i)
		}
	} else if !refx.IsBasic(result) {
		refx.ForEach(result, itrFn)
	}

	if it.Close != "" {
		ctx.Append(it.Close)
	}

	return true
}
