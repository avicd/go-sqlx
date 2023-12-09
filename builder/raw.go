package builder

import (
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-sqlx/session"
	"github.com/avicd/go-utilx/refx"
	"github.com/avicd/go-utilx/xmlx"
	"strings"
)

type RawBuilder struct {
	Ns     string
	Script string
	Stmt   *session.Stmt
}

func (it *RawBuilder) Build(config *session.Config) {
	root, err := xmlx.Parse(strings.NewReader(it.Script))
	if err != nil {
		logger.Fatal(err.Error())
	}
	stNode := root.FindOne("select|insert|update|delete")
	if stNode != nil {
		stBuilder := &StmtBuilder{Ns: it.Ns, Nodes: refx.AsListOf[*xmlx.Node](stNode)}
		stBuilder.Build(config)
		for _, stmt := range stBuilder.Stmts {
			it.Stmt = stmt
			break
		}
	} else {
		logger.Error("missing statement in script text")
	}
}
