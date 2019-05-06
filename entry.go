package ftpserver

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var (
	errGetPathStat  = errors.New("Error: Can't get file stat.")
	errPathNonExist = errors.New("Error: The Path is not exist.")
	errNonDirPath   = errors.New("Error: The path is not a dictionary.")
	errPathIsEmpty  = errors.New("Error: Input Path is empty.")
	errHasBeenRoot  = errors.New("Error: The current path has been in root.")
	errReadDirs     = errors.New("Error: Read the Dirs has been Error.")
)

type EntryDriver interface {
	/* Operations related to file directories.
	Enter the folder.Get the current file path.
	get the current file list information. */
	SetRootEntry(uid int) error
	EnterEntry(folder string) error
	GetPwd() string
	Getlist(folder string) ([]byte, error)
	//DeleteFile(path string) error

	GetRootDir() string
	GetCurDir() string
}

type EntryRequire interface {
	Response(string) error
	WaitDataConn()
	WriteAll([]byte) error
	DataClose()
	CheckAuth(uint) bool
}

type Entry struct {
	rootPath string
	curPath  string
}

func isValidDir(folder string) error {
	f, err := os.Stat(folder)
	if err != nil {
		if os.IsNotExist(err) {
			return errPathNonExist
		}
		log.Println(err)
		return errGetPathStat
	}
	if !f.IsDir() {
		return errNonDirPath
	}
	return nil
}

func (entry *Entry) SetRootEntry(uid int) error {
	var folder = Conf.Users[uid].Root

	if err := isValidDir(folder); err != nil {
		return err
	}

	if folder[len(folder)-1] == '/' {
		entry.rootPath = folder[:len(folder)-1]
	} else {
		entry.rootPath = folder
	}

	entry.curPath = entry.rootPath + "/"
	return nil
}

func (entry *Entry) EnterEntry(folder string) error {

	/* return the root dir */
	if folder == "/" {
		entry.curPath = entry.rootPath + "/"
		return nil
	}

	//Debugln("EnterEntry folder.", folder)

	/* return the parent dir */
	if folder == ".." {
		var relaPath = entry.curPath[len(entry.rootPath):]
		if relaPath == "/" {
			return errHasBeenRoot
		}

		var index = strings.LastIndex(
			relaPath[:len(relaPath)-1], "/")
		entry.curPath = entry.rootPath + relaPath[:index+1]
		return nil
	}

	if len(folder) == 0 {
		return errPathIsEmpty
	}

	var path string
	if folder[0] == '/' {
		path = entry.rootPath + folder
	} else {
		path = entry.curPath + folder
	}

	if err := isValidDir(path); err != nil {
		return err
	}
	entry.curPath = path + "/"
	return nil
}

func (entry *Entry) GetPwd() string {
	return entry.curPath[len(entry.rootPath):]
}

func (entry *Entry) Getlist(folder string) ([]byte, error) {
	if folder == "" {
		folder = entry.curPath
	}

	dirList, err := ioutil.ReadDir(folder)
	if err != nil {
		log.Println(err)
		return nil, errReadDirs
	}

	const time_layet = "Jan 2 15:04"

	var msg = ""
	for _, f := range dirList {
		msg += fmt.Sprintf("%s %5d %4d %4d %8d %s %s\r\n",
			f.Mode(), 1, 0, 0, f.Size(),
			time.Unix(f.ModTime().Unix(), 0).Format(time_layet),
			f.Name())
	}

	return []byte(msg), nil
}

func (entry *Entry) GetRootDir() string {
	return entry.rootPath
}

func (entry *Entry) GetCurDir() string {
	return entry.curPath
}

func commandCwd(info []byte, driver EntryDriver, require EntryRequire) error {
	if err := driver.EnterEntry(string(info)); err != nil {
		if err == errPathIsEmpty || err == errPathNonExist {
			return require.Response(
				"501 Parameter syntax error.Please Input correct folder path.")
		} else if err == errNonDirPath {
			var resMsg = fmt.Sprintf(
					"501 Parameter syntax error.%s is not a dictionary\r\n",
					string(info))
			return require.Response(resMsg)
		} else if err == errGetPathStat {
			return require.Response("451 Has unknown local Error\r\n")
		} else {
			Warnln(err)
		}
	}
	return require.Response("250 Requested File Operation Completed\r\n")
}

func commandCdup(info []byte, driver EntryDriver, require EntryRequire) error {
	if len(info) != 0 {
		var resMsg = fmt.Sprintf(
			"501 Parameter syntax error.Can't idenfy \"%s\"", string(info))
		return require.Response(resMsg)
	}

	if err := driver.EnterEntry(".."); err != nil {
		if err == errHasBeenRoot {
			return require.Response(
				"550 The operation that did not execute,Has been root dir\r\n")
		} else {
			Warnln(err)
		}
	}
	return require.Response(
		"200 Command succeed,Return Parent folder\r\n")
}

func commandList(info []byte, driver EntryDriver, require EntryRequire) error {
	var list, err = driver.Getlist(string(info))
	if err != nil {
		if err == errReadDirs {
			return require.Response(
				"451 Has unknown local Error.Get dictionary Error\r\n")
		} else {
			Warnln(err)
		}
	}

	require.WaitDataConn()

	if err := require.WriteAll(list); err != nil {
		return require.Response(
			"550 The operation that did not execute, data connection error\r\n")
	}

	require.DataClose()

	return require.Response(
		"226 Close the data connection, the requested file operation is successful\r\n")
}

func commandPwd(info []byte, driver EntryDriver, require EntryRequire) error {
	return require.Response(
		fmt.Sprintf("257 %s\r\n", driver.GetPwd()))
}

func commandMkr(info []byte, dirver EntryDriver, require EntryRequire) error {
	if len(info) == 0 {
		return require.Response("501 Parameter syntax error." +
			"Please input dictionary name\r\n")
	} else if len(info) > 64 {
		return require.Response("550 The operation that did not execute." +
			"Dictionary name too long\r\n")
	}

	if !require.CheckAuth(MKDIR) {
		return require.Response("530 Parameter denied\r\n")
	}

	if strings.Contains(string(info), "/") {
		/* for security. Preventive use "../" Return to the upper directory.*/
		return require.Response("550 The operation that did not execute." +
			"Only support relative path\r\n")
	}

	var dirName = dirver.GetCurDir() + string(info)
	if err := isValidDir(dirName); err == nil || err == errNonDirPath{
		return require.Response("550 The operation that did not execute." +
			"The dictionary has been exist\r\n")
	} else if err == errGetPathStat {
		return require.Response("451 Abort the operation of the request\r\n")
	}

	if err := os.Mkdir(dirName, os.ModePerm); err != nil {
		Warnln(err)
		return require.Response("451 Abort the operation of the request\r\n")
	} else {
		Debugln("Create Dictionary " + dirName)
		return require.Response("257 Create dictionary succeed\r\n")
	}
}

func DirProc(command string, info []byte, ftp *Ftp) error {

	if command == "CDUP" {
		return commandCdup(info, ftp, ftp)
	} else if command == "CWD" {
		return commandCwd(info, ftp, ftp)
	} else if command == "LIST" {
		return commandList(info, ftp, ftp)
	} else if command == "PWD" {
		return commandPwd(info, ftp, ftp)
	} else if command == "MKD" {
		return commandMkr(info, ftp, ftp)
	}
	Fataln()
	return nil
}

func init() {
	register("CDUP", DirProc)
	register("CWD", DirProc)
	register("LIST", DirProc)
	register("PWD", DirProc)
	register("MKD", DirProc)
}
