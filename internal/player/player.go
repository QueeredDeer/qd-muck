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

package player

import (
	"net"

	"github.com/sirupsen/logrus"
)

type Player struct {
	Name     string
	MsgQueue chan string
	EndChan  chan int
	JsonBlob string
}

func New(name string, blob string) *Player {
	player := Player{
		Name:     name,
		MsgQueue: make(chan string),
		EndChan:  make(chan int),
		JsonBlob: blob,
	}

	return &player
}

func (p *Player) write(msg string, conn net.Conn) {
	_, werr := conn.Write([]byte(msg))
	if werr != nil {
		logrus.WithFields(logrus.Fields{
			"ip":     conn.RemoteAddr().String(),
			"player": p.Name,
			"error":  werr.Error(),
			"msg":    msg,
		}).Warn("Could not write message out to connection")
	}
}

func (p *Player) WriteFromChannel(conn net.Conn) {
	for {
		select {
		case msg := <-p.MsgQueue:
			p.write(msg, conn)
		case <-p.EndChan:
			return
		}
	}
}
