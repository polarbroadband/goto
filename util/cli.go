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
// GetInput display prompt and return trimed input string
// return empty string if input not valid
func GetInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt + ": ")
	s, err := reader.ReadString('\n')
	if err != nil {
		log.WithError(err).Warnf("erroneous input of %s", prompt)
		return ""
	}
	return strings.TrimSpace(s)
}

// GetCred prompt for entering username and password
// return empty strings if input not valid
// no screen echo for entering password
func GetCred() (string, string) {
	uid := GetInput("Username")
	if uid == "" {
		return "", ""
	}
	fmt.Print("Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.WithError(err).Warn("erroneous input of password")
		return "", ""
	}
	fmt.Println()
	return uid, strings.TrimSpace(string(bytePassword))
}
