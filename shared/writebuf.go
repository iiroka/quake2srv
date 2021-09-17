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
 * Movement message (forward, backward, left, right, etc) handling.
 *
 * =======================================================================
 */
package shared

import "log"

type QWritebuf struct {
	Allowoverflow bool /* if false, do a Com_Error */
	Overflowed    bool /* set to true if the buffer size failed */
	data          []byte
	Cursize       int
}

func QWritebufCreate(size int) *QWritebuf {
	wb := &QWritebuf{}
	wb.data = make([]byte, size)
	return wb
}

func (buf *QWritebuf) Data() []byte {
	return buf.data[:buf.Cursize]
}

func (buf *QWritebuf) Clear() {
	buf.Cursize = 0
	buf.Overflowed = false
}

func (buf *QWritebuf) getSpace(length int) []byte {

	if buf.Cursize+length > len(buf.data) {
		if !buf.Allowoverflow {
			log.Fatal("SZ_GetSpace: overflow without allowoverflow set")
		}

		if length > len(buf.data) {
			log.Fatalf("SZ_GetSpace: %v is > full buffer size", length)
		}

		buf.Clear()
		buf.Overflowed = true
		println("SZ_GetSpace: overflow\n")
	}

	data := buf.data[buf.Cursize:]
	buf.Cursize += length

	return data
}

func (sb *QWritebuf) WriteChar(c int) {

	buf := sb.getSpace(1)
	buf[0] = byte(c)
}

func (sb *QWritebuf) WriteByte(c int) {

	buf := sb.getSpace(1)
	buf[0] = byte(c & 0xFF)
}

func (sb *QWritebuf) WriteLong(c int) {

	buf := sb.getSpace(4)
	buf[0] = byte(c & 0xff)
	buf[1] = byte((c >> 8) & 0xff)
	buf[2] = byte((c >> 16) & 0xff)
	buf[3] = byte(c >> 24)
}

func (sb *QWritebuf) WriteShort(c int) {

	buf := sb.getSpace(2)
	buf[0] = byte(c & 0xff)
	buf[1] = byte(c >> 8)
}

func (sb *QWritebuf) Write(data []byte) {
	buf := sb.getSpace(len(data))
	copy(buf, data)
}

func (sb *QWritebuf) WriteString(s string) {
	if len(s) == 0 {
		sb.WriteChar(0)
	} else {
		sb.Write([]byte(s))
		sb.WriteChar(0)
	}
}

func (sb *QWritebuf) Print(s string) {

	if sb.Cursize > 0 {
		if sb.data[sb.Cursize-1] != 0 {
			sb.Write([]byte(s))
		} else {
			sb.Cursize--
			sb.Write([]byte(s)) /* write over trailing 0 */
		}
	} else {
		sb.Write([]byte(s))
	}
	sb.WriteChar(0)
}

func (sb *QWritebuf) WriteCoord(f float32) {
	sb.WriteShort(int(f * 8))
}

func (sb *QWritebuf) WriteAngle(f float32) {
	sb.WriteByte(int(f*256/360) & 255)
}

func (sb *QWritebuf) WriteAngle16(f float32) {
	sb.WriteShort(int(ANGLE2SHORT(f)))
}

