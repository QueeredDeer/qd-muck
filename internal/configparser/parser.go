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
}

type ServerSettings struct {
	CertFile    string `toml:"certificate_file"`
	PrivKeyFile string `toml:"private_key_file"`
	SslPort     int    `toml:"ssl_port"`
}

type LogSettings struct {
	LogFile string `toml:"log_file"`
}

func ReadConfig(conf string) TomlConfig {
	var tconfig TomlConfig
	_, err := toml.DecodeFile(conf, &tconfig)
	if err != nil {
		logrus.Fatal("Could not parse config file '" + conf + "'")
	}

	return tconfig
}
