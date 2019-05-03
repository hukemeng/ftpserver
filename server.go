package ftpserver

import (
	"bufio"
	"io"
	"log"
	"net"
)

type Ftp struct {
	*User
	*Entry
	*DataConn
	*File
	ctl *net.TCPConn
}

func (ftp *Ftp) welcome() error {
	_, err := ftp.ctl.Write([]byte("220 HKM FTP Server Ready\r\n"))
	return err
}

func (ftp *Ftp) Response(msg string) error {
	_, err := ftp.ctl.Write([]byte(msg))
	return err
}

func newFtp(conn *net.TCPConn) *Ftp {
	return &Ftp{
		User:     NewUser(),
		Entry:    new(Entry),
		DataConn: NewDataConn(),
		File:     new(File),
		ctl:      conn,
	}
}

func exitFtp(ftp *Ftp) {
	if err := ftp.ctl.Close(); err != nil {
		Warnln(err)
	}
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
	var reader = bufio.NewReader(ftp.ctl)
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
	exitFtp(ftp)
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

		//log.Println("accept control connection.", conn.RemoteAddr())

		var ftp = newFtp(conn.(*net.TCPConn))
		if ftp.welcome() != nil {
			exitFtp(ftp)
			continue
		}

		go ftpPerform(ftp)
	}
}
