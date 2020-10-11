package sendmail

import (
	"io/ioutil"
	"mime"
	"os/exec"
)

var (
	sendmailCmd		string

)

func init() {
	sendmailCmd = "/usr/sbin/sendmail"
}

func Send(fromEmail, toEmail, msg string) error {
	sendmail := exec.Command(sendmailCmd, "-f", fromEmail, toEmail)

	stdin, err := sendmail.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := sendmail.StdoutPipe()
	if err != nil {
		return err
	}

	sendmail.Start()
	stdin.Write([]byte(msg))
	stdin.Close()
	ioutil.ReadAll(stdout)
	sendmail.Wait()

	return nil
}

func SendMail(fromEmail, toEmail, subj, body string) error {
	msg := "From: <" + fromEmail + ">\n"
	msg += "To: <" + toEmail + ">\n"

	enc := new(mime.WordEncoder)
	msg += "Subject: " + enc.Encode("utf-8", subj)

	msg += "\n\n"+body

	return Send(fromEmail, toEmail, msg)
}
