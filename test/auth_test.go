package test

import (
	. "ftpserver"
	"strconv"
	"testing"
	"time"
)

func getStatus(t *testing.T, buf []byte) int {
	if len(buf) < 3 {
		t.Fatal(string(buf))
	}

	ret, err := strconv.Atoi(string(buf[:3]))
	check_err(err, t)
	return ret
}

func authCheck(t *testing.T, command string, status int) {
	create_test_environment(t)
	defer clean_test_environment(t)

	ctl := create_control(t, "root", "root")
	time.Sleep(time.Second)

	/* read all control channel data */
	var buf = make([]byte, 2048)
	n, err := ctl.Read(buf)
	check_err(err, t)

	_, err = ctl.Write([]byte(command))
	check_err(err, t)

	n, err = ctl.Read(buf)
	check_err(err, t)

	if getStatus(t, buf[:n]) != status {
		t.Fatal(string(buf[:n]), command, status)
	}
}

func Test_Get(t *testing.T) {
	Conf.Users[0].Get = false
	authCheck(t, "RETR test\r\n", 530)
	Conf.Users[0].Get = true
}

func Test_Put(t *testing.T) {
	Conf.Users[0].Put = false
	authCheck(t, "STOR test\r\n", 530)
	Conf.Users[0].Put = true
}

func Test_Del(t *testing.T) {
	Conf.Users[0].Delete = false
	authCheck(t, "DELE download.bin\r\n", 530)
	Conf.Users[0].Delete = true
}

func Test_Recover(t *testing.T) {
	Conf.Users[0].Recover = false
	authCheck(t, "STOR download.bin\r\n", 530)
	Conf.Users[0].Recover = true
}

func Test_Mkdr(t *testing.T) {
	Conf.Users[0].MkDir = false
	authCheck(t, "MKD testMkdir\r\n", 530)
	Conf.Users[0].MkDir = true

	authCheck(t, "MKD testMkdir\r\n", 257)
}

func Test_DelDir(t *testing.T) {
	Conf.Users[0].DelDir = false
	authCheck(t, "RMD testMkdir\r\n", 530)
	Conf.Users[0].DelDir = true

	Conf.Users[0].MkDir = true
	authCheck(t, "MKD testMkdir\r\n", 257)

	authCheck(t, "RMD testMkdir\r\n", 250)
}
