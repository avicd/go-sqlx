package builder

import "github.com/avicd/go-sqlx/session"

type Builder interface {
	Build(config *session.Config)
}
