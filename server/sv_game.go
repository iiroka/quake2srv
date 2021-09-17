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
 * Interface between the server and the game module.
 *
 * =======================================================================
 */
package server

import (
	"fmt"
	"log"
	"quake2srv/game"
	"quake2srv/shared"
)

type qGameImp struct {
	T *qServer
}

/*
 * Debug print to server console
 */
func (G *qGameImp) Dprintf(format string, a ...interface{}) {
	log.Printf(format, a...)
}

func (G *qGameImp) Cvar(var_name, value string, flags int) *shared.CvarT {
	return G.T.common.Cvar_Get(var_name, value, flags)
}

func (G *qGameImp) Error(format string, a ...interface{}) error {
	return G.T.common.Com_Error(shared.ERR_DROP, "Game Error: %s", fmt.Sprintf(format, a...))
}

func (G *qGameImp) Configstring(num int, str string) error {
	if (num < 0) || (num >= shared.MAX_CONFIGSTRINGS) {
		return G.T.common.Com_Error(shared.ERR_DROP, "configstring: bad index %i\n", num)
	}

	/* change the string in sv */
	G.T.sv.configstrings[num] = str

	if G.T.sv.state != ss_loading {
		/* send the update to everyone */
		G.T.sv.multicast.Clear()
		G.T.sv.multicast.WriteChar(shared.SvcConfigstring)
		G.T.sv.multicast.WriteShort(num)
		G.T.sv.multicast.WriteString(str)
		G.T.svMulticast([]float32{0, 0, 0}, shared.MULTICAST_ALL_R)
	}
	return nil
}

func (G *qGameImp) Modelindex(name string) int {
	return G.T.svFindIndex(name, shared.CS_MODELS, shared.MAX_MODELS, true)
}

func (G *qGameImp) Soundindex(name string) int {
	return G.T.svFindIndex(name, shared.CS_SOUNDS, shared.MAX_SOUNDS, true)
}

func (G *qGameImp) Imageindex(name string) int {
	return G.T.svFindIndex(name, shared.CS_IMAGES, shared.MAX_IMAGES, true)
}

func (G *qGameImp) Linkentity(ent shared.Edict_s) {
	G.T.svLinkEdict(ent)
}

func (G *qGameImp) Unlinkentity(ent shared.Edict_s) {
	G.T.svUnlinkEdict(ent)
}

func (G *qGameImp) Pmove(pmove *shared.Pmove_t) {
	G.T.common.Pmove(pmove)
}

func (G *qGameImp) Trace(start, mins, maxs, end []float32, passent shared.Edict_s, contentmask int) shared.Trace_t {
	return G.T.svTrace(start, mins, maxs, end, passent, contentmask)
}

func (G *qGameImp) Pointcontents(point []float32) int {
	return G.T.svPointContents(point)
}

func (G *qGameImp) BoxEdicts(mins, maxs []float32, edicts []shared.Edict_s, maxcount, areatype int) int {
	return G.T.svAreaEdicts(mins, maxs, edicts, maxcount, areatype)
}

/*
 * Also sets mins and maxs for inline bmodels
 */
func (G *qGameImp) Setmodel(ent shared.Edict_s, name string) error {

	if len(name) == 0 {
		return G.T.common.Com_Error(shared.ERR_DROP, "PF_setmodel: NULL")
	}

	i := G.T.svFindIndex(name, shared.CS_MODELS, shared.MAX_MODELS, true)
	ent.S().Modelindex = i

	/* if it is an inline model, get
	the size information for it */
	if name[0] == '*' {
		mod, err := G.T.common.CMInlineModel(name)
		if err != nil {
			return err
		}
		copy(ent.Mins(), mod.Mins[:])
		copy(ent.Maxs(), mod.Maxs[:])
		G.T.svLinkEdict(ent)
	}
	return nil
}

/*
 * Also checks portalareas so that doors block sight
 */
func (G *qGameImp) InPVS(p1, p2 []float32) bool {
	//  int leafnum;
	//  int cluster;
	//  int area1, area2;
	//  byte *mask;

	leafnum := G.T.common.CMPointLeafnum(p1)
	cluster := G.T.common.CMLeafCluster(leafnum)
	area1 := G.T.common.CMLeafArea(leafnum)
	mask := G.T.common.CMClusterPVS(cluster)

	leafnum = G.T.common.CMPointLeafnum(p2)
	cluster = G.T.common.CMLeafCluster(leafnum)
	area2 := G.T.common.CMLeafArea(leafnum)

	if mask != nil && ((mask[cluster>>3] & (1 << (cluster & 7))) == 0) {
		return false
	}

	if !G.T.common.CMAreasConnected(area1, area2) {
		return false /* a door blocks sight */
	}

	return true
}

