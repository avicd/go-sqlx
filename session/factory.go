package session

import (
	"context"
	"database/sql"
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-utilx/goid"
	"reflect"
	"unsafe"
)

type Factory struct {
	config *Config
	local  map[int64]*Keeper
}

func (it *Factory) Keeper() *Keeper {
	sid := goid.Id()
	if keeper, ok := it.local[sid]; ok {
		return keeper
	}
	ret := &Keeper{
		factory: it,
		Context: context.Background(),
	}
	it.local[sid] = ret
	return ret
}

func (it *Factory) ResetKeeper() {
	sid := goid.Id()
	delete(it.local, sid)
}

func (it *Factory) Open() *Session {
	keeper := it.Keeper()
	if !keeper.Locked() {
		session := &Session{
			config: it.config,
			keeper: keeper,
			ctx:    keeper.Context,
		}
		keeper.Push(session)
		logger.Debugf("normal session %v opened", unsafe.Pointer(session))
		return session
	} else {
		keeper.Reuse()
		return keeper.Current()
	}
}

func (it *Factory) OpenTx() *Session {
	return it.OpenTxWith(nil)
}

func (it *Factory) OpenTxWith(txOpts *sql.TxOptions) *Session {
	keeper := it.Keeper()
	if keeper.Locked() {
		session := keeper.Current()
		if reflect.DeepEqual(session.txOpts, txOpts) {
			keeper.Reuse()
			return session
		}
	}
	session := &Session{
		config: it.config,
		keeper: keeper,
		ctx:    keeper.Context,
		txOn:   true,
		txOpts: txOpts,
	}
	keeper.Push(session)
	logger.Debugf("transactional session %v opened", unsafe.Pointer(session))
	return session
}
