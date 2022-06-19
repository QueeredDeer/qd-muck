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

package main

import (
	"flag"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/QueeredDeer/qd-muck/internal/configparser"
)

func ConfigureLogging(f *os.File) {
	logrus.SetOutput(f)
}

func ValidateEnvironment() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		logrus.Fatal("MONGO_URI must be set in the environment")
	}
}

func main() {
	configFilePtr := flag.String("conf", "config.toml",
		"Server configuration file (TOML)")

	flag.Parse()

	configSettings := configparser.ReadConfig(*configFilePtr)

	logFile := configSettings.Logging.LogFile
	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logrus.Fatal("Could not open log file '" + logFile + "'")
	}
	defer file.Close()

	ConfigureLogging(file)
	configSettings.LogConfiguration()

	logrus.Info("Hello from the QD MUCK server!")
}
