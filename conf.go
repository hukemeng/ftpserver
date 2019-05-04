package ftpserver

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type userConf struct {
	Name    string
	Pass    string
	Root    string
	Get     bool `json:"get"`
	Put     bool `json:"put"`
	Delete  bool `json:"delete"`
	Recover bool `json:"recover"`
	DelDir  bool `json:"deldir"`
	MkDir   bool `json:"mkdir"`
}

type ftpserverConf struct {
	Ftp_addr      string `json:"ftp_addr"`
	Ftp_port      string `json:"ftp_port"`
	Ftp_network   string
	Ftp_d_port    string     `json:"ftp_data_port"`
	Ftp_d_timeout int        `json:"ftp_data_timeout"`
	Users         []userConf `json:"user"`
}

var Conf = ftpserverConf{}

func Load_config(path string) {

	const max_size = 5 * 1024 * 1024

	if stat, err := os.Stat(path); err != nil {
		log.Fatal(err)
	} else {
		if stat.Size() > max_size {
			log.Fatal("Error: the file ", path, " size is too big!")
		}
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(data, &Conf)
	if err != nil {
		log.Fatalln(err, data, path)
	}

	if strings.Contains(Conf.Ftp_addr, ":") {
		Conf.Ftp_network = "tcp6"
	} else {
		Conf.Ftp_network = "tcp4"
	}
}

func init() {
	const conf_file = "/Users/shangli/Go/src/ftpserver/conf.json"
	Load_config(conf_file)

}
