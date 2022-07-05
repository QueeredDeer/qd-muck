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
	"github.com/QueeredDeer/qd-muck/internal/player"
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

type PlayerConn struct {
	Conn           net.Conn
	LoginSettings  *configparser.LoginSettings
	PlayerRegistry *playerreg.ActivePlayerRegistry
	DbClient       *mongo.Client
}

func New(conn net.Conn, lsettings *configparser.LoginSettings, preg *playerreg.ActivePlayerRegistry) (*PlayerConn, error) {
	p := PlayerConn{
		Conn:           conn,
		LoginSettings:  lsettings,
		PlayerRegistry: preg,
	}

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return &PlayerConn{}, errors.New("must set MONGO_URI in environment")
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return &PlayerConn{}, errors.New("failed to create mongodb client")
	}

	p.DbClient = client

	return &p, nil
}

// FIXME: this module should probably maintain state, e.g. login settings and MongoDB connections

func (pc *PlayerConn) clearStrikes(profile *userProfile) {
	// FIXME: remove magic strings
	profile.LoginStrikes = 0
	// must use UTC here because mongodb doesn't understand timezones via structs
	profile.Timeout = time.Unix(0, 0).UTC()

	timeout := time.Duration(pc.LoginSettings.DbTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := pc.DbClient.Connect(ctx)
	if err != nil {
		logrus.Error("failed to connect to mongodb")
		return
	}
	defer pc.DbClient.Disconnect(ctx)

	// TODO: need to test this
	userCollection := pc.DbClient.Database("userprofiles").Collection("players")
	filter := bson.D{{Key: "name", Value: profile.Name}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "login_strikes", Value: profile.LoginStrikes}, {Key: "timeout", Value: profile.Timeout}}}}
	_, uerr := userCollection.UpdateOne(ctx, filter, update)
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

func (pc *PlayerConn) userOffline(user string) bool {
	rchan := make(chan bool)
	prcallback := playerreg.RegistryCallback{
		PlayerName: user,
		Callback:   rchan,
	}

	pc.PlayerRegistry.QueryPlayer <- &prcallback

	status := <-prcallback.Callback
	return status
}

func (pc *PlayerConn) loadProfile(user string) (*userProfile, error) {

	timeout := time.Duration(pc.LoginSettings.DbTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := pc.DbClient.Connect(ctx)
	if err != nil {
		// TODO: find better way to handle system errors, instead of penalizing logins
		return &userProfile{}, errors.New("failed to connect to database")
	}
	defer pc.DbClient.Disconnect(ctx)

	userCollection := pc.DbClient.Database("userprofiles").Collection("players")

	var lookup userProfile
	lerr := userCollection.FindOne(ctx, bson.D{{Key: "name", Value: user}}).Decode(&lookup)
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

func (pc *PlayerConn) checkUser(user string) (*userProfile, error) {
	// check user exists in DB, user is not currently logged in, and not timed out
	offline := pc.userOffline(user)
	if !offline {
		return &userProfile{}, errors.New("user '" + user + "' is currently logged in")
	}

	profile, err := pc.loadProfile(user)
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

func (pc *PlayerConn) timeoutUsers(strikes *map[string]int) {
	// pc.LoginSettings.LockoutCount
	// FIXME: implement

	// iterate over map, loading profile data and incrementing strike count
	// if total count is > timeout threshold, update timeout clock
	// save all changes to DB

}

func (pc *PlayerConn) validateLogin() (string, bool) {
	user := ""
	attempts := 0
	passwordStrikes := make(map[string]int)

	for attempts < pc.LoginSettings.LoginAttempts {
		user, password := listenLogin(pc.Conn)

		uprofile, uerr := pc.checkUser(user)
		if uerr != nil {
			attempts++
			pc.Conn.Write([]byte(invalidUser))
			logrus.WithFields(logrus.Fields{
				"ip":     pc.Conn.RemoteAddr().String(),
				"player": user,
				"error":  uerr.Error(),
			}).Warn("Invalid username in login attempt")
			continue
		}

		// validate password
		perr := checkPassword(uprofile, password)
		if perr == nil {
			// password validated, safe to login
			pc.clearStrikes(uprofile)
			logrus.WithFields(logrus.Fields{
				"ip":     pc.Conn.RemoteAddr().String(),
				"player": user,
			}).Info("Successful login")
			return user, true
		}

		// default: bad password, log and increment
		attempts++
		strikes := passwordStrikes[uprofile.Name]
		passwordStrikes[uprofile.Name] = strikes + 1
		pc.Conn.Write([]byte(invalidPass))
		logrus.WithFields(logrus.Fields{
			"ip":     pc.Conn.RemoteAddr().String(),
			"player": user,
			"error":  perr.Error(),
		}).Warn("Invalid password in login attempt")
	}

	pc.timeoutUsers(&passwordStrikes)
	return user, false
}

func loadPlayer(name string) (*player.Player, error) {

	return nil, errors.New("loadPlayer not implemented")
}

func (pc *PlayerConn) listenInput() {

}

func (pc *PlayerConn) cleanup(player *player.Player) {
	// FIXME
	// remove player from active registry

	// close player DB connection

	// send kill signal to player listen loop
}

func (pc *PlayerConn) Launch() {
	defer pc.Conn.Close()

	name, ok := pc.validateLogin()
	if !ok {
		pc.Conn.Write([]byte("Closing connection..."))
		return
	}

	player, err := loadPlayer(name)
	if err != nil {
		pc.Conn.Write([]byte(err.Error()))
		logrus.WithFields(logrus.Fields{
			"ip":     pc.Conn.RemoteAddr().String(),
			"player": name,
			"error":  err.Error(),
		}).Error("failed to load player data")
		return
	}
	defer pc.cleanup(player)

	go player.WriteFromChannel(pc.Conn)

	pc.listenInput()
}
