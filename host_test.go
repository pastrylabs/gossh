package gossh

import (
	"fmt"
	"os"
	"os/user"
	"testing"
)

var sudopass string
var currentuser string

func init() {
	sudopass = os.Getenv("SUDOPASS") // TODO - This is probably not a good idea
	if sudopass == "" {
		fmt.Println("###### Remember to set the env var SUDOPASS using \" export SUDOPASS=pwd\"")
	}
	u, _ := user.Current()
	currentuser = u.Username
}

func TestNSpaces(t *testing.T) {

	var tests = []struct {
		in     int
		expect string
	}{
		{-9, ""},
		{-2, ""},
		{-1, ""},
		{0, ""},
		{1, " "},
		{2, " │"},
		{9, " │ │ │ │ "},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d", test.in), func(t *testing.T) {

			got := nSpaces(test.in)

			if got != test.expect {
				t.Errorf("value: got \"%s\" - expect \"%s\"", got, test.expect)
			}

		})
	}
}

/*
func TestLocal(t *testing.T) {

	l := local{sudopass: sudopass}

	var tests = []struct {
		cmd    string
		sudo   bool
		user   string
		stdin  string
		expect Response
	}{
		{
			cmd: `echo "hello"`,
			expect: Response{
				Stdout:     "hello",
				Stderr:     "",
				ExitStatus: 0,
			},
		},
		{
			cmd: `echo -n "hello"`,
			expect: Response{
				Stdout:     "hello",
				Stderr:     "",
				ExitStatus: 0,
			},
		},
		{
			cmd: `somecommandthatdoesnotexist`,
			expect: Response{
				Stdout:     "",
				Stderr:     "bash: somecommandthatdoesnotexist: command not found",
				ExitStatus: 127,
			},
		},
		{
			cmd: `cat filethatdoesntexist`,
			expect: Response{
				Stdout:     "",
				Stderr:     "cat: filethatdoesntexist: No such file or directory",
				ExitStatus: 1,
			},
		},
		{
			cmd:   `sed s/a/X/ | sed s/c/Z/`,
			stdin: "abc",
			expect: Response{
				Stdout:     "XbZ",
				Stderr:     "",
				ExitStatus: 0,
			},
		},
		{
			cmd:   `sed s/a/X/ | sed s/c/Z/`,
			sudo:  true,
			user:  "root",
			stdin: "abc",
			expect: Response{
				Stdout:     "XbZ",
				Stderr:     "",
				ExitStatus: 0,
			},
		},
		{
			cmd:  `ls /root`,
			sudo: true,
			expect: Response{
				Stdout:     "",
				Stderr:     "",
				ExitStatus: 0,
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s %v %s", test.cmd, test.stdin, test.sudo, test.user), func(t *testing.T) {

			got, err := l.run(test.cmd, test.stdin, test.sudo, test.user)
			if err != nil {
				t.Errorf("errored: %v", err)
			}

			if got.Stdout != test.expect.Stdout {
				t.Errorf("stdout: got \"%s\" - expect \"%s\"", got.Stdout, test.expect.Stdout)
			}
			if got.Stderr != test.expect.Stderr {
				t.Errorf("stderr: got \"%s\" - expect \"%s\"", got.Stderr, test.expect.Stderr)
			}
			if got.ExitStatus != test.expect.ExitStatus {
				t.Errorf("exitstatus: got \"%d\" - expect \"%d\"", got.ExitStatus, test.expect.ExitStatus)
			}

		})
	}
}
*/
