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

import (
	"log"
	"strings"
)

type QReadbuf struct {
	data      []byte
	readcount int
}

func QReadbufCreate(data []byte) *QReadbuf {
	return &QReadbuf{data, 0}
}

func (msg *QReadbuf) Size() int {
	return len(msg.data)
}

func (msg *QReadbuf) Count() int {
	return msg.readcount
}

func (msg *QReadbuf) IsEmpty() bool {
	return msg.readcount >= len(msg.data)
}

func (msg *QReadbuf) IsOver() bool {
	return msg.readcount > len(msg.data)
}

func (msg *QReadbuf) BeginReading() {
	msg.readcount = 0
}

func (msg *QReadbuf) ReadByte() int {

	var c int
	if msg.readcount+1 > len(msg.data) {
		c = -1
	} else {
		c = int(uint8(msg.data[msg.readcount]))
	}
	msg.readcount += 1
	return c
}

func (msg *QReadbuf) ReadChar() int {

	var c int
	if msg.readcount+1 > len(msg.data) {
		c = -1
	} else {
		c = int(int8(uint8(msg.data[msg.readcount])))
	}
	msg.readcount += 1
	return c
}

func (msg *QReadbuf) ReadShort() int {

	var c int
	if msg.readcount+2 > len(msg.data) {
		c = -1
	} else {
		c = int(int16(uint32(msg.data[msg.readcount]) |
			(uint32(msg.data[msg.readcount+1]) << 8)))
	}
	msg.readcount += 2
	return c
}

func (msg *QReadbuf) ReadLong() int {

	var c int
	if msg.readcount+4 > len(msg.data) {
		c = -1
	} else {
		c = int(int32(uint32(msg.data[msg.readcount]) |
			(uint32(msg.data[msg.readcount+1]) << 8) |
			(uint32(msg.data[msg.readcount+2]) << 16) |
			(uint32(msg.data[msg.readcount+3]) << 24)))
	}
	msg.readcount += 4
	return c
}

func (msg *QReadbuf) ReadString() string {

	var r strings.Builder
	for {
		c := msg.ReadByte()
		if (c == -1) || (c == 0) {
			break
		}

		r.WriteByte(byte(c))
	}
	return r.String()
}

func (msg *QReadbuf) ReadStringLine() string {

	var r strings.Builder
	for {
		c := msg.ReadByte()
		if (c == -1) || (c == 0) || (c == '\n') {
			break
		}

		r.WriteByte(byte(c))
	}
	return r.String()
}

func (msg *QReadbuf) ReadCoord() float32 {
	return float32(msg.ReadShort()) * 0.125
}

func (msg *QReadbuf) ReadPos() []float32 {
	return []float32{
		float32(msg.ReadShort()) * 0.125,
		float32(msg.ReadShort()) * 0.125,
		float32(msg.ReadShort()) * 0.125,
	}
}

func (msg *QReadbuf) ReadAngle() float32 {
	return float32(msg.ReadChar()) * 1.40625
}

func (msg *QReadbuf) ReadAngle16() float32 {
	return SHORT2ANGLE(msg.ReadShort())
}

func (msg *QReadbuf) ReadData(data []byte, len int) {

	for i := 0; i < len; i++ {
		data[i] = byte(msg.ReadByte())
	}
}

func (msg *QReadbuf) ReadDir() []float32 {

	b := msg.ReadByte()
	if b < 0 || b >= len(bytedirs) {
		log.Fatal("MSF_ReadDir: out of range")
	}

	return bytedirs[b]
}

func (msg *QReadbuf) ReadDeltaUsercmd(from, move *Usercmd_t) {

	move.Copy(*from)

	bits := msg.ReadByte()

	/* read current angles */
	if (bits & CM_ANGLE1) != 0 {
		move.Angles[0] = int16(msg.ReadShort())
	}

	if (bits & CM_ANGLE2) != 0 {
		move.Angles[1] = int16(msg.ReadShort())
	}

	if (bits & CM_ANGLE3) != 0 {
		move.Angles[2] = int16(msg.ReadShort())
	}

	/* read movement */
	if (bits & CM_FORWARD) != 0 {
		move.Forwardmove = int16(msg.ReadShort())
	}

	if (bits & CM_SIDE) != 0 {
		move.Sidemove = int16(msg.ReadShort())
	}

	if (bits & CM_UP) != 0 {
		move.Upmove = int16(msg.ReadShort())
	}

	/* read buttons */
	if (bits & CM_BUTTONS) != 0 {
		move.Buttons = byte(msg.ReadByte())
	}

	if (bits & CM_IMPULSE) != 0 {
		move.Impulse = byte(msg.ReadByte())
	}

	/* read time to run command */
	move.Msec = byte(msg.ReadByte())

	/* read the light level */
	move.Lightlevel = byte(msg.ReadByte())
}

