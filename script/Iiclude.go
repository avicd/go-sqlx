package script

type IncludeNode struct {
	Content SqlNode
}

func (it *IncludeNode) Render(ctx Context) bool {
	if it.Content != nil {
		it.Content.Render(ctx)
		return true
	}
	return false
}
