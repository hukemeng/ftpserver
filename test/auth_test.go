package test

import (
	. "ftpserver2"
	"testing"
)

func Test_Get(t *testing.T) {
	create_test_environment(t)
	defer clean_test_environment(t)

	Conf.Users[0].Get = false


}
