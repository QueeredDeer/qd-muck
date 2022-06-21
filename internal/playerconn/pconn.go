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

// FIXME: consider using contexts to handle auto-closing of goroutines (see std lib)

import (
	"bufio"
	"context"
	"errors"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"github.com/QueeredDeer/qd-muck/internal/configparser"
	"github.com/QueeredDeer/qd-muck/internal/playerreg"
)

const invalidUser string = "Invalid login attempt"
const invalidPass string = "Invalid username or password"

var connParser *regexp.Regexp = regexp.MustCompile(`connect\s+(?P<user>\S+)\s+(?P<password>\S.*)`)
var uindex int = connParser.SubexpIndex("user")
var pindex int = connParser.SubexpIndex("password")

type userProfile struct {
	Name         string    `bson:"name"`
	PasswordHash []byte    `bson:"password_hash"`
	LoginStrikes int       `bson:"login_strikes"`
	Timeout      time.Time `bson:"timeout"`
}

// FIXME: this module should probably maintain state, e.g. login settings and MongoDB connections

func (profile *userProfile) clearStrikes() {
	// FIXME: remove magic strings
	profile.LoginStrikes = 0
	// must use UTC here because mongodb doesn't understand timezones via structs
	profile.Timeout = time.Unix(0, 0).UTC()

	uri := os.Getenv("MONGO_URI")
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		logrus.Error("failed to create mongodb client")
		return
	}

	err = client.Connect(context.TODO())
	if err != nil {
		logrus.Error("failed to connect to mongodb")
		return
	}

	defer client.Disconnect(context.TODO())

	// TODO: need to test this
	userCollection := client.Database("userprofiles").Collection("players")
	filter := bson.D{{Key: "name", Value: profile.Name}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "login_strikes", Value: profile.LoginStrikes}, {Key: "timeout", Value: profile.Timeout}}}}
	_, uerr := userCollection.UpdateOne(context.TODO(), filter, update)
	if uerr != nil {
		logrus.Error("failed to update profile for user '" + profile.Name + "'")
		return
	}
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

func userOffline(user string, preg *playerreg.ActivePlayerRegistry) bool {
	rchan := make(chan bool)
	prcallback := playerreg.RegistryCallback{
		PlayerName: user,
		Callback:   rchan,
	}

	preg.QueryPlayer <- &prcallback

	status := <-prcallback.Callback
	return status
}

func loadProfile(user string) (*userProfile, error) {
	// FIXME: remove magic strings
	uri := os.Getenv("MONGO_URI")
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		// TODO: find better way to handle system errors, instead of penalizing logins
		return &userProfile{}, errors.New("failed to create mongodb client")
	}

	err = client.Connect(context.TODO())
	if err != nil {
		// TODO: find better way to handle system errors, instead of penalizing logins
		return &userProfile{}, errors.New("failed to connect to mongodb")
	}

	defer client.Disconnect(context.TODO())

	userCollection := client.Database("userprofiles").Collection("players")

	var lookup userProfile
	lerr := userCollection.FindOne(context.TODO(), bson.D{{Key: "name", Value: user}}).Decode(&lookup)
	if lerr != nil {
		if lerr == mongo.ErrNoDocuments {
			// here we should fail on bad username
			return &userProfile{}, errors.New("user '" + user + "' does not exist")
		}
		// TODO: find better way to handle system errors, instead of penalizing logins
		return &userProfile{}, lerr
	}

	return &lookup, nil
}

func checkUser(user string, preg *playerreg.ActivePlayerRegistry) (*userProfile, error) {
	// check user exists in DB, user is not currently logged in, and not timed out
	offline := userOffline(user, preg)
	if !offline {
		return &userProfile{}, errors.New("user '" + user + "' is currently logged in")
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

	// iterate over map, loading profile data and incrementing strike count
	// if total count is > timeout threshold, update timeout clock
	// save all changes to DB

}

func validateLogin(conn net.Conn, lsettings *configparser.LoginSettings, preg *playerreg.ActivePlayerRegistry) (string, bool) {
	user := ""
	attempts := 0
	passwordStrikes := make(map[string]int)

	for attempts < lsettings.LoginAttempts {
		user, password := listenLogin(conn)

		uprofile, uerr := checkUser(user, preg)
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

	return errors.New("loadPlayer not implemented")
}

func pipeOutput(player string, conn net.Conn) {

}

func listenInput(player string, conn net.Conn) {

}

func Launch(conn net.Conn, lsettings *configparser.LoginSettings, preg *playerreg.ActivePlayerRegistry) {
	defer conn.Close()

	player, ok := validateLogin(conn, lsettings, preg)
	if !ok {
		conn.Write([]byte("Closing connection..."))
		return
	}

	err := loadPlayer(player)
	if err != nil {
		conn.Write([]byte(err.Error()))
		logrus.WithFields(logrus.Fields{
			"ip":     conn.RemoteAddr().String(),
			"player": player,
			"error":  err.Error(),
		}).Error("failed to load player data")
		return
	}

	go pipeOutput(player, conn)

	listenInput(player, conn)
}
