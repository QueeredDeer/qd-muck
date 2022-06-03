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
	"fmt"

	"github.com/QueeredDeer/qd-muck/internal/configparser"
)

func main() {
	configFilePtr := flag.String("conf", "config.toml", "Server configuration file (TOML)")

	flag.Parse()

	configSettings := configparser.ReadConfig(*configFilePtr)

	fmt.Println("Hello from the QD MUCK server!")

	fmt.Println("CertFile:", configSettings.Server.CertFile)
	fmt.Println("Private Key:", configSettings.Server.PrivKeyFile)
	fmt.Println("SSL port:", configSettings.Server.SslPort)

	fmt.Println("Log file:", configSettings.Logging.LogFile)
}
