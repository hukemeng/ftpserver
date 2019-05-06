package ftpserver

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	errFileNameEmpty   = errors.New("File name is empty.")
	errFileNonExist    = errors.New("FIle is not exist.")
	errFileUnkSystem   = errors.New("Unknown system error.")
	errFileSameNameDir = errors.New("Has same name dictionary.")
	errFileTransfer    = errors.New("File transfer unknown error.")
	errFileReciver     = errors.New("File receive unknown error.")
	errFileCreate      = errors.New("File create error.")
)

type FileDriver interface {
	FileIsExist(path string) error
	GetFileSize(string) (int64, error)
	Sendfile(string, io.Writer) error
	Recvfile(string, io.Reader) error
}

type FileRequire interface {
	Response(string) error
	GetCurDir() string
	GetRootDir() string
	GetUserName() string
	CheckAuth(uint) bool

	WaitDataConn()
	Write(msg []byte) (int, error)
	Read(msg []byte) (int, error)
	DataClose()
}

type File struct {
}

func (file *File) FileIsExist(path string) error {
	f, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errFileNonExist
		}
		Warnln("Unknown system error", err)
		return errFileUnkSystem
	}
	if f.IsDir() {
		return errFileSameNameDir
	}
	return nil
}

func (file *File) GetFileSize(path string) (int64, error) {
	if path == "" {
		return 0, errFileNameEmpty
	}

	if err := file.FileIsExist(path); err != nil {
		return 0, err
	}

	var stat, err = os.Stat(path)
	if err != nil {
		Warnln("Unknown system error", err)
		return 0, errFileUnkSystem
	}

	return stat.Size(), nil
}

func (file *File) Sendfile(path string, writer io.Writer) error {

	if err := file.FileIsExist(path); err != nil {
		return err
	}

	reader, err := os.OpenFile(path, os.O_RDONLY, 0440)
	if err != nil {
		Warnln(err)
		return errFileUnkSystem
	}

	_, err = io.Copy(writer, reader)
	if err != nil {
		Warnln(err)
		return errFileTransfer
	}

	return nil
}

func (file *File) Recvfile(path string, reader io.Reader) error {
	var writer, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		Warnln(err)
		return errFileCreate
	}

	_, err = io.Copy(writer, reader)
	if err != nil {
		Warnln(err)
		return errFileReciver
	}
	return nil
}

func commandRetr(info []byte, driver FileDriver, require FileRequire) error {
	if len(info) == 0 {
		return require.Response(
			"501 Parameter syntax error.Please input file name\r\n")
	}

	if !require.CheckAuth(GET) {
		Debugln(require.GetUserName() + " Has No Permisson To Get File.")
		return require.Response("530 Permission denied\r\n")
	}

	var path string
	if info[0] == '/' {
		path = require.GetRootDir() + string(info)
	} else {
		path = require.GetCurDir() + string(info)
	}

	var size, err = driver.GetFileSize(string(path))
	if err == errFileNonExist {
		return require.Response(fmt.Sprintf(
			"501 Parameter syntax error.Please input correctly file name\r\n"))
	} else if err == errFileSameNameDir {
		return require.Response(fmt.Sprintf(
			"501 Parameter syntax error.This is a dictionary\r\n"))
	} else if err == errFileUnkSystem {
		return require.Response(fmt.Sprintf(
			"451 Abort the operation of the request,there are local errors\r\n"))
	}

	var msg = fmt.Sprintf("150 opeing %s mode data"+
		"connection for %s (%dbytes)\r\n",
		"Binary", string(info), size)

	if err := require.Response(msg); err != nil {
		return err
	}

	require.WaitDataConn()
	if err := driver.Sendfile(path, require); err != nil {
		return require.Response("451 Abort the operation." + err.Error() + "\r\n")
	}
	require.DataClose()

	return require.Response(
		"226 Close the data connection, the requested file operation is successful\r\n")
}

func commandStor(info []byte, driver FileDriver, require FileRequire) error {
	if len(info) == 0 {
		return require.Response(fmt.Sprintf(
			"501 Parameter syntax error.Please input file name\r\n"))
	}

	if !require.CheckAuth(PUT) {
		Debugln(require.GetUserName() + " Has No Permisson To Put File.")
		return require.Response("530 Permission denied\r\n")
	}

	var path string
	if info[0] == '/' {
		path = require.GetRootDir() + string(info)
	} else {
		path = require.GetCurDir() + string(info)
	}

	err := driver.FileIsExist(path)
	/* the file has been exist */
	if err == nil {
		if !require.CheckAuth(RECOVER) {
			Debugln(require.GetUserName() + " Has No Permisson To Recover File.")
			return require.Response("530 Permission deny.The same file already exists\r\n")
		}

		/* delete the file*/
		if err := os.Remove(path); err != nil {
			Debugln("Recover File " + path + " from " + require.GetUserName())
			return require.Response(
				"451 Abort the operation of the request,there are local errors\r\n")
		}
	} else if err == errFileUnkSystem {
		return require.Response(
			"451 Abort the operation of the request,there are local errors\r\n")
	} else if err == errFileSameNameDir {
		return require.Response("550 The operation that did not execute." +
			"The same dictionary already exists\r\n")
	}

	var msg = fmt.Sprintf("150 opeing %s mode data"+
		"connection for %s \r\n", "Binary", string(info))
	if err := require.Response(msg); err != nil {
		return err
	}

	require.WaitDataConn()
	if err := driver.Recvfile(path, require); err != nil {
		Ignore(os.Remove(path))
		Warnln("Write file Failed", err)
		return require.Response("451 Abort the operation." + err.Error())
	}
	require.DataClose()

	Debugln("Receive File " + path + " from " + require.GetUserName())
	return require.Response(
		"226 Close the data connection, the requested file operation is successful\r\n")
}

func commandDele(info []byte, driver FileDriver, require FileRequire) error {
	if len(info) == 0 {
		return require.Response("501 Parameter syntax error.Please input file name\r\n")
	}

	if !require.CheckAuth(DELETE) {
		Debugln(require.GetUserName() + " Has No Permisson To Delete File.")
		return require.Response("530 Parameter denied\r\n")
	}

	var path string
	if info[0] == '/' {
		path = require.GetRootDir() + string(info)
	} else {
		path = require.GetCurDir() + string(info)
	}

	err := driver.FileIsExist(path)

	if err == nil {
		/* delete the file*/
		if err := os.Remove(path); err != nil {
			Warnln("Delete file Failed", err)
			return require.Response(
				"451 Abort the operation of the request,there are local errors\r\n")
		} else {
			Debugln("Delete File " + path + "from" + require.GetUserName())
			return require.Response(
				"250 Requested File Operation Completed\r\n")
		}
	} else {
		return require.Response(
			"501 Parameter syntax error.Please input correctly file name\r\n")
	}

}

func FileProc(command string, info []byte, ftp *Ftp) error {
	if command == "STOR" {
		return commandStor(info, ftp, ftp)
	} else if command == "RETR" {
		return commandRetr(info, ftp, ftp)
	} else if command == "DELE" {
		return commandDele(info, ftp, ftp)
	}
	Fataln(command)
	return nil
}

func init() {
	register("STOR", FileProc)
	register("RETR", FileProc)
	register("DELE", FileProc)
}
