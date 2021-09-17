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
 * Network connections over IPv4, IPv6 and IPX via Winsocks.
 *
 * =======================================================================
 */
package common

func (Q *qCommon) RegisterClient(addr string, handler func([]byte, interface{}), context interface{}) {
	Q.net_clients[addr] = qNetClient{handler, context}
}

func (Q *qCommon) RxHandler(from string, data []byte) {
	Q.net_ch <- qNetMsg{from, data}
}

func (Q *qCommon) DisconnectHandler(addr string) {
	println("DisconnectHandler", addr)
	Q.net_disc <- addr
}

func (Q *qCommon) NET_GetDisconnected() string {
	select {
	case adr := <-Q.net_disc:
		println("NET_GetDisconnected", adr)
		return adr
	default:
		return ""
	}
}

func (Q *qCommon) NET_GetPacket() (string, []byte) {
	select {
	case rx := <-Q.net_ch:
		return rx.from, rx.data
	default:
		return "", nil
	}
}

func (Q *qCommon) NET_SendPacket(data []byte, addr string) {
	if cl, ok := Q.net_clients[addr]; ok {
		cl.handler(data, cl.context)
	}
}
