package ftpserver

import "github.com/pkg/errors"

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
	if fn, ok := cmdModules[command]; ok {
		return fn(command, info, ftp)
	}
	if command == "QUIT" {
		return normalExit
	}
	if command == "TYPE" {
		return ftp.Response("220 Binary\r\n")
	}
	return ftp.Response("502 Command not implemented\r\n")
}
