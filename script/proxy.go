package script

type ProxyNode struct {
	SqlNodes []SqlNode
}

func (it *ProxyNode) Render(ctx Context) bool {
	for _, p := range it.SqlNodes {
		p.Render(ctx)
	}
	return true
}
