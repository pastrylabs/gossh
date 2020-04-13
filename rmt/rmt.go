package rmt

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/krilor/gossh"
	"github.com/krilor/gossh/rmt/suftp"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Package rmt contains functionality for Remote targets
// A Remote target is a target that is connected to using SSH

// Remote represents a Remote target, connected to over SSH
type Remote struct {
	addr string

	// auth
	conn     *ssh.Client
	connuser string
	// sudopass is connusers sudo password
	sudopass string

	// the user currently operating as
	activeuser string

	// sftp holds all sftp connections. key is username. Pointer?
	sftp map[string]*sftp.Client
}

// New returns a new Remote target from connection details
func New(addr string, user string, sudopass string, hostkeycallback ssh.HostKeyCallback, auths ...ssh.AuthMethod) (Remote, error) {

	r := Remote{
		addr:       addr,
		connuser:   user,
		sudopass:   sudopass,
		activeuser: user,
		sftp:       map[string]*sftp.Client{},
	}

	cc := ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: hostkeycallback,
	}

	var err error
	r.conn, err = ssh.Dial("tcp", addr, &cc)
	if err != nil {
		return r, errors.Wrapf(err, "unable to establish ssh connection to %s", addr)
	}

	return r, nil

}

// Close closes all underlying connections
func (r Remote) Close() error {
	for _, c := range r.sftp {
		c.Close()
	}

	return r.conn.Close()
}

// sftpClient returns a sftp client for r.activeuser
// if client does not exist, it will be created
func (r Remote) sftpClient() (*sftp.Client, error) {
	var c *sftp.Client
	var err error
	var ok bool
	c, ok = r.sftp[r.activeuser]
	if ok {
		return c, nil
	}
	// need to create a new connection
	if r.sudo() {
		c, err = suftp.NewSudoClient(r.conn, r.activeuser, r.sudopass)
	} else {
		c, err = sftp.NewClient(r.conn)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not start sftp connection for %s", r.activeuser)
	}
	r.sftp[r.activeuser] = c
	return c, nil
}

// sudo reports if operations must be done using sudo
func (r Remote) sudo() bool {
	return r.activeuser != r.connuser
}

// As returns a new Remote that will use the same underlying connections, but all operations will be done as user.
//
// No tests are done in this method. If user does not exist or does not have sudo rights, that will only be evident when trying to use methods on the returned object.
func (r Remote) As(user string) Remote {
	r.activeuser = user
	return r
}

// Mkdir creates a directory in the given path
func (r Remote) Mkdir(path string) error {
	sftp, err := r.sftpClient()
	if err != nil {
		return err
	}
	return sftp.Mkdir(path)
}

// Put puts the contents of a Reader on a path on the Remote machine
//
// Inspiration:
// https://github.com/laher/scp-go/blob/master/scp/toRemote.go
// https://gist.github.com/jedy/3357393
//
// SCP notes:
// https://web.archive.org/web/20170215184048/https://blogs.oracle.com/janp/entry/how_the_scp_protocol_works
// https://en.wikipedia.org/wiki/Secure_copy#cite_note-Pechanec-2
func (r *Remote) put(content io.Reader, size int64, path string, mode uint32) error {

	// consider using github.com/pkg/sftp

	session, err := r.conn.NewSession()
	if err != nil {
		return errors.Wrap(err, "failed to create scp session")
	}
	defer session.Close()

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()

		// header message has the format C<mode> <size> <filename>
		fmt.Fprintf(w, "C%04o %d %s\n", mode, size, filepath.Base(path))

		io.Copy(w, content)

		// transfer end with \x00
		fmt.Fprint(w, "\x00")
	}()

	if b, err := session.CombinedOutput(fmt.Sprintf("/usr/bin/scp -tr %s", path)); err != nil {
		return errors.Wrapf(err, "unable to copy content: %s", string(b))
	}

	return nil
}

// User returns the currently active user
func (r Remote) User() string {
	return r.activeuser
}

func (r *Remote) String() string {
	return fmt.Sprintf("%s@%s", r.connuser, r.addr)
}

// run runs cmd on Remote
func (r *Remote) run(cmd string, stdin string, sudo bool, user string) (gossh.Response, error) {

	session, err := r.conn.NewSession()
	resp := gossh.Response{}

	if err != nil {
		return resp, errors.Wrap(err, "unable to create new session")
	}
	defer session.Close()

	o := bytes.Buffer{}
	e := bytes.Buffer{}

	session.Stdout = &o
	session.Stderr = &e

	// TODO - consider using session.Shell - http://networkbit.ch/golang-ssh-client/#multiple_commands
	if sudo {
		session.Stdin = strings.NewReader(r.sudopass + "\n" + stdin + "\n")
		if user == "" || user == "-" {
			user = "root"
		}
		sudocmd := fmt.Sprintf(`sudo -k -S -u %s bash -c "%s"`, user, cmd)
		err = session.Run(sudocmd)

	} else {
		session.Stdin = strings.NewReader(stdin + "\n")
		err = session.Run(cmd)
	}

	resp.Stdout = scrubStd(o.String())
	resp.Stderr = scrubStd(e.String())

	if err != nil {

		switch t := err.(type) {
		case *ssh.ExitError:
			resp.ExitStatus = t.Waitmsg.ExitStatus()
		case *ssh.ExitMissingError:
			resp.ExitStatus = -1
		default:
			return resp, errors.Wrap(err, "run of command failed")
		}

	} else {
		resp.ExitStatus = 0
	}

	return resp, nil
}

// AgentAuths is a helper function to get SSH keys from an ssh agent.
// If any errors occur, an empty PublicKeys ssh.AuthMethod will be returned.
func AgentAuths() ssh.AuthMethod {

	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return ssh.PublicKeys()
	}
	defer conn.Close()

	agentClient := agent.NewClient(conn)

	// TODO how do we close these clients?
	return ssh.PublicKeysCallback(agentClient.Signers)
}

// sudopattern matches sudo prompt
var sudopattern *regexp.Regexp = regexp.MustCompile(`\[sudo\] password for [^:]+: `)

// scrubStd cleans an out/err string. Removes trailing newline and sudo prompt.
func scrubStd(in string) string {
	return sudopattern.ReplaceAllString(strings.Trim(in, "\n"), "")
}