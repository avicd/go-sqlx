package session

import (
	"database/sql"
	"github.com/avicd/go-sqlx/logger"
	"unsafe"
)

const NameSpace = "github.com/avicd/go-sqlx"

type RsMap = map[string][]string

type Config struct {
	XmlScan string
	factory *Factory
	mainDB  *sql.DB
	dbMap   map[string]*sql.DB
	nss     map[string]string
	stmts   map[string]*Stmt
	rsMaps  map[string]RsMap
	plugins []Plugin
}

func (it *Config) MainDB() *sql.DB {
	return it.mainDB
}

func (it *Config) SetMainDB(db *sql.DB) {
	it.mainDB = db
	logger.Debugf("set main database %v", unsafe.Pointer(db))
}

func (it *Config) SetDB(id string, db *sql.DB) *Config {
	if it.dbMap == nil {
		it.dbMap = map[string]*sql.DB{}
	}
	if it.mainDB == nil {
		it.SetMainDB(db)
	}
	it.dbMap[id] = db
	logger.Debugf("set database id='%s' %v", id, unsafe.Pointer(db))
	return it
}

func (it *Config) GetDB(id string) *sql.DB {
	if db, ok := it.dbMap[id]; ok {
		return db
	}
	return nil
}

func (it *Config) AddNs(ns string, xml string) {
	if it.nss == nil {
		it.nss = map[string]string{}
	}
	it.nss[ns] = xml
	logger.Debugf("add mapper '%s', file=%s", ns, xml)
}

func (it *Config) GetNsXml(ns string) string {
	if str, ok := it.nss[ns]; ok {
		return str
	}
	return ""
}

func (it *Config) HasNs(ns string) bool {
	if _, exist := it.nss[ns]; exist {
		return true
	}
	return false
}

func (it *Config) AddRsMap(id string, rsMap RsMap) {
	if it.rsMaps == nil {
		it.rsMaps = map[string]RsMap{}
	}
	it.rsMaps[id] = rsMap
	logger.Debugf("add resultMap '%s'", id)
}

func (it *Config) GetRsMap(id string) RsMap {
	if val, ok := it.rsMaps[id]; ok {
		return val
	}
	return nil
}

func (it *Config) AddStmt(stmt *Stmt) {
	if it.stmts == nil {
		it.stmts = map[string]*Stmt{}
	}
	stmt.config = it
	it.stmts[stmt.Id] = stmt
	logger.Debugf("add statement '%s'", stmt.Id)
}

func (it *Config) GetStmt(id string) *Stmt {
	id = StmtIdOf(id)
	if stmt, ok := it.stmts[id]; ok {
		return stmt
	}
	return nil
}

func (it *Config) AddPlugin(plugin Plugin) {
	if plugin == nil {
		return
	}
	if len(it.plugins) > 0 {
		order := plugin.Id()
		var dest []Plugin
		list := it.plugins
		index := len(list)
		for index > 0 && list[index-1].Id() > order {
			index--
		}
		dest = append(dest, list[:index]...)
		dest = append(dest, plugin)
		dest = append(dest, list[index:]...)
		it.plugins = dest
	} else {
		it.plugins = append(it.plugins, plugin)
	}
}

func (it *Config) Factory() *Factory {
	if it.factory == nil {
		it.factory = &Factory{config: it, local: map[int64]*Keeper{}}
	}
	return it.factory
}
