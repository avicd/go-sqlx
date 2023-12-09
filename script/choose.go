package script

type ChooseNode struct {
	Default SqlNode
	IfNodes []SqlNode
}

func (it *ChooseNode) Render(ctx Context) bool {
	for _, p := range it.IfNodes {
		if p.Render(ctx) {
			return true
		}
	}
	if it.Default != nil {
		it.Default.Render(ctx)
		return true
	}
	return false
}
