# go-sqlx
A effective ORM framework built for golang

## installation
```bash
go get github.com/avicd/go-sqlx
```

## usage
Please refer to [test cases](tests/user_test.go) with more details

```golang
type User struct {
	Id       string
	Name     string
	RealName string
	Age      int
	Comment  string
}

type UserDao struct {
	FindById   func(id string) *User
	Insert     func(user *User) int
	UpdateById func(user *User) int
	DeleteById func(id string) int
}

func test()  {
	db, _ := sql.Open("sqlite3", "test.db")
	config := &session.Config{XmlScan: "**/*.xml"}
	config.SetMainDB(db)
	ins := sqlx.New(config)
	
	// create a mapper not using a namespace
	// it will use a struct's PkgPath as namespace
	var userDao UserDao
	ins.AsMapper(&userDao)
	
	// create a mapper using a namespace
	ins.AsMapper(&userDao, "github.com.avicd.go-sqlx.tests.UserDao")
	
	// create a proxy function from an existed stmt
	var getUser func(id string) *User
	ins.AsProxy(&getUser, "github.com.avicd.go-sqlx.tests.UserDao.FindById")

	// create a proxy function from raw script 
	var rawGetUser func(id string) *User
	script :=
		`<select id='getUser' args='id'>
			select * from user where id = #{id}
		</select>`
	stmt := ins.StmtOf(script, "namespace-test")
	stmt.Scan(&rawGetUser)
}
```
