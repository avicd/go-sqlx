package script

type StaticNode struct {
	Text string
}

func (it *StaticNode) Render(ctx Context) bool {
	ctx.Append(it.Text)
	return true
}
