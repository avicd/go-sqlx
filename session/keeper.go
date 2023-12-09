package session

import (
	"context"
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-utilx/datax"
	"unsafe"
)

type Keeper struct {
	factory *Factory
	reused  int
	chain   datax.LinkedList[*Session]
	done    datax.LinkedList[*Session]
	Context context.Context
}

func (it *Keeper) Current() *Session {
	val, _ := it.chain.Last()
	return val
}

func (it *Keeper) Reuse() {
	it.reused++
	session := it.Current()
	if session.txOn {
		logger.Debugf("transactional session %v reused", unsafe.Pointer(session))
	} else {
		logger.Debugf("normal session %v reused", unsafe.Pointer(session))
	}
}

func (it *Keeper) Push(session *Session) {
	it.chain.Push(session)
}

func (it *Keeper) Pop() {
	if it.reused > 0 {
		it.reused--
		return
	}
	if val, ok := it.chain.Pop(); ok {
		it.done.Push(val)
	}
}

func (it *Keeper) Locked() bool {
	return it.chain.Len() > 0
}

func (it *Keeper) reset() {
	it.factory.ResetKeeper()
}

func (it *Keeper) Commit() {
	it.done.ForEach(func(i int, item *Session) {
		if item.txOn {
			logger.Debugf("transactional session %v closed", unsafe.Pointer(item))
		} else {
			logger.Debugf("normal session %v closed", unsafe.Pointer(item))
		}
		item.close(false)
	})
	it.done.Clear()
	it.reset()
}

func (it *Keeper) Rollback() {
	closeFunc := func(i int, item *Session) {
		if item.txOn {
			logger.Debugf("transactional session %v rolled back", unsafe.Pointer(item))
		} else {
			logger.Debugf("normal session %v closed by forcing", unsafe.Pointer(item))
		}
		item.close(false)
	}
	it.done.ForEach(closeFunc)
	it.done.Clear()
	it.chain.ForEach(closeFunc)
	it.chain.Clear()
	it.reset()
}
