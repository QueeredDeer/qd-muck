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

package configparser

import (
	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
)

type TomlConfig struct {
	Server  ServerSettings `toml:"Server"`
	Logging LogSettings    `toml:"Logging"`
	Login   LoginSettings  `toml:"Login"`
}

func (t *TomlConfig) LogConfiguration() {
	t.Server.Log()
	t.Logging.Log()
	t.Login.Log()
}

func (t *TomlConfig) setDefaults() {
	t.Server.setDefaults()
	t.Logging.setDefaults()
	t.Login.setDefaults()
}

type ServerSettings struct {
	CertFile    string `toml:"certificate_file"`
	PrivKeyFile string `toml:"private_key_file"`
	SslPort     int    `toml:"ssl_port"`
}

func (s *ServerSettings) Log() {
	logrus.WithFields(logrus.Fields{
		"certificate_file": s.CertFile,
		"private_key_file": s.PrivKeyFile,
		"ssl_port":         s.SslPort,
	}).Info("Server settings")
}

func (s *ServerSettings) setDefaults() {
	s.CertFile = ""
	s.PrivKeyFile = ""
	s.SslPort = 443
}

type LogSettings struct {
	LogFile string `toml:"log_file"`
}

func (l *LogSettings) Log() {
	// nothing for now, don't need to log name of log file in log file itself...
}

func (l *LogSettings) setDefaults() {
	l.LogFile = "log.server"
}

type LoginSettings struct {
	LoginAttempts int `toml:"login_attempts"`
	LockoutCount  int `toml:"lockout_count"`
	DbTimeout     int `toml:"database_timeout"`
}

func (l *LoginSettings) Log() {
	logrus.WithFields(logrus.Fields{
		"login_attempts":   l.LoginAttempts,
		"lockout_count":    l.LockoutCount,
		"database_timeout": l.DbTimeout,
	}).Info("Login settings")
}

func (l *LoginSettings) setDefaults() {
	l.LoginAttempts = 5
	l.LockoutCount = 5
	l.DbTimeout = 10
}

func ReadConfig(conf string) TomlConfig {
	var tconfig TomlConfig
	tconfig.setDefaults()
	_, err := toml.DecodeFile(conf, &tconfig)
	if err != nil {
		logrus.Fatal("Could not parse config file '" + conf + "'")
	}

	return tconfig
}
