package session

type Hook = int

const (
	ProcessArgs Hook = iota
	SqlQuery
	ProcessResults
	Always
)

type Order = int

const (
	Before Order = iota
	After
)

type Plugin interface {
	Intercept(payload *Payload) bool
	Hook() Hook
	Order() Order
	Id() int
}
