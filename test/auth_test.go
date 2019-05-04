package test

import (
	. "ftpserver"
	"strconv"
	"testing"
)

func getStatus(t *testing.T, buf []byte) int {
	if len(buf) < 3 {
		t.Fatal(string(buf))
	}

	ret, err := strconv.Atoi(string(buf[:3]))
	check_err(err, t)
	return ret
}

func permissonCheck(t *testing.T, command string) {
	create_test_environment(t)
	defer clean_test_environment(t)

	ctl, _ := create_port_conn(t, "root", "root")

	/* read all control channel data */
	var buf = make([]byte, 2048)
	n, err := ctl.Read(buf)
	check_err(err, t)

	_, err = ctl.Write([]byte(command))
	check_err(err, t)

	n, err = ctl.Read(buf)
	check_err(err, t)

	if getStatus(t, buf[:n]) != 530 {
		t.Fatal(string(buf[:n]), command)
	}
}

func Test_Get(t *testing.T) {
	Conf.Users[0].Get = false
	permissonCheck(t, "RETR test\r\n")
	Conf.Users[0].Get = true
}

func Test_put(t *testing.T) {
	Conf.Users[0].Put = false
	permissonCheck(t, "STOR test\r\n")
	Conf.Users[0].Put = true
}

func Test_Del(t *testing.T) {
	Conf.Users[0].Delete = false
	permissonCheck(t, "DELE download.bin\r\n")
	Conf.Users[0].Delete = true
}

func Test_Recover(t *testing.T) {
	Conf.Users[0].Recover = false
	permissonCheck(t, "STOR download.bin\r\n")
	Conf.Users[0].Recover = true
}
