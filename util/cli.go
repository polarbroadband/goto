package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

/* ****************************************
cli utility functions
**************************************** */

// GetCred prompt for entering username and password
// return empty strings if input not valid
// no screen echo for entering password
func GetCred() (uid, pwd string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Username: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		log.WithError(err).Warn("erroneous username input")
		return "", ""
	}
	fmt.Print("Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.WithError(err).Warn("erroneous password input")
		return "", ""
	}
	fmt.Println()
	return strings.TrimSpace(name), strings.TrimSpace(string(bytePassword))
}