func (sb *QWritebuf) WriteDeltaUsercmd(from, cmd *Usercmd_t) {

	/* Movement messages */
	bits := 0

	if cmd.Angles[0] != from.Angles[0] {
		bits |= CM_ANGLE1
	}

	if cmd.Angles[1] != from.Angles[1] {
		bits |= CM_ANGLE2
	}

	if cmd.Angles[2] != from.Angles[2] {
		bits |= CM_ANGLE3
	}

	if cmd.Forwardmove != from.Forwardmove {
		bits |= CM_FORWARD
	}

	if cmd.Sidemove != from.Sidemove {
		bits |= CM_SIDE
	}

	if cmd.Upmove != from.Upmove {
		bits |= CM_UP
	}

	if cmd.Buttons != from.Buttons {
		bits |= CM_BUTTONS
	}

	if cmd.Impulse != from.Impulse {
		bits |= CM_IMPULSE
	}

	sb.WriteByte(bits)

	if (bits & CM_ANGLE1) != 0 {
		sb.WriteShort(int(cmd.Angles[0]))
	}

	if (bits & CM_ANGLE2) != 0 {
		sb.WriteShort(int(cmd.Angles[1]))
	}

	if (bits & CM_ANGLE3) != 0 {
		sb.WriteShort(int(cmd.Angles[2]))
	}

	if (bits & CM_FORWARD) != 0 {
		sb.WriteShort(int(cmd.Forwardmove))
	}

	if (bits & CM_SIDE) != 0 {
		sb.WriteShort(int(cmd.Sidemove))
	}

	if (bits & CM_UP) != 0 {
		sb.WriteShort(int(cmd.Upmove))
	}

	if (bits & CM_BUTTONS) != 0 {
		sb.WriteByte(int(cmd.Buttons))
	}

	if (bits & CM_IMPULSE) != 0 {
		sb.WriteByte(int(cmd.Impulse))
	}

	sb.WriteByte(int(cmd.Msec))
	sb.WriteByte(int(cmd.Lightlevel))
}

/*
 * Writes part of a packetentities message.
 * Can delta from either a baseline or a previous packet_entity
 */