var bytedirs = [][]float32{
	{-0.525731, 0.000000, 0.850651},
	{-0.442863, 0.238856, 0.864188},
	{-0.295242, 0.000000, 0.955423},
	{-0.309017, 0.500000, 0.809017},
	{-0.162460, 0.262866, 0.951056},
	{0.000000, 0.000000, 1.000000},
	{0.000000, 0.850651, 0.525731},
	{-0.147621, 0.716567, 0.681718},
	{0.147621, 0.716567, 0.681718},
	{0.000000, 0.525731, 0.850651},
	{0.309017, 0.500000, 0.809017},
	{0.525731, 0.000000, 0.850651},
	{0.295242, 0.000000, 0.955423},
	{0.442863, 0.238856, 0.864188},
	{0.162460, 0.262866, 0.951056},
	{-0.681718, 0.147621, 0.716567},
	{-0.809017, 0.309017, 0.500000},
	{-0.587785, 0.425325, 0.688191},
	{-0.850651, 0.525731, 0.000000},
	{-0.864188, 0.442863, 0.238856},
	{-0.716567, 0.681718, 0.147621},
	{-0.688191, 0.587785, 0.425325},
	{-0.500000, 0.809017, 0.309017},
	{-0.238856, 0.864188, 0.442863},
	{-0.425325, 0.688191, 0.587785},
	{-0.716567, 0.681718, -0.147621},
	{-0.500000, 0.809017, -0.309017},
	{-0.525731, 0.850651, 0.000000},
	{0.000000, 0.850651, -0.525731},
	{-0.238856, 0.864188, -0.442863},
	{0.000000, 0.955423, -0.295242},
	{-0.262866, 0.951056, -0.162460},
	{0.000000, 1.000000, 0.000000},
	{0.000000, 0.955423, 0.295242},
	{-0.262866, 0.951056, 0.162460},
	{0.238856, 0.864188, 0.442863},
	{0.262866, 0.951056, 0.162460},
	{0.500000, 0.809017, 0.309017},
	{0.238856, 0.864188, -0.442863},
	{0.262866, 0.951056, -0.162460},
	{0.500000, 0.809017, -0.309017},
	{0.850651, 0.525731, 0.000000},
	{0.716567, 0.681718, 0.147621},
	{0.716567, 0.681718, -0.147621},
	{0.525731, 0.850651, 0.000000},
	{0.425325, 0.688191, 0.587785},
	{0.864188, 0.442863, 0.238856},
	{0.688191, 0.587785, 0.425325},
	{0.809017, 0.309017, 0.500000},
	{0.681718, 0.147621, 0.716567},
	{0.587785, 0.425325, 0.688191},
	{0.955423, 0.295242, 0.000000},
	{1.000000, 0.000000, 0.000000},
	{0.951056, 0.162460, 0.262866},
	{0.850651, -0.525731, 0.000000},
	{0.955423, -0.295242, 0.000000},
	{0.864188, -0.442863, 0.238856},
	{0.951056, -0.162460, 0.262866},
	{0.809017, -0.309017, 0.500000},
	{0.681718, -0.147621, 0.716567},
	{0.850651, 0.000000, 0.525731},
	{0.864188, 0.442863, -0.238856},
	{0.809017, 0.309017, -0.500000},
	{0.951056, 0.162460, -0.262866},
	{0.525731, 0.000000, -0.850651},
	{0.681718, 0.147621, -0.716567},
	{0.681718, -0.147621, -0.716567},
	{0.850651, 0.000000, -0.525731},
	{0.809017, -0.309017, -0.500000},
	{0.864188, -0.442863, -0.238856},
	{0.951056, -0.162460, -0.262866},
	{0.147621, 0.716567, -0.681718},
	{0.309017, 0.500000, -0.809017},
	{0.425325, 0.688191, -0.587785},
	{0.442863, 0.238856, -0.864188},
	{0.587785, 0.425325, -0.688191},
	{0.688191, 0.587785, -0.425325},
	{-0.147621, 0.716567, -0.681718},
	{-0.309017, 0.500000, -0.809017},
	{0.000000, 0.525731, -0.850651},
	{-0.525731, 0.000000, -0.850651},
	{-0.442863, 0.238856, -0.864188},
	{-0.295242, 0.000000, -0.955423},
	{-0.162460, 0.262866, -0.951056},
	{0.000000, 0.000000, -1.000000},
	{0.295242, 0.000000, -0.955423},
	{0.162460, 0.262866, -0.951056},
	{-0.442863, -0.238856, -0.864188},
	{-0.309017, -0.500000, -0.809017},
	{-0.162460, -0.262866, -0.951056},
	{0.000000, -0.850651, -0.525731},
	{-0.147621, -0.716567, -0.681718},
	{0.147621, -0.716567, -0.681718},
	{0.000000, -0.525731, -0.850651},
	{0.309017, -0.500000, -0.809017},
	{0.442863, -0.238856, -0.864188},
	{0.162460, -0.262866, -0.951056},
	{0.238856, -0.864188, -0.442863},
	{0.500000, -0.809017, -0.309017},
	{0.425325, -0.688191, -0.587785},
	{0.716567, -0.681718, -0.147621},
	{0.688191, -0.587785, -0.425325},
	{0.587785, -0.425325, -0.688191},
	{0.000000, -0.955423, -0.295242},
	{0.000000, -1.000000, 0.000000},
	{0.262866, -0.951056, -0.162460},
	{0.000000, -0.850651, 0.525731},
	{0.000000, -0.955423, 0.295242},
	{0.238856, -0.864188, 0.442863},
	{0.262866, -0.951056, 0.162460},
	{0.500000, -0.809017, 0.309017},
	{0.716567, -0.681718, 0.147621},
	{0.525731, -0.850651, 0.000000},
	{-0.238856, -0.864188, -0.442863},
	{-0.500000, -0.809017, -0.309017},
	{-0.262866, -0.951056, -0.162460},
	{-0.850651, -0.525731, 0.000000},
	{-0.716567, -0.681718, -0.147621},
	{-0.716567, -0.681718, 0.147621},
	{-0.525731, -0.850651, 0.000000},
	{-0.500000, -0.809017, 0.309017},
	{-0.238856, -0.864188, 0.442863},
	{-0.262866, -0.951056, 0.162460},
	{-0.864188, -0.442863, 0.238856},
	{-0.809017, -0.309017, 0.500000},
	{-0.688191, -0.587785, 0.425325},
	{-0.681718, -0.147621, 0.716567},
	{-0.442863, -0.238856, 0.864188},
	{-0.587785, -0.425325, 0.688191},
	{-0.309017, -0.500000, 0.809017},
	{-0.147621, -0.716567, 0.681718},
	{-0.425325, -0.688191, 0.587785},
	{-0.162460, -0.262866, 0.951056},
	{0.442863, -0.238856, 0.864188},
	{0.162460, -0.262866, 0.951056},
	{0.309017, -0.500000, 0.809017},
	{0.147621, -0.716567, 0.681718},
	{0.000000, -0.525731, 0.850651},
	{0.425325, -0.688191, 0.587785},
	{0.587785, -0.425325, 0.688191},
	{0.688191, -0.587785, 0.425325},
	{-0.955423, 0.295242, 0.000000},
	{-0.951056, 0.162460, 0.262866},
	{-1.000000, 0.000000, 0.000000},
	{-0.850651, 0.000000, 0.525731},
	{-0.955423, -0.295242, 0.000000},
	{-0.951056, -0.162460, 0.262866},
	{-0.864188, 0.442863, -0.238856},
	{-0.951056, 0.162460, -0.262866},
	{-0.809017, 0.309017, -0.500000},
	{-0.864188, -0.442863, -0.238856},
	{-0.951056, -0.162460, -0.262866},
	{-0.809017, -0.309017, -0.500000},
	{-0.681718, 0.147621, -0.716567},
	{-0.681718, -0.147621, -0.716567},
	{-0.850651, 0.000000, -0.525731},
	{-0.688191, 0.587785, -0.425325},
	{-0.587785, 0.425325, -0.688191},
	{-0.425325, 0.688191, -0.587785},
	{-0.425325, -0.688191, -0.587785},
	{-0.587785, -0.425325, -0.688191},
	{-0.688191, -0.587785, -0.425325}}
