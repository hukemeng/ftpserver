package ftpserver

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
)

type Ftp struct {
	*User
	*Entry
	*DataConn
	*File
	*Controller
}

func newFtp(conn *net.TCPConn) *Ftp {
	return &Ftp{
		User:       NewUser(),
		Entry:      new(Entry),
		DataConn:   NewDataConn(),
		File:       new(File),
		Controller: NewControler(conn),
	}
}

type cmdFn func(string, []byte, *Ftp) error

var cmdModules = make(map[string]cmdFn)

var normalExit = errors.New("Normal Exit.")

func register(command string, fn cmdFn) {
	if _, ok := cmdModules[command]; ok {
		Fataln("Repeated registration：", command)
	}
	cmdModules[command] = fn
}

func PerformHandle(ftp *Ftp, command string, info []byte) error {
	//Debugln(command, ":", string(info))
	if command == "" {
		return nil
	}

	if fn, ok := cmdModules[command]; ok {
		return fn(command, info, ftp)
	}
	if command == "QUIT" {
		return normalExit
	}
	if command == "TYPE" {
		return ftp.Response("220 Binary\r\n")
	}
	Debugln("command " + command + " has no implemented")
	return ftp.Response("502 Command not implemented\r\n")
}

func decode(msg []byte) (string, []byte) {

	var cmd = string(msg)
	var info []byte

	for i, r := range msg {
		if r == ' ' {
			cmd = string(msg[:i])
			if i+1 < len(msg) {
				info = msg[i+1:]
			}
		}
	}

	return cmd, info
}

func ftpPerform(ftp *Ftp) {
	var reader = ftp.Reader()
	for {
		msg, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			Warnln(err)
			break
		}

		cmd, info := decode(msg)

		if err := PerformHandle(ftp, cmd, info); err != nil {
			if err != normalExit {
				Warnln("can't handle the Error, exit.Error:", err)
			}
			break
		}
	}
	ftp.ExitControl()
}

func Start() {

	var listen, err = net.Listen("tcp4", Conf.Ftp_addr+":"+Conf.Ftp_port)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("tcp4: Listen succeed.", Conf.Ftp_addr+":"+Conf.Ftp_port)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		//Debugln("accept control connection from ", conn.RemoteAddr())

		var ftp = newFtp(conn.(*net.TCPConn))
		if ftp.Welcome() != nil {
			ftp.ExitControl()
			continue
		}

		go ftpPerform(ftp)
	}
}

/* some public function */
func println(caller int, prefix string, v ...interface{}) {
	fn, _, line, ok := runtime.Caller(caller)
	if !ok {
		log.Println("Unkown Error: Get runtime Caller Error.\n"+
			"Target print:", v)
	}

	var logprefix = fmt.Sprintf(
		"%s FUNC:%s LINE:%d", prefix, runtime.FuncForPC(fn).Name(), line)

	log.Println(logprefix, v)
}

func Warnln(v ...interface{}) {
	println(2, "[WARN]", v...)
}

var Fataln = log.Fatalln

func Debugln(v ...interface{}) {
	println(2, "[DEBUG]", v...)
}
