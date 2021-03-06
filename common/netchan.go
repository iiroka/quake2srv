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
 * The low level, platform independant network code
 *
 * =======================================================================
 */
package common

import (
	"fmt"
	"quake2srv/shared"
)

/*
 * packet header
 * -------------
 * 31	sequence
 * 1	does this message contain a reliable payload
 * 31	acknowledge sequence
 * 1	acknowledge receipt of even/odd message
 * 16	qport
 *
 * The remote connection never knows if it missed a reliable message,
 * the local side detects that it has been dropped by seeing a sequence
 * acknowledge higher thatn the last reliable sequence, but without the
 * correct even/odd bit for the reliable set.
 *
 * If the sender notices that a reliable message has been dropped, it
 * will be retransmitted.  It will not be retransmitted again until a
 * message after the retransmit has been acknowledged and the reliable
 * still failed to get there.
 *
 * if the sequence number is -1, the packet should be handled without a
 * netcon
 *
 * The reliable message can be added to at any time by doing MSG_Write*
 * (&netchan->message, <data>).
 *
 * If the message buffer is overflowed, either by a single message, or
 * by multiple frames worth piling up while the last reliable transmit
 * goes unacknowledged, the netchan signals a fatal error.
 *
 * Reliable messages are always placed first in a packet, then the
 * unreliable message is included if there is sufficient room.
 *
 * To the receiver, there is no distinction between the reliable and
 * unreliable parts of the message, they are just processed out as a
 * single larger message.
 *
 * Illogical packet sequence numbers cause the packet to be dropped, but
 * do not kill the connection.  This, combined with the tight window of
 * valid reliable acknowledgement numbers provides protection against
 * malicious address spoofing.
 *
 * The qport field is a workaround for bad address translating routers
 * that sometimes remap the client's source port on a packet during
 * gameplay.
 *
 * If the base part of the net address matches and the qport matches,
 * then the channel matches even if the IP port differs.  The IP port
 * should be updated to the new value before sending out any replies.
 *
 * If there is no information that needs to be transfered on a given
 * frame, such as during the connection stage while waiting for the
 * client to load, then a packet only needs to be delivered if there is
 * something in the unacknowledged reliable
 */

func (T *qCommon) netchanInit() {

	/* This is a little bit fishy:

	The original code used Sys_Milliseconds() as base. It worked
	because the original Sys_Milliseconds included some amount of
	random data (Windows) or was dependend on seconds since epoche
	(Unix). Our Sys_Milliseconds() always starts at 0, so there's a
	very high propability - nearly 100 percent for something like
	`./quake2 +connect example.com - that two or more clients end up
	with the same qport.

	We can't use rand() because we're always starting with the same
	seed. So right after client start we'll nearly always get the
	same random numbers. Again there's a high propability that two or
	more clients end up with the same qport.

	Just calling time() should be portable and is more less what
	Windows did in the original code. There's still a rather small
	propability that two clients end up with the same qport, but
	that needs to fixed somewhere else with some kind of fallback
	logic. */
	// port := time.Now().Nanosecond() & 0xffff

	// T.showpackets = T.Cvar_Get("showpackets", "0", 0)
	// T.showdrop = T.Cvar_Get("showdrop", "0", 0)
	// T.qport = T.Cvar_Get("qport", fmt.Sprintf("%v", port), shared.CVAR_NOSET)
}

/*
 * Sends an out-of-band datagram
 */
func (T *qCommon) Netchan_OutOfBand(adr string, data []byte) error {

	/* write the packet header */
	send := shared.QWritebufCreate(shared.MAX_MSGLEN)

	send.WriteLong(-1) /* -1 sequence means out of band */
	send.Write(data)

	/* send the datagram */
	T.NET_SendPacket(send.Data(), adr)
	return nil
}

/*
 * Sends a text message in an out-of-band datagram
 */
func (T *qCommon) Netchan_OutOfBandPrint(adr string, format string, a ...interface{}) error {
	str := fmt.Sprintf(format, a...)
	return T.Netchan_OutOfBand(adr, []byte(str))
}

// func (T *qCommon) QPort() int {
// 	return T.qport.Int()
// }
