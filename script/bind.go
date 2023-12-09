package script

type BindNode struct {
	Name  string
	Value string
}

func (it *BindNode) Render(ctx Context) bool {
	result := ctx.Eval(it.Value)
	ctx.Bind(it.Name, result)
	return true
}
