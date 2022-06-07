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

	"github.com/QueeredDeer/qd-muck/internal/configparser"
)

const invalidUser string = "Invalid login attempt"
const invalidPass string = "Invalid username or password"

var connParser *regexp.Regexp = regexp.MustCompile(`connect\s+(?P<user>\S+)\s+(?P<password>\S.*)`)

type userProfile struct {
	Name         string
	PasswordHash string
	LoginStrikes int
	TimeoutSet   bool
	Timeout      time.Time
}

func (profile *userProfile) clearStrikes() {
	// FIXME: write these to db instead
	profile.LoginStrikes = 0
	profile.TimeoutSet = false
	profile.Timeout = time.Unix(0, 0)
}

func listenLogin(conn net.Conn) (string, string) {
	reader := bufio.NewReader(conn)

	uindex := connParser.SubexpIndex("user")
	pindex := connParser.SubexpIndex("password")

	// TODO: add timeout to this loop?

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Write([]byte(err.Error()))
			continue
		}

		cmd := strings.TrimSpace(string(line))

		if !connParser.MatchString(cmd) {
			conn.Write([]byte("Unrecognized command format"))
			continue
		}

		matches := connParser.FindStringSubmatch(cmd)
		user := matches[uindex]
		pass := matches[pindex]

		return user, pass
	}

	return "", ""
}

func checkUser(user string) (*userProfile, error) {
	// check user exists in DB, user is not currently logged in, and not timed out
	return &userProfile{}, errors.New("checkUser unimplemented")
}

func checkPassword(uprofile *userProfile, password string) error {
	return errors.New("checkPassword unimplemented")
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
			}).Warn("Invalid login attempt")
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
		}).Warn("Invalid login attempt")
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