func (sb *QWritebuf) WriteDeltaEntity(from, to *Entity_state_t,
	force, newentity bool) {

	if to.Number == 0 {
		log.Fatal("Unset entity number")
	}

	if to.Number >= MAX_EDICTS {
		log.Fatal("Entity number >= MAX_EDICTS")
	}

	/* send an update */
	bits := 0

	if to.Number >= 256 {
		bits |= U_NUMBER16 /* number8 is implicit otherwise */
	}

	if to.Origin[0] != from.Origin[0] {
		bits |= U_ORIGIN1
	}

	if to.Origin[1] != from.Origin[1] {
		bits |= U_ORIGIN2
	}

	if to.Origin[2] != from.Origin[2] {
		bits |= U_ORIGIN3
	}

	if to.Angles[0] != from.Angles[0] {
		bits |= U_ANGLE1
	}

	if to.Angles[1] != from.Angles[1] {
		bits |= U_ANGLE2
	}

	if to.Angles[2] != from.Angles[2] {
		bits |= U_ANGLE3
	}

	if to.Skinnum != from.Skinnum {
		if uint(to.Skinnum) < 256 {
			bits |= U_SKIN8
		} else if uint(to.Skinnum) < 0x10000 {
			bits |= U_SKIN16
		} else {
			bits |= (U_SKIN8 | U_SKIN16)
		}
	}

	if to.Frame != from.Frame {
		if to.Frame < 256 {
			bits |= U_FRAME8
		} else {
			bits |= U_FRAME16
		}
	}

	if to.Effects != from.Effects {
		if to.Effects < 256 {
			bits |= U_EFFECTS8
		} else if to.Effects < 0x8000 {
			bits |= U_EFFECTS16
		} else {
			bits |= U_EFFECTS8 | U_EFFECTS16
		}
	}

	if to.Renderfx != from.Renderfx {
		if to.Renderfx < 256 {
			bits |= U_RENDERFX8
		} else if to.Renderfx < 0x8000 {
			bits |= U_RENDERFX16
		} else {
			bits |= U_RENDERFX8 | U_RENDERFX16
		}
	}

	if to.Solid != from.Solid {
		bits |= U_SOLID
	}

	/* event is not delta compressed, just 0 compressed */
	if to.Event != 0 {
		bits |= U_EVENT
	}

	if to.Modelindex != from.Modelindex {
		bits |= U_MODEL
	}

	if to.Modelindex2 != from.Modelindex2 {
		bits |= U_MODEL2
	}

	if to.Modelindex3 != from.Modelindex3 {
		bits |= U_MODEL3
	}

	if to.Modelindex4 != from.Modelindex4 {
		bits |= U_MODEL4
	}

	if to.Sound != from.Sound {
		bits |= U_SOUND
	}

	if newentity || (to.Renderfx&RF_BEAM) != 0 {
		bits |= U_OLDORIGIN
	}

	/* write the message */
	if bits == 0 && !force {
		return /* nothing to send! */
	}

	if (bits & 0xff000000) != 0 {
		bits |= U_MOREBITS3 | U_MOREBITS2 | U_MOREBITS1
	} else if (bits & 0x00ff0000) != 0 {
		bits |= U_MOREBITS2 | U_MOREBITS1
	} else if (bits & 0x0000ff00) != 0 {
		bits |= U_MOREBITS1
	}

	sb.WriteByte(bits & 255)

	if (bits & 0xff000000) != 0 {
		sb.WriteByte((bits >> 8) & 255)
		sb.WriteByte((bits >> 16) & 255)
		sb.WriteByte((bits >> 24) & 255)
	} else if (bits & 0x00ff0000) != 0 {
		sb.WriteByte((bits >> 8) & 255)
		sb.WriteByte((bits >> 16) & 255)
	} else if (bits & 0x0000ff00) != 0 {
		sb.WriteByte((bits >> 8) & 255)
	}

	if (bits & U_NUMBER16) != 0 {
		sb.WriteShort(to.Number)
	} else {
		sb.WriteByte(to.Number)
	}

	if (bits & U_MODEL) != 0 {
		sb.WriteByte(to.Modelindex)
	}

	if (bits & U_MODEL2) != 0 {
		sb.WriteByte(to.Modelindex2)
	}

	if (bits & U_MODEL3) != 0 {
		sb.WriteByte(to.Modelindex3)
	}

	if (bits & U_MODEL4) != 0 {
		sb.WriteByte(to.Modelindex4)
	}

	if (bits & U_FRAME8) != 0 {
		sb.WriteByte(to.Frame)
	}

	if (bits & U_FRAME16) != 0 {
		sb.WriteShort(to.Frame)
	}

	if (bits&U_SKIN8) != 0 && (bits&U_SKIN16) != 0 { /*used for laser colors */
		sb.WriteLong(to.Skinnum)
	} else if (bits & U_SKIN8) != 0 {
		sb.WriteByte(to.Skinnum)
	} else if (bits & U_SKIN16) != 0 {
		sb.WriteShort(to.Skinnum)
	}

	if (bits & (U_EFFECTS8 | U_EFFECTS16)) == (U_EFFECTS8 | U_EFFECTS16) {
		sb.WriteLong(int(to.Effects))
	} else if (bits & U_EFFECTS8) != 0 {
		sb.WriteByte(int(to.Effects))
	} else if (bits & U_EFFECTS16) != 0 {
		sb.WriteShort(int(to.Effects))
	}

	if (bits & (U_RENDERFX8 | U_RENDERFX16)) == (U_RENDERFX8 | U_RENDERFX16) {
		sb.WriteLong(to.Renderfx)
	} else if (bits & U_RENDERFX8) != 0 {
		sb.WriteByte(to.Renderfx)
	} else if (bits & U_RENDERFX16) != 0 {
		sb.WriteShort(to.Renderfx)
	}

	if (bits & U_ORIGIN1) != 0 {
		sb.WriteCoord(to.Origin[0])
	}

	if (bits & U_ORIGIN2) != 0 {
		sb.WriteCoord(to.Origin[1])
	}

	if (bits & U_ORIGIN3) != 0 {
		sb.WriteCoord(to.Origin[2])
	}

	if (bits & U_ANGLE1) != 0 {
		sb.WriteAngle(to.Angles[0])
	}

	if (bits & U_ANGLE2) != 0 {
		sb.WriteAngle(to.Angles[1])
	}

	if (bits & U_ANGLE3) != 0 {
		sb.WriteAngle(to.Angles[2])
	}

	if (bits & U_OLDORIGIN) != 0 {
		sb.WriteCoord(to.Old_origin[0])
		sb.WriteCoord(to.Old_origin[1])
		sb.WriteCoord(to.Old_origin[2])
	}

	if (bits & U_SOUND) != 0 {
		sb.WriteByte(to.Sound)
	}

	if (bits & U_EVENT) != 0 {
		sb.WriteByte(to.Event)
	}

	if (bits & U_SOLID) != 0 {
		sb.WriteShort(to.Solid)
	}
}