/*
 * Also checks portalareas so that doors block sound
 */
func (G *qGameImp) InPHS(p1, p2 []float32) bool {
	//  int leafnum;
	//  int cluster;
	//  int area1, area2;
	//  byte *mask;

	leafnum := G.T.common.CMPointLeafnum(p1)
	cluster := G.T.common.CMLeafCluster(leafnum)
	area1 := G.T.common.CMLeafArea(leafnum)
	mask := G.T.common.CMClusterPHS(cluster)

	leafnum = G.T.common.CMPointLeafnum(p2)
	cluster = G.T.common.CMLeafCluster(leafnum)
	area2 := G.T.common.CMLeafArea(leafnum)

	if mask != nil && ((mask[cluster>>3] & (1 << (cluster & 7))) == 0) {
		return false /* more than one bounce away */
	}

	if !G.T.common.CMAreasConnected(area1, area2) {
		return false /* a door blocks hearing */
	}

	return true
}

/*
 * Called when either the entire server is being killed, or
 * it is changing to a different game directory.
 */
func (T *qServer) svShutdownGameProgs() {
	if T.ge == nil {
		return
	}

	T.ge.Shutdown()
	T.ge = nil
}

/*
 * Init the game subsystem for a new map
 */
func (T *qServer) svInitGameProgs() error {
	// 	 game_import_t import;

	// 	 /* unload anything we have now */
	// 	 if (ge)
	// 	 {
	// 		 SV_ShutdownGameProgs();
	// 	 }

	log.Printf("-------- game initialization -------\n")

	/* load a new game dll */
	// 	 import.multicast = SV_Multicast;
	// 	 import.unicast = PF_Unicast;
	// 	 import.bprintf = SV_BroadcastPrintf;
	// 	 import.dprintf = PF_dprintf;
	// 	 import.cprintf = PF_cprintf;
	// 	 import.centerprintf = PF_centerprintf;
	// 	 import.error = PF_error;

	// 	 import.linkentity = SV_LinkEdict;
	// 	 import.unlinkentity = SV_UnlinkEdict;
	// 	 import.BoxEdicts = SV_AreaEdicts;
	// 	 import.trace = SV_Trace;
	// 	 import.pointcontents = SV_PointContents;
	// 	 import.setmodel = PF_setmodel;
	// 	 import.inPVS = PF_inPVS;
	// 	 import.inPHS = PF_inPHS;
	// 	 import.Pmove = Pmove;

	// 	 import.modelindex = SV_ModelIndex;
	// 	 import.soundindex = SV_SoundIndex;
	// 	 import.imageindex = SV_ImageIndex;

	// 	 import.configstring = PF_Configstring;
	// 	 import.sound = PF_StartSound;
	// 	 import.positioned_sound = SV_StartSound;

	// 	 import.WriteChar = PF_WriteChar;
	// 	 import.WriteByte = PF_WriteByte;
	// 	 import.WriteShort = PF_WriteShort;
	// 	 import.WriteLong = PF_WriteLong;
	// 	 import.WriteFloat = PF_WriteFloat;
	// 	 import.WriteString = PF_WriteString;
	// 	 import.WritePosition = PF_WritePos;
	// 	 import.WriteDir = PF_WriteDir;
	// 	 import.WriteAngle = PF_WriteAngle;

	// 	 import.TagMalloc = Z_TagMalloc;
	// 	 import.TagFree = Z_Free;
	// 	 import.FreeTags = Z_FreeTags;

	// 	 import.cvar = Cvar_Get;
	// 	 import.cvar_set = Cvar_Set;
	// 	 import.cvar_forceset = Cvar_ForceSet;

	// 	 import.argc = Cmd_Argc;
	// 	 import.argv = Cmd_Argv;
	// 	 import.args = Cmd_Args;
	// 	 import.AddCommandString = Cbuf_AddText;

	//  #ifndef DEDICATED_ONLY
	// 	 import.DebugGraph = SCR_DebugGraph;
	//  #endif

	// 	 import.SetAreaPortalState = CM_SetAreaPortalState;
	// 	 import.AreasConnected = CM_AreasConnected;

	T.ge = game.QGameCreate(&qGameImp{T})
	if T.ge == nil {
		return T.common.Com_Error(shared.ERR_DROP, "failed to load game DLL")
	}

	// 	 if (ge->apiversion != GAME_API_VERSION)
	// 	 {
	// 		 Com_Error(ERR_DROP, "game is version %i, not %i", ge->apiversion,
	// 				 GAME_API_VERSION);
	// 	 }

	T.ge.Init()

	log.Printf("------------------------------------\n\n")
	return nil
}
