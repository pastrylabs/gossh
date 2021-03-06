package local

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/lithammer/shortuuid"
)

var testsudopass string
var testdir string

func TestMain(m *testing.M) {
	// set sudopass
	testsudopass = os.Getenv("GOSSH_SUDOPASS")
	if testsudopass == "" {
		log.Fatal("missing GOSSH_SUDOPASS env variable")
	}

	testdir = fmt.Sprintf("%s/gossh-%s", os.TempDir(), shortuuid.New())
	err := os.Mkdir(testdir, 0777)
	if err != nil {
		log.Fatal("could not create testdir:", testdir, "-", err)
	}

	code := m.Run()

	err = os.RemoveAll(testdir)
	if err != nil {
		log.Fatal("could not clear testdir", testdir, "-", err)
	}

	os.Exit(code)
}

func TestMkdir(t *testing.T) {

	l, err := New(testsudopass)

	tests := []struct {
		activeuser string
		path       string
	}{
		{l.user, testdir + "/gossh_testmkdir2-" + shortuuid.New()},
		{"root", testdir + "/gossh_testmkdir1-" + shortuuid.New()},
	}

	if err != nil {
		t.Fatal("could not get local:", err)
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {

			l.activeuser = test.activeuser
			err = l.Mkdir(test.path)
			if err != nil {
				t.Error("test file creation errored", err)
			}

			stat := exec.Command("stat", "--format=%U", test.path)
			o, err := stat.Output()
			if err != nil {
				t.Error("stat failed", err)
			}

			owner := strings.Trim(string(o), " \n")
			if owner != test.activeuser {
				t.Errorf("wrong ownership. got '%s'", o)
			}

		})
	}

}

func TestCreate(t *testing.T) {

	l, err := New(testsudopass)
	if err != nil {
		t.Fatal("could not get local:", err)
	}

	tests := []struct {
		activeuser string
		path       string
		content    string
	}{
		{l.user, testdir + "/testcreate1", "some\ncontent"},
		{"root", testdir + "/testcreate2", "content"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {

			l.activeuser = test.activeuser
			f, err := l.Create(test.path)
			if err != nil {
				t.Fatal("create errored", err)
			}

			_, err = f.Write([]byte(test.content))
			if err != nil {
				t.Fatal("could not write to file", err)
			}

			f.Close()

			stat := exec.Command("stat", "--format=%U", test.path)
			o, err := stat.Output()
			if err != nil {
				t.Error("stat failed", err)
			}

			owner := strings.Trim(string(o), " \n")
			if owner != test.activeuser {
				t.Errorf("wrong ownership. got '%s'", o)
			}

			content, err := ioutil.ReadFile(test.path)
			if err != nil {
				t.Error("could not read file", err)
			}

			if string(content) != test.content {
				t.Errorf(`file content wrong - expect: "%s", got : "%s"`, test.content, string(content))
			}

		})
	}

}

func TestOpen(t *testing.T) {

	l, err := New(testsudopass)
	if err != nil {
		t.Fatal("could not get local:", err)
	}

	tests := []struct {
		activeuser string
		path       string
		content    string
	}{
		{l.user, testdir + "/testopen1", "some\ncontent"},
		{"root", testdir + "/testopen2", "content"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {

			l.activeuser = test.activeuser

			fc, _ := l.Create(test.path)
			fc.Write([]byte(test.content))
			fc.Close()

			f, err := l.Open(test.path)
			if err != nil {
				t.Fatal("open errored", err)
			}
			defer f.Close()

			content, err := ioutil.ReadAll(f)
			if err != nil {
				t.Fatal("could not read from file", err)
			}

			if string(content) != test.content {
				t.Errorf(`file content wrong - expect: "%s", got : "%s"`, test.content, string(content))
			}

		})
	}

}

func TestAppend(t *testing.T) {

	l, err := New(testsudopass)
	if err != nil {
		t.Fatal("could not get local:", err)
	}

	orignalcontent := "somecontent\nwith\nnewlines"

	tests := []struct {
		activeuser string
		path       string
		content    string
	}{
		{l.user, testdir + "/testappend1", "some\ncontent"},
		{"root", testdir + "/testappend2", "content"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {

			l.activeuser = test.activeuser

			fc, _ := l.Create(test.path)
			fc.Write([]byte(orignalcontent))
			fc.Close()

			f, err := l.Append(test.path)
			if err != nil {
				t.Fatal("open errored", err)
			}

			_, err = f.Write([]byte(test.content))
			if err != nil {
				t.Fatal("could not write to file", err)
			}

			f.Close()

			content, err := ioutil.ReadFile(test.path)
			if err != nil {
				t.Error("could not read file", err)
			}

			expectedcontent := orignalcontent + test.content

			if string(content) != expectedcontent {
				t.Errorf(`file content wrong - expect: "%s", got : "%s"`, expectedcontent, string(content))
			}

		})
	}

}
