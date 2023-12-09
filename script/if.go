package script

import (
	"github.com/avicd/go-utilx/refx"
)

type IfNode struct {
	Content SqlNode
	Test    string
}

func (it *IfNode) Render(ctx Context) bool {
	val := ctx.Eval(it.Test)
	if refx.AsBool(val) {
		it.Content.Render(ctx)
		return true
	}
	return false
}
