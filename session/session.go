package session

import (
	"context"
	"database/sql"
	"github.com/avicd/go-sqlx/logger"
)

type Session struct {
	config  *Config
	txOn    bool
	txOpts  *sql.TxOptions
	ctx     context.Context
	keeper  *Keeper
	txs     []*sql.Tx
	cons    []*sql.Conn
	stmts   []*sql.Stmt
	proxyDB map[*sql.DB]ProxyDB
	closed  bool
}

type ProxyDB interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func (it *Session) getProxyDB(stmt *Stmt) (ProxyDB, error) {
	if it.proxyDB == nil {
		it.proxyDB = map[*sql.DB]ProxyDB{}
	}
	if db, ok := it.proxyDB[stmt.DB()]; ok {
		return db, nil
	}
	conn, err := stmt.DB().Conn(it.ctx)
	if err != nil {
		return nil, err
	}
	it.cons = append(it.cons, conn)
	var proxy ProxyDB
	if it.txOn {
		tx, err1 := conn.BeginTx(it.ctx, it.txOpts)
		if err1 != nil {
			return nil, err1
		}
		it.proxyDB[stmt.DB()] = tx
		it.txs = append(it.txs, tx)
		proxy = tx
	} else {
		it.proxyDB[stmt.DB()] = conn
		proxy = conn
	}
	return proxy, nil
}

func (it *Session) prepare(stmt *Stmt, sql string) (*sql.Stmt, error) {
	proxy, err := it.getProxyDB(stmt)
	if err != nil {
		return nil, err
	}
	return proxy.PrepareContext(it.ctx, sql)
}

func (it *Session) Query(stmt *Stmt, sql string, values []any) (*sql.Rows, error) {
	prepared, err := it.prepare(stmt, sql)
	if err != nil {
		return nil, err
	}
	return prepared.QueryContext(context.Background(), values...)
}

func (it *Session) Exec(stmt *Stmt, sql string, values []any) (sql.Result, error) {
	prepared, err := it.prepare(stmt, sql)
	if err != nil {
		return nil, err
	}
	return prepared.ExecContext(context.Background(), values...)
}

func (it *Session) Commit() {
	if it.closed {
		return
	}
	it.keeper.Pop()
	if !it.keeper.Locked() {
		it.keeper.Commit()
	}
}

func (it *Session) Rollback() {
	if it.closed {
		return
	}
	it.keeper.Rollback()
}

func (it *Session) close(rollback bool) {
	if it.closed {
		return
	}
	it.closed = true
	if it.txOn {
		for _, tx := range it.txs {
			if rollback {
				err := tx.Rollback()
				if err != nil {
					logger.Error(err.Error())
				}
			} else {
				err := tx.Commit()
				if err != nil {
					logger.Error(err.Error())
				}
			}
		}
	}
	for _, stmt := range it.stmts {
		err := stmt.Close()
		if err != nil {
			logger.Error(err.Error())
		}
	}
	for _, conn := range it.cons {
		err := conn.Close()
		if err != nil {
			logger.Error(err.Error())
		}
	}
	it.cons = nil
	it.txs = nil
	it.stmts = nil
}
