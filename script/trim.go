package script

import "strings"

type trimScope struct {
	Context
	*TrimNode
	prefixApplied bool
	suffixApplied bool
}

type TrimNode struct {
	Content            SqlNode
	Prefix             string
	Suffix             string
	PrefixesToOverride []string
	SuffixesToOverride []string
}

func (ctx *trimScope) Append(sql string) {
	trimmed := strings.TrimSpace(sql)
	if len(trimmed) > 0 {
		if !ctx.prefixApplied {
			ctx.prefixApplied = true
			for _, cut := range ctx.PrefixesToOverride {
				if cut != "" {
					trimmed = strings.TrimPrefix(trimmed, cut)
				}
			}
			if ctx.Prefix != "" {
				trimmed = ctx.Prefix + " " + trimmed
			}
		}
		if !ctx.suffixApplied {
			ctx.suffixApplied = true
			for _, cut := range ctx.SuffixesToOverride {
				if cut != "" {
					trimmed = strings.TrimSuffix(trimmed, cut)
				}
			}
			if ctx.Suffix != "" {
				trimmed += " " + ctx.Suffix
			}
		}
	}
	ctx.Context.Append(trimmed)
}

func (it *TrimNode) Render(ctx Context) bool {
	scope := &trimScope{
		Context:       ctx,
		TrimNode:      it,
		prefixApplied: false,
		suffixApplied: false,
	}
	return it.Content.Render(scope)
}
