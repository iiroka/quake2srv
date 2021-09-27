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
 * Message sending and multiplexing.
 *
 * =======================================================================
 */
package server

import (
	"fmt"
	"log"
	"quake2srv/shared"
)

func (T *qServer) svSendClientDatagram(client *client_t) bool {
	// byte msg_buf[MAX_MSGLEN];
	// sizebuf_t msg;

	T.svBuildClientFrame(client)

	msg := shared.QWritebufCreate(shared.MAX_MSGLEN)
	msg.Allowoverflow = true

	/* send over all the relevant entity_state_t
	   and the player_state_t */
	T.svWriteFrameToClient(client, msg)

	/* copy the accumulated multicast datagram
	   for this client out to the message
	   it is necessary for this to be after the WriteEntities
	   so that entity references will be current */
	if client.datagram.Overflowed {
		log.Printf("WARNING: datagram overflowed for %s\n", client.name)
	} else {
		msg.Write(client.datagram.Data())
	}

	client.datagram.Clear()

	if msg.Overflowed {
		/* must have room left for the packet header */
		log.Printf("WARNING: msg overflowed for %s\n", client.name)
		msg.Clear()
	}

	/* send the datagram */
	client.netchan.Transmit(msg.Data())

	/* record the size for rate estimation */
	// client.message_size[sv.framenum%RATE_MESSAGES] = msg.cursize

	return true
}

/*
 * Sends text to all active clients
 */
func (T *qServer) svBroadcastCommand(format string, a ...interface{}) {

	if T.sv.state == ss_dead {
		return
	}

	str := fmt.Sprintf(format, a...)

	T.sv.multicast.WriteByte(shared.SvcStufftext)
	T.sv.multicast.WriteString(str)
	T.svMulticast(nil, shared.MULTICAST_ALL_R)
}

/*
 * Sends the contents of sv.multicast to a subset of the clients,
 * then clears sv.multicast.
 *
 * MULTICAST_ALL	same as broadcast (origin can be NULL)
 * MULTICAST_PVS	send to clients potentially visible from org
 * MULTICAST_PHS	send to clients potentially hearable from org
 */
func (T *qServer) svMulticast(origin []float32, to shared.Multicast_t) {
	//  client_t *client;
	//  byte *mask;
	//  int leafnum = 0, cluster;
	//  int j;
	//  qboolean reliable;
	//  int area1, area2;

	reliable := false

	area1 := 0
	if (to != shared.MULTICAST_ALL_R) && (to != shared.MULTICAST_ALL) {
		leafnum := T.common.CMPointLeafnum(origin)
		area1 = T.common.CMLeafArea(leafnum)
	}

	/* if doing a serverrecord, store everything */
	//  if (svs.demofile) {
	// 	 SZ_Write(&svs.demo_multicast, sv.multicast.data, sv.multicast.cursize);
	//  }

	var mask []byte = nil
	switch to {
	case shared.MULTICAST_ALL_R:
		reliable = true /* intentional fallthrough */
		fallthrough
	case shared.MULTICAST_ALL:
		break

	case shared.MULTICAST_PHS_R:
		reliable = true /* intentional fallthrough */
		fallthrough
	case shared.MULTICAST_PHS:
		leafnum := T.common.CMPointLeafnum(origin)
		cluster := T.common.CMLeafCluster(leafnum)
		mask = T.common.CMClusterPHS(cluster)
		break

	case shared.MULTICAST_PVS_R:
		reliable = true /* intentional fallthrough */
		fallthrough
	case shared.MULTICAST_PVS:
		leafnum := T.common.CMPointLeafnum(origin)
		cluster := T.common.CMLeafCluster(leafnum)
		mask = T.common.CMClusterPVS(cluster)
		break

	default:
		// mask = NULL
		log.Fatalf("SV_Multicast: bad to:%v", to)
	}

	/* send the data to all relevent clients */
	for j, client := range T.svs.clients {
		if (client.state == cs_free) || (client.state == cs_zombie) {
			continue
		}

		if (client.state != cs_spawned) && !reliable {
			continue
		}

		if mask != nil {
			leafnum := T.common.CMPointLeafnum(client.edict.S().Origin[:])
			cluster := T.common.CMLeafCluster(leafnum)
			area2 := T.common.CMLeafArea(leafnum)

			if !T.common.CMAreasConnected(area1, area2) {
				continue
			}

			if (mask[cluster>>3] & (1 << (cluster & 7))) == 0 {
				continue
			}
		}

		if reliable {
			T.svs.clients[j].netchan.Message.Write(T.sv.multicast.Data())
		} else {
			T.svs.clients[j].datagram.Write(T.sv.multicast.Data())
		}
	}

	T.sv.multicast.Clear()
}

func (T *qServer) svSendClientMessages() {

	var msgbuf []byte = nil

	/* read the next demo message if needed */
	// if T.sv.demofile != nil && (T.sv.state == ss_demo) {
	// 	if !T.sv_paused.Bool() {
	// 		/* get the next message */
	// 		bfr := T.sv.demofile.Read(4)
	// 		if len(bfr) != 4 {
	// 			T.svDemoCompleted()
	// 			return
	// 		}

	// 		msglen := int(shared.ReadInt32(bfr))
	// 		if msglen == -1 {
	// 			T.svDemoCompleted()
	// 			return
	// 		}

	// 		if msglen > shared.MAX_MSGLEN {
	// 			T.common.Com_Error(shared.ERR_DROP,
	// 				"SV_SendClientMessages: msglen > MAX_MSGLEN")
	// 		}

	// 		msgbuf = T.sv.demofile.Read(msglen)
	// 		if len(msgbuf) != msglen {
	// 			T.svDemoCompleted()
	// 			return
	// 		}
	// 	}
	// }

	/* send a message to each connected client */
	for i, c := range T.svs.clients {
		if c.state == cs_free {
			continue
		}

		/* if the reliable message
		   overflowed, drop the
		   client */
		if c.netchan.Message.Overflowed {
			T.svs.clients[i].netchan.Message.Clear()
			T.svs.clients[i].datagram.Clear()
			// SV_BroadcastPrintf(PRINT_HIGH, "%s overflowed\n", c->name);
			// 		SV_DropClient(c);
		}

		if (T.sv.state == ss_cinematic) ||
			(T.sv.state == ss_demo) ||
			(T.sv.state == ss_pic) {
			T.svs.clients[i].netchan.Transmit(msgbuf)
		} else if c.state == cs_spawned {
			// 		/* don't overrun bandwidth */
			// 		if (SV_RateDrop(c)) {
			// 			continue;
			// 		}

			T.svSendClientDatagram(&T.svs.clients[i])
		} else {
			/* just update reliable	if needed */
			if c.netchan.Message.Cursize > 0 || (T.common.Curtime()-c.netchan.LastSent) > 1000 {
				T.svs.clients[i].netchan.Transmit(msgbuf)
			}
		}
	}
}
