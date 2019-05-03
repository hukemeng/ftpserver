package test

import (
	. "ftpserver2"
	"testing"
)

func Test_config(t *testing.T) {

	Load_config("/Users/shangli/Go/src/ftpserver/conf.json")

	if Conf.Ftp_addr != "127.0.0.1" {
		t.FailNow()
	}

	if Conf.Ftp_port != "8090" {
		t.FailNow()
	}

	if Conf.Ftp_d_port != "8089" {
		t.FailNow()
	}

	if Conf.Users[0].Name != "root" {
		t.FailNow()
	}
	if Conf.Users[0].Pass != "root" {
		t.FailNow()
	}

}

/*. this test only for linux
func Test_config_multi_users(t *testing.T) {
	var max = 1000
	var shell = `cp conf.json conf2.json
		sed -i "7,$ d" conf.json
		max=` + strconv.Itoa(max) + `
		for ((i=1;i<${max};i++))
		do
			echo {\"name\": \"test${i}\", >> conf.json
			echo  \"pass\": \"pass${i}\", >> conf.json
			echo  \"root\": \"/home/FtpTest\"}, >> conf.json
		done
		echo {\"name\": \"test${max}\", >> conf.json
		echo  \"pass\": \"pass${max}\", >> conf.json
		echo  \"root\": \"/home/FtpTest\"} >> conf.json
		echo ]} >> conf.json`
	var cmd = exec.Command("sh", "-c", shell)
	_, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	load_config("conf.json")
	if len(conf.Users) != max {
		t.Fatal("check decode json Users length is not succeed:", len(conf.Users))
	}
	for index, user := range conf.Users {
		name_id := strconv.Itoa(index + 1)
		if user.Name != "test"+name_id {
			t.Fatal("user.name is not expect:", user.Name, "name_id:", name_id)
		}
		if user.Pass != "pass"+name_id {
			t.Fatal("user.pass is not expect:", user.Pass, "name_id:", name_id)
		}
	}
	cmd = exec.Command("sh", "-c", "mv -f conf2.json conf.json")
	cmd.Output()
}
*/
