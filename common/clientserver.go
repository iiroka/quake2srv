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
 * Client / Server interactions
 *
 * =======================================================================
 */
package common

import (
	"fmt"
	"log"
	"quake2srv/shared"
)

type AbortFrame struct{}

func (m *AbortFrame) Error() string {
	return "abortframe"
}

/*
 * Both client and server can use this, and it will
 * do the apropriate things.
 */
func (T *qCommon) Com_Error(code int, format string, a ...interface{}) error {

	if T.recursive {
		log.Fatalf("recursive error after: %v", T.msg)
	}

	T.recursive = true

	T.msg = fmt.Sprintf(format, a...)

	if code == shared.ERR_DISCONNECT {
		// CL_Drop()
		T.recursive = false
		return &AbortFrame{}
	} else if code == shared.ERR_DROP {
		log.Printf("********************\nERROR: %s\n********************\n", T.msg)
		// SV_Shutdown(va("Server crashed: %s\n", msg), false)
		// CL_Drop()
		T.recursive = false
		return &AbortFrame{}
	} else {
		// SV_Shutdown(va("Server fatal crashed: %s\n", msg), false)
		// CL_Shutdown()
	}

	// if logfile {
	// 	fclose(logfile)
	// 	logfile = NULL
	// }

	log.Fatal(T.msg)
	T.recursive = false
	return nil
}
