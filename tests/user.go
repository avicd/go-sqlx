package tests

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
