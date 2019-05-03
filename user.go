package ftpserver

type UserDriver interface {
	/* Check the user name and passwd is valid.
	If the user name and passwd is valid,and
	need to save the user name and passwd */
	CheckUser(string) bool
	CheckPass(string) bool

	GetUserName() string
	GetUid() int
	CheckAuth(uint) bool
}

type UserRequire interface {
	Response(string) error
	SetRootEntry(int) error
}

const invalidUid = -1

const (
	GET = iota
	PUT
	DELETE
	RECOVER
	MKDIR
	DELDIR
)

type User struct {
	name     string
	pass     string
	uid      int
	authFlag uint
}

func NewUser() *User {
	return &User{
		uid: invalidUid,
	}
}

func (user *User) CheckUser(name string) bool {
	var setFlag = func(permit bool, flag uint) {
		if permit {
			user.authFlag |= uint(1) << flag
		}
	}

	for index, value := range Conf.Users {

		if value.Name != name {
			continue
		}

		user.name = name
		user.uid = index
		user.pass = value.Pass

		setFlag(value.Get, GET)
		setFlag(value.Put, PUT)
		setFlag(value.Delete, DELETE)
		setFlag(value.Recover, RECOVER)
		setFlag(value.MkDir, MKDIR)
		setFlag(value.DelDir, DELDIR)

		return true
	}
	return false
}

func (user *User) CheckPass(pass string) bool {
	return user.pass == pass
}

func (user *User) GetUserName() string {
	return user.name
}

func (user *User) GetUid() int {
	return user.uid
}

func (user *User) CheckAuth(auth uint) bool {
	if user.uid == invalidUid {
		return false
	}
	auth = uint(1) << auth
	if auth&user.authFlag != 0 {
		return true
	}
	return false
}

func commandUser(info []byte, user UserDriver, require UserRequire) error {
	/* Whether the check is successful or not, all return to success. */
	user.CheckUser(string(info))
	if user.GetUid() != invalidUid {
		if err := require.SetRootEntry(user.GetUid()); err != nil {
			Fataln(err)
		}
	}
	return require.Response("331 Login OK, send your password\r\n")
}

func commandPass(info []byte, user UserDriver, require UserRequire) error {

	if user.GetUid() != invalidUid && user.CheckPass(string(info)) {
		return require.Response("230 Login OK\r\n")
	} else {
		return require.Response("530 Permission denied\r\n")
	}
}

func AuthProc(command string, info []byte, ftp *Ftp) error {
	if command == "USER" {
		return commandUser(info, ftp, ftp)
	} else if command == "PASS" {
		return commandPass(info, ftp, ftp)
	}
	Fataln(command)
	return nil
}

func init() {
	register("USER", AuthProc)
	register("PASS", AuthProc)
}
