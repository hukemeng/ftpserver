package ftpserver

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	//errSetKeepalived = errors.New("Error:data connection set keepalived error.")
	errSetTimeout = errors.New("Error:data connection set timeout error.")
	errDataCreate = errors.New("Error:tcp listener set deadline error")
	errDataWrite  = errors.New("Error:data connection write error.")
	errDataRead   = errors.New("Error:data connection read error.")
	errListener   = errors.New("Error:listen error.")
)

type DataDriver interface {
	Write(msg []byte) (int, error)
	Read(msg []byte) (int, error)
	WriteAll(msg []byte) error
	WaitDataConn()
	DataClose()
	DataCreatePort(remote *net.TCPAddr)
	DataCreatePasv(*net.TCPListener)
	GetDataConn() *net.TCPConn
}

type DataRequire interface {
	Response(string) error
}

type DataConn struct {
	wait chan int
	conn *net.TCPConn
}

func NewDataConn() *DataConn {
	return &DataConn{
		wait: make(chan int, 1),
	}
}

func (data *DataConn) WaitDataConn() {
	<-data.wait
}

func (data *DataConn) Write(msg []byte) (int, error) {

	if data.conn == nil {
		return 0, errDataCreate
	}

	if err := data.conn.SetWriteDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return 0, errSetTimeout
	}

	n, err := data.conn.Write(msg)
	if err != nil {
		Warnln(err)
		return 0, errDataWrite
	}

	return n, nil
}

func (data *DataConn) WriteAll(msg []byte) error {

	if data.conn == nil {
		return errDataCreate
	}

	var length = len(msg)
	var start = 0
	if err := data.conn.SetWriteDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return errSetTimeout
	}

	for {
		n, err := data.conn.Write(msg[start:])
		if err != nil {
			Warnln(err)
			return errDataWrite
		}

		if n == length {
			break
		}
		start = n
	}
	return nil
}

func (data *DataConn) Read(msg []byte) (int, error) {

	if data.conn == nil {
		return 0, errDataCreate
	}

	if err := data.conn.SetReadDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return 0, errSetTimeout
	}
	num, err := data.conn.Read(msg)
	if err != nil {
		if err == io.EOF {
			return num, err
		}
		Warnln(err)
		return 0, errDataRead
	}
	return num, nil
}

func (data *DataConn) DataClose() {
	data.conn.SetLinger(-1)
	if err := data.conn.Close(); err != nil {
		Warnln(err)
	}
	data.conn = nil
}

func (data *DataConn) DataCreatePort(remote *net.TCPAddr) {

	conn, err := net.DialTCP(Conf.Ftp_network, nil, remote)
	if err != nil {
		log.Println(err)
	} else {
		if err := conn.SetKeepAlivePeriod(time.Second * 20); err != nil {
			Warnln(err)
		}
		data.conn = conn
	}
	data.wait <- 1
}

func (data *DataConn) DataCreatePasv(listen *net.TCPListener) {

	if err := listen.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		Warnln(err)
	}
	conn, err := listen.AcceptTCP()
	if err != nil {
		Warnln(err)
	} else {
		if err := conn.SetKeepAlivePeriod(20 * time.Second); err != nil {
			Warnln(err)
		}
		data.conn = conn
	}
	data.wait <- 1
}

func (data *DataConn) GetDataConn() *net.TCPConn {
	return data.conn
}

func commandPort(info []byte, driver DataDriver, requeire DataRequire) error {
	var portInfo = strings.Split(string(info), ",")
	if len(portInfo) != 6 {
		return requeire.Response("501 Parameter syntax error.Can't idenfiy port info\r\n")
	}

	if driver.GetDataConn() != nil {
		return requeire.Response(
			"550 The operation that did not execute.The data connection has been created\r\n")
	}

	var port int
	for i, bit := range portInfo {
		num, err := strconv.Atoi(bit)
		if err != nil || num < 0 || num > 255 {
			return requeire.Response("501 Parameter syntax error.Can't idenfiy port info\r\n")
		}

		if i >= 4 {
			port = port<<8 + num
		}
	}

	var remoteStr = fmt.Sprintf("%s.%s.%s.%s:%d",
		portInfo[0], portInfo[1], portInfo[2], portInfo[3], port)

	remote, err := net.ResolveTCPAddr("tcp4", remoteStr)
	if err != nil {
		return requeire.Response(
			"550 The operation that did not execute.There are local unknown errors\r\n")
	}

	go driver.DataCreatePort(remote)

	return requeire.Response("226 Creating data connection\r\n")
}

func commandPasv(info []byte, driver DataDriver, requeire DataRequire) error {
	if len(info) != 0 {
		return requeire.Response(
			fmt.Sprintf("501 Parameter syntax error,Can't idetify %s", string(info)))
	}

	listen, err := net.Listen("tcp4", Conf.Ftp_addr+":0")
	if err != nil {
		Warnln(err)
		return errListener
	}

	var ip = listen.Addr().(*net.TCPAddr).IP
	var port = listen.Addr().(*net.TCPAddr).Port

	var encode = fmt.Sprintf("(%d,%d,%d,%d,%d,%d) \r\n",
		ip[0], ip[1], ip[2], ip[3],
		(port&0xFF00)>>8, (port & 0x00FF))

	var msg = fmt.Sprintf("227 %s%s\r\n",
		"Entering Passive Mode", encode)

	go driver.DataCreatePasv(listen.(*net.TCPListener))

	return requeire.Response(msg)
}

func DataProc(command string, info []byte, ftp *Ftp) error {
	if command == "PORT" {
		return commandPort(info, ftp, ftp)
	} else if command == "PASV" {
		return commandPasv(info, ftp, ftp)
	}

	Fataln(command)
	return nil
}

func init() {
	register("PORT", DataProc)
	register("PASV", DataProc)
}
