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
 * Prototypes witch are shared between the client, the server and the
 * game. This is the main game API, changes here will most likely
 * requiere changes to the game ddl.
 *
 * =======================================================================
 */
package shared

import (
	"math"
	"strings"
)

const (
	YQ2VERSION  = "8.00pre"
	BASEDIRNAME = "baseq2"

	/* PROTOCOL */

	PROTOCOL_VERSION = 34

	/* ========================================= */

	PORT_MASTER = 27900
	PORT_CLIENT = 27901
	PORT_SERVER = 27910

	/* ========================================= */

	UPDATE_BACKUP = 16 /* copies of entity_state_t to keep buffered */
	UPDATE_MASK   = (UPDATE_BACKUP - 1)
)

const (
	PORT_ANY      = -1
	MAX_MSGLEN    = 1400 /* max length of a message */
	PACKET_HEADER = 10   /* two ints and a short */

	/* server to client */
	SvcBad = 0

	/* these ops are known to the game dll */
	SvcMuzzleflash  = 1
	SvcMuzzleflash2 = 2
	SvcTempEntity   = 3
	SvcLayout       = 4
	SvcInventory    = 5

	/* the rest are private to the client and server */
	SvcNop                 = 5
	SvcDisconnect          = 7
	SvcReconnect           = 8
	SvcSound               = 9  /* <see code> */
	SvcPrint               = 10 /* [byte] id [string] null terminated string */
	SvcStufftext           = 11 /* [string] stuffed into client's console buffer, should be \n terminated */
	SvcServerdata          = 12 /* [long] protocol ... */
	SvcConfigstring        = 13 /* [short] [string] */
	SvcSpawnbaseline       = 14
	SvcCenterprint         = 15 /* [string] to put in center of the screen */
	SvcDownload            = 16 /* [short] size [size bytes] */
	SvcPlayerinfo          = 17 /* variable */
	SvcPacketentities      = 18 /* [...] */
	SvcDeltapacketentities = 19 /* [...] */
	SvcFrame               = 20

	/* ============================================== */

	/* client to server */
	ClcBad       = 0
	ClcNop       = 1
	ClcMove      = 2 /* [[usercmd_t] */
	ClcUserinfo  = 3 /* [[userinfo string] */
	ClcStringcmd = 4 /* [string] message */

	/* ============================================== */

	/* plyer_state_t communication */
	PS_M_TYPE         = (1 << 0)
	PS_M_ORIGIN       = (1 << 1)
	PS_M_VELOCITY     = (1 << 2)
	PS_M_TIME         = (1 << 3)
	PS_M_FLAGS        = (1 << 4)
	PS_M_GRAVITY      = (1 << 5)
	PS_M_DELTA_ANGLES = (1 << 6)

	PS_VIEWOFFSET  = (1 << 7)
	PS_VIEWANGLES  = (1 << 8)
	PS_KICKANGLES  = (1 << 9)
	PS_BLEND       = (1 << 10)
	PS_FOV         = (1 << 11)
	PS_WEAPONINDEX = (1 << 12)
	PS_WEAPONFRAME = (1 << 13)
	PS_RDFLAGS     = (1 << 14)

	/*============================================== */

	/* user_cmd_t communication */

	/* ms and light always sent, the others are optional */
	CM_ANGLE1  = (1 << 0)
	CM_ANGLE2  = (1 << 1)
	CM_ANGLE3  = (1 << 2)
	CM_FORWARD = (1 << 3)
	CM_SIDE    = (1 << 4)
	CM_UP      = (1 << 5)
	CM_BUTTONS = (1 << 6)
	CM_IMPULSE = (1 << 7)

	/*============================================== */

	/* a sound without an ent or pos will be a local only sound */
	SND_VOLUME      = (1 << 0) /* a byte */
	SND_ATTENUATION = (1 << 1) /* a byte */
	SND_POS         = (1 << 2) /* three coordinates */
	SND_ENT         = (1 << 3) /* a short 0-2: channel, 3-12: entity */
	SND_OFFSET      = (1 << 4) /* a byte, msec offset from frame start */

	DEFAULT_SOUND_PACKET_VOLUME      = 1.0
	DEFAULT_SOUND_PACKET_ATTENUATION = 1.0

	/*============================================== */

	/* entity_state_t communication */

	/* try to pack the common update flags into the first byte */
	U_ORIGIN1   = (1 << 0)
	U_ORIGIN2   = (1 << 1)
	U_ANGLE2    = (1 << 2)
	U_ANGLE3    = (1 << 3)
	U_FRAME8    = (1 << 4) /* frame is a byte */
	U_EVENT     = (1 << 5)
	U_REMOVE    = (1 << 6) /* REMOVE this entity, don't add it */
	U_MOREBITS1 = (1 << 7) /* read one additional byte */

	/* second byte */
	U_NUMBER16  = (1 << 8) /* NUMBER8 is implicit if not set */
	U_ORIGIN3   = (1 << 9)
	U_ANGLE1    = (1 << 10)
	U_MODEL     = (1 << 11)
	U_RENDERFX8 = (1 << 12) /* fullbright, etc */
	U_EFFECTS8  = (1 << 14) /* autorotate, trails, etc */
	U_MOREBITS2 = (1 << 15) /* read one additional byte */

	/* third byte */
	U_SKIN8      = (1 << 16)
	U_FRAME16    = (1 << 17) /* frame is a short */
	U_RENDERFX16 = (1 << 18) /* 8 + 16 = 32 */
	U_EFFECTS16  = (1 << 19) /* 8 + 16 = 32 */
	U_MODEL2     = (1 << 20) /* weapons, flags, etc */
	U_MODEL3     = (1 << 21)
	U_MODEL4     = (1 << 22)
	U_MOREBITS3  = (1 << 23) /* read one additional byte */

	/* fourth byte */
	U_OLDORIGIN = (1 << 24)
	U_SKIN16    = (1 << 25)
	U_SOUND     = (1 << 26)
	U_SOLID     = (1 << 27)
)

func ReadInt32(b []byte) int32 {
	return int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
}

func ReadUint32(b []byte) uint32 {
	return (uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24) & 0xFFFFFFFF
}

func ReadFloat32(b []byte) float32 {
	d := ReadUint32(b)
	return math.Float32frombits(d)
}

func ReadInt16(b []byte) int16 {
	return int16(uint16(b[0]) | uint16(b[1])<<8)
}

func ReadUint16(b []byte) uint16 {
	return (uint16(b[0]) | uint16(b[1])<<8) & 0xFFFF
}

func ReadString(b []byte, maxLen int) string {
	var r strings.Builder

	for i := 0; i < maxLen && b[i] != 0; i++ {
		r.WriteByte(b[i])
	}

	return r.String()
}
