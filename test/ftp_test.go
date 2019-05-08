package test

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	. "ftpserver"
)

const default_test_path = "/home/FtpTest"
const default_download_path = default_test_path + "/download.bin"
const default_upload_path = default_test_path + "/upload.bin"

var check_err = func(err error, t *testing.T) {
	fn, _, line, ok := runtime.Caller(1)
	if !ok {
		t.Fatal()
	}
	if err != nil {
		t.Fatal("FUNC:"+runtime.FuncForPC(fn).Name()+
			",LINE:"+strconv.Itoa(line), err)
	}
}

func create_test_environment(t *testing.T) {

	err := os.Mkdir(default_test_path, os.ModePerm)
	if err != nil {
		// t.Fatal(err)
	}

	down_file, err := os.OpenFile(default_download_path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		check_err(down_file.Close(), t)
	}()

	up_file, err := os.OpenFile(default_upload_path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		check_err(up_file.Close(), t)
	}()

	var buf = make([]byte, 65536)
	for i := 0; i < 32; i++ {
		for j := 0; j < 65536; j++ {
			buf[j] = byte(rand.Intn(255))
		}
		if _, err := down_file.Write(buf); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 32; i++ {
		for j := 0; j < 65536; j++ {
			buf[j] = byte(rand.Intn(255))
		}
		if _, err := up_file.Write(buf); err != nil {
			t.Fatal(err)
		}
	}
}

func clean_test_environment(t *testing.T) {
	if err := os.RemoveAll(default_test_path); err != nil {
		t.Fatal(err)
	}
}

func create_control(t *testing.T, user string, pass string) net.Conn {
	var ctl, err = net.Dial("tcp4", Conf.Ftp_addr+":"+Conf.Ftp_port)
	check_err(err, t)

	_, err = ctl.Write([]byte(
		"USER " + user + "\r\nPASS " + pass + "\r\n"))
	check_err(err, t)
	return ctl
}

func create_pasv_conn(t *testing.T, user string, pass string) (net.Conn, net.Conn) {

	var ctl = create_control(t, user, pass)

	_, err := ctl.Write([]byte("PASV\r\n"))
	check_err(err, t)

	var pasv string

	var reader = bufio.NewReader(ctl)
	for {
		msg, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}

		if len(msg) < 3 {
			break
		}

		if string(msg[:3]) == "227" &&
			strings.Contains(string(msg), "Passive") {
			pasv = string(msg)
		}
	}

	var start = strings.Index(pasv, "(")
	var end = strings.Index(pasv, ")")
	if end <= start {
		t.Fatal(pasv)
	}

	var ss = strings.Split(pasv[start+1:end], ",")
	var port1, _ = strconv.Atoi(ss[4])
	var port2, _ = strconv.Atoi(ss[5])

	var port = strconv.Itoa(port1*256 + port2)
	data, err := net.Dial("tcp4", "127.0.0.1:"+port)
	check_err(err, t)

	return ctl, data
}

func create_port_conn(t *testing.T, user string, pass string) (net.Conn, net.Conn) {

	var ctl = create_control(t, user, pass)

	port_lis, err := net.Listen("tcp4", "127.0.0.1:0")
	check_err(err, t)

	var ip = port_lis.Addr().(*net.TCPAddr).IP
	var port = port_lis.Addr().(*net.TCPAddr).Port

	var encode_port = fmt.Sprintf("%d,%d,%d,%d,%d,%d",
		ip[0], ip[1], ip[2], ip[3],
		(port&0xFF00)>>8, (port & 0x00FF))

	_, err = ctl.Write([]byte("PORT " + encode_port + "\r\n"))
	check_err(err, t)

	data, err := port_lis.Accept()
	check_err(err, t)

	return ctl, data
}

var wg sync.WaitGroup

func download(t *testing.T, user string, pass string, prefix string) {

	defer wg.Done()

	var ctl, data net.Conn

	if rand.Intn(255)%2 == 1 {
		ctl, data = create_port_conn(t, user, pass)
	} else {
		ctl, data = create_pasv_conn(t, user, pass)
	}

	defer func() {
		_, err := ctl.Write([]byte("QUIT\r\n"))
		check_err(err, t)
		check_err(ctl.Close(), t)
	}()

	_, err := ctl.Write([]byte("RETR download.bin\r\n"))
	check_err(err, t)

	var buf = make([]byte, 2048)
	var path = default_test_path + "/" + prefix + user + "download.bin"

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	check_err(err, t)
	//defer check_err(os.Remove(path), t)

	for {
		n, err := data.Read(buf)
		if err != nil && err != io.EOF {
			check_err(err, t)
		}
		if err == io.EOF {
			check_err(data.Close(), t)
			break
		}

		_, err = f.Write(buf[:n])
		check_err(err, t)
	}

	//check_err(f.Close(), t)

	check_err(f.Sync(), t)
	/* wait data connection succeed */
	for {
		n, err := ctl.Read(buf)
		check_err(err, t)

		if strings.Contains(string(buf[:n]), "successful") {
			break
		}
	}

	if !compare(path, default_download_path) {
		t.Fatal(path, default_download_path, "cmp command is not same!")
	}
}

func upload(t *testing.T, user string, pass string, prefix string) {
	defer wg.Done()

	var ctl, data net.Conn

	if rand.Intn(255)%2 == 1 {
		ctl, data = create_port_conn(t, user, pass)
	} else {
		ctl, data = create_pasv_conn(t, user, pass)
	}

	defer func() {
		_, err := ctl.Write([]byte("QUIT\r\n"))
		check_err(err, t)
		check_err(ctl.Close(), t)
	}()

	var file_name = prefix + user + "upload.bin"
	var path = default_test_path + "/" + file_name
	_, err := ctl.Write([]byte("STOR " + file_name + "\r\n"))
	check_err(err, t)

	f, err := os.Open(default_upload_path)
	check_err(err, t)

	var buf = make([]byte, 2048)
	_, err = ctl.Read(buf)
	check_err(err, t)

	for {
		n, err := f.Read(buf)
		if err != nil && err == io.EOF {
			break
		}
		check_err(err, t)

		n, err = data.Write(buf[:n])
		if err != nil {
			t.Fatal(err)
		}
	}

	check_err(data.(*net.TCPConn).SetLinger(-1), t)
	check_err(data.Close(), t)

	/* wait data connection succeed */
	for {
		n, err := ctl.Read(buf)
		check_err(err, t)

		if strings.Contains(string(buf[:n]), "successful") {
			break
		}
	}

	time.Sleep(time.Second)

	if !compare(path, default_upload_path) {
		t.Fatal(path, default_upload_path, "cmp command is not same!")
	}
}

func compare(file1 string, file2 string) bool {
	var command = "cmp " + file1 + " " + file2

	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	if len(output) != 0 {
		return false
	}

	return true
}

func Test_Transfer(t *testing.T) {
	create_test_environment(t)
	//defer clean_test_environment(t)

	const second, speed = 10, 100

	for i := 0; i < second; i++ {
		for j := 0; j < speed; j++ {
			wg.Add(1)
			go download(t, "root", "root", strconv.Itoa(i*speed+j))
		}

		time.Sleep(time.Second)
	}
	wg.Wait()

	for i := 0; i < second; i++ {
		for j := 0; j < speed; j++ {
			wg.Add(1)
			go upload(t, "root", "root", strconv.Itoa(i*speed+j))
		}

		time.Sleep(time.Second)
	}
	wg.Wait()
	/* for recover */
	//upload(t, "root", "root", "1")
}

func init() {
	go Start()
	time.Sleep(1 * time.Second)
}
