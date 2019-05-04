package ftpserver

import (
	"bufio"
	"net"
)

type CtrlDriver interface {
	Welcome() error
	Response(string) error
	ExitControl()
	Reader() *bufio.Reader
}

type Controller struct {
	ctrl *net.TCPConn
}

func (ctrl *Controller) Welcome() error {
	_, err := ctrl.ctrl.Write([]byte("220 HKM FTP Server Ready\r\n"))
	return err
}

func (ctrl *Controller) Response(msg string) error {
	_, err := ctrl.ctrl.Write([]byte(msg))
	return err
}

func (ctrl *Controller) ExitControl() {
	if err := ctrl.ctrl.Close(); err != nil {
		Warnln(err)
	}
}

func (ctrl *Controller) Reader() *bufio.Reader {
	return bufio.NewReader(ctrl.ctrl)
}

func NewControler(conn *net.TCPConn) *Controller {
	return &Controller{
		ctrl: conn,
	}
}
