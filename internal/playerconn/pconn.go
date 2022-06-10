/*
	QD MUCK is Copyright 2022 by QueeredDeer under a GPL-3.0-or-later license.

	This file is part of QD MUCK.

	QD MUCK is free software: you can redistribute it and/or modify it under
	the terms of the GNU General Public License as published by the Free
	Software Foundation, either version 3 of the License, or (at your option)
	any later version.

	QD MUCK is distributed in the hope that it will be useful, but WITHOUT
	ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
	FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License
	for more details.

	You should have received a copy of the GNU General Public License along
	with QD MUCK. If not, see <https://www.gnu.org/licenses/>.
*/

package playerconn

import (
	"bufio"
	"errors"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/QueeredDeer/qd-muck/internal/configparser"
)

const invalidUser string = "Invalid login attempt"
const invalidPass string = "Invalid username or password"

var connParser *regexp.Regexp = regexp.MustCompile(`connect\s+(?P<user>\S+)\s+(?P<password>\S.*)`)
var uindex int = connParser.SubexpIndex("user")
var pindex int = connParser.SubexpIndex("password")

type userProfile struct {
	Name         string
	PasswordHash []byte
	LoginStrikes int
	Timeout      time.Time
}

func (profile *userProfile) clearStrikes() {
	// FIXME: write these to db instead
	profile.LoginStrikes = 0
	profile.Timeout = time.Unix(0, 0)
}

func parseConnect(line string) (string, string, bool) {
	cmd := strings.TrimSpace(string(line))

	if !connParser.MatchString(cmd) {
		return "", "", false
	}

	matches := connParser.FindStringSubmatch(cmd)
	user := matches[uindex]
	pass := matches[pindex]

	return user, pass, true
}

func listenLogin(conn net.Conn) (string, string) {
	reader := bufio.NewReader(conn)

	// TODO: add timeout to this loop?
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Write([]byte(err.Error()))
			continue
		}

		user, pass, ok := parseConnect(line)
		if !ok {
			conn.Write([]byte("Unrecognized command format"))
			continue
		}

		return user, pass
	}
}

func userOffline(user string) bool {
	// FIXME: implement
	return false
}

func loadProfile(user string) (*userProfile, error) {
	// FIXME: implement
	return &userProfile{}, errors.New("profile loading not implemented")
}

func checkUser(user string) (*userProfile, error) {
	// check user exists in DB, user is not currently logged in, and not timed out
	offline := userOffline(user)
	if !offline {
		return &userProfile{}, errors.New("user '" + user + "' already logged in")
	}

	profile, err := loadProfile(user)
	if err != nil {
		return profile, err
	}

	if time.Now().Before(profile.Timeout) {
		return &userProfile{}, errors.New("user '" + user + "' has login timeout")
	}

	return profile, nil
}

func checkPassword(uprofile *userProfile, password string) error {
	return bcrypt.CompareHashAndPassword(uprofile.PasswordHash, []byte(password))
}

func timeoutUsers(strikes *map[string]int, lcount int) {
	// FIXME: implement
}

func validateLogin(conn net.Conn, lsettings *configparser.LoginSettings) (string, bool) {
	user := ""
	attempts := 0
	passwordStrikes := make(map[string]int)

	for attempts < lsettings.LoginAttempts {
		user, password := listenLogin(conn)

		uprofile, uerr := checkUser(user)
		if uerr != nil {
			attempts++
			conn.Write([]byte(invalidUser))
			logrus.WithFields(logrus.Fields{
				"ip":     conn.RemoteAddr().String(),
				"player": user,
				"error":  uerr.Error(),
			}).Warn("Invalid username in login attempt")
			continue
		}

		// validate password
		perr := checkPassword(uprofile, password)
		if perr == nil {
			// password validated, safe to login
			uprofile.clearStrikes()
			logrus.WithFields(logrus.Fields{
				"ip":     conn.RemoteAddr().String(),
				"player": user,
			}).Info("Successful login")
			return user, true
		}

		// default: bad password, log and increment
		attempts++
		strikes := passwordStrikes[uprofile.Name]
		passwordStrikes[uprofile.Name] = strikes + 1
		conn.Write([]byte(invalidPass))
		logrus.WithFields(logrus.Fields{
			"ip":     conn.RemoteAddr().String(),
			"player": user,
			"error":  perr.Error(),
		}).Warn("Invalid password in login attempt")
	}

	timeoutUsers(&passwordStrikes, lsettings.LockoutCount)
	return user, false
}

func loadPlayer(player string) error {

	return nil
}

func pipeOutput(player string, conn net.Conn) {

}

func listenInput(player string, conn net.Conn) {

}

func Launch(conn net.Conn, lsettings *configparser.LoginSettings) {
	defer conn.Close()

	player, ok := validateLogin(conn, lsettings)
	if !ok {
		conn.Write([]byte("Closing connection..."))
		return
	}

	err := loadPlayer(player)
	if err != nil {
		conn.Write([]byte(err.Error()))
	}

	go pipeOutput(player, conn)

	listenInput(player, conn)
}
