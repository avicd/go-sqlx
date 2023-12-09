package tests

import (
	"database/sql"
	"github.com/avicd/go-sqlx"
	"github.com/avicd/go-sqlx/session"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestMapper(t *testing.T) {
	db, _ := sql.Open("sqlite3", "test.db")
	config := &session.Config{XmlScan: "**/*.xml"}
	config.SetMainDB(db)
	ins := sqlx.New(config)

	// create a mapper use PkgPath as namespace
	var userDao UserDao
	ins.AsMapper(&userDao)

	user0 := &User{
		Name:     "test-name",
		RealName: "Allen",
		Age:      20,
		Comment:  "This is a test user",
	}

	// test insert
	userDao.Insert(user0)

	// test setting keyProp with last insertId
	assert.NotEqual(t, 0, user0.Id)

	// test select
	// automatic result mapping will be applied if missing
	// as: underline case => big camel case
	user1 := userDao.FindById(user0.Id)
	assert.Equal(t, true, reflect.DeepEqual(user0, user1))

	// test unnecessary update with if nodes
	rows := userDao.UpdateById(&User{Id: user1.Id, Name: "test-name-updated"})
	assert.Equal(t, 1, rows)

	// check update result
	user2 := userDao.FindById(user1.Id)
	assert.NotEqual(t, true, reflect.DeepEqual(user1, user2))
	assert.Equal(t, "test-name-updated", user2.Name)
	assert.Equal(t, "Allen", user2.RealName)

	// test delete
	rows = userDao.DeleteById(user2.Id)
	assert.Equal(t, 1, rows)

	user3 := userDao.FindById(user1.Id)
	assert.Equal(t, true, user3 == nil)

}

func TestProxy(t *testing.T) {
	db, _ := sql.Open("sqlite3", "test.db")
	config := &session.Config{XmlScan: "**/*.xml"}
	config.SetMainDB(db)
	ins := sqlx.New(config)

	// create a mapper use PkgPath as namespace
	var userDao UserDao
	ins.AsMapper(&userDao)

	user0 := &User{
		Name:     "test-name",
		RealName: "Allen",
		Age:      20,
		Comment:  "This is a test user",
	}
	userDao.Insert(user0)

	// create proxy function
	var getUser func(id string) *User
	ins.AsProxy(&getUser, "github.com.avicd.go-sqlx.tests.UserDao.FindById")

	user1 := getUser(user0.Id)
	// check result
	assert.Equal(t, true, reflect.DeepEqual(user0, user1))

	// create proxy function from raw script
	var rawGetUser func(id string) *User
	script :=
		`<select id='getUser' args='id'>
			select * from user where id = #{id}
		</select>`
	stmt := ins.StmtOf(script, "namespace-test")
	stmt.Scan(&rawGetUser)

	user2 := getUser(user0.Id)
	// check result
	assert.Equal(t, true, reflect.DeepEqual(user0, user2))
}
