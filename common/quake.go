/*
 * Copyright (C) 1997-2001 Id Software, Inc.
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or (at
 * your option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 * See the GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA
 * 02111-1307, USA.
 *
 * =======================================================================
 *
 * Platform independent initialization, main loop and frame handling.
 *
 * =======================================================================
 */
package common

import (
	"quake2srv/shared"
	"time"
)

type qNetClient struct {
	handler func([]byte, interface{})
	context interface{}
}

type qNetMsg struct {
	from string
	data []byte
}

type xcommand_t struct {
	function func([]string, interface{}) error
	param    interface{}
}

type qCommon struct {
	server          shared.QServer
	net_clients     map[string]qNetClient
	net_ch          chan qNetMsg
	net_disc        chan string
	running         bool
	curtime         int
	server_state    int
	startTime       time.Time
	servertimedelta int
	packetdelta     int
	args            []string

	recursive bool
	msg       string

	cvarVars         map[string]*shared.CvarT
	userinfoModified bool

	fs shared.QFileSystem

	cmd_text      string
	alias_count   int
	cmd_functions map[string]xcommand_t
	cmd_alias     map[string]string

	collision qCollision

	pm_stopspeed       float32
	pm_maxspeed        float32
	pm_duckspeed       float32
	pm_accelerate      float32
	pm_airaccelerate   float32
	pm_wateraccelerate float32
	pm_friction        float32
	pm_waterfriction   float32
	pm_waterspeed      float32
}

func (Q *qCommon) SetServer(srvr shared.QServer) {
	Q.server = srvr
}

func (Q *qCommon) LoadFile(path string) ([]byte, error) {
	return Q.fs.LoadFile(path)
}

func CreateQuekeCommon(fs shared.QFileSystem) shared.QCommon {
	q := &qCommon{}
	q.fs = fs
	q.servertimedelta = 0
	q.packetdelta = 1000000
	q.net_clients = make(map[string]qNetClient)
	q.net_ch = make(chan qNetMsg, 1024)
	q.net_disc = make(chan string, 10)
	q.cvarVars = make(map[string]*shared.CvarT)
	q.cmd_functions = make(map[string]xcommand_t)
	q.cmd_alias = make(map[string]string)
	q.pm_stopspeed = 100
	q.pm_maxspeed = 300
	q.pm_duckspeed = 100
	q.pm_accelerate = 10
	q.pm_airaccelerate = 0
	q.pm_wateraccelerate = 10
	q.pm_friction = 6
	q.pm_waterfriction = 1
	q.pm_waterspeed = 400
	return q
}
