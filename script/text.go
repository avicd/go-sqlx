package script

import (
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-utilx/refx"
	"github.com/avicd/go-utilx/tokx"
	"regexp"
)

const InjectPattern = `(?:')|(?:--)|(/\\*(?:.|[\\n\\r])*?\\*/)|(\b(select|update|and|or|delete|insert|trancate|char|chr|into|substr|ascii|declare|exec|count|master|into|drop|execute)\b)`

type TextNode struct {
	Text string
}

func (it *TextNode) IsDynamic() bool {
	return tokx.NewPair("${", "}").Match(it.Text)
}

func (it *TextNode) Render(ctx Context) bool {
	sqlStr := tokx.NewPair("${", "}").Map(it.Text, func(expr string) string {
		result := refx.AsString(ctx.Eval(expr))
		if result != "" && regexp.MustCompile(InjectPattern).MatchString(result) {
			logger.Fatalf("Invalid input. Please conform to regex \"%s\"", InjectPattern)
		}
		return result
	})
	ctx.Append(sqlStr)
	return true
}
