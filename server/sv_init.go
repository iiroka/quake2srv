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
 * Server startup.
 *
 * =======================================================================
 */
package server

import (
	"fmt"
	"log"
	"quake2srv/shared"
	"strconv"
	"strings"
)

func (T *qServer) svFindIndex(name string, start, max int, create bool) int {

	if len(name) == 0 {
		return 0
	}

	index := -1
	for i := 1; i < max; i++ {
		if len(T.sv.configstrings[start+i]) == 0 {
			index = i
			break
		}
		if T.sv.configstrings[start+i] == name {
			return i
		}
	}

	if !create {
		return 0
	}

	if index < 0 {
		T.common.Com_Error(shared.ERR_DROP, "*Index: overflow")
		return 0
	}

	T.sv.configstrings[start+index] = name

	if T.sv.state != ss_loading {
		/* send the update to everyone */
		T.sv.multicast.WriteChar(shared.SvcConfigstring)
		T.sv.multicast.WriteShort(start + index)
		T.sv.multicast.WriteString(name)
		T.svMulticast([]float32{0, 0, 0}, shared.MULTICAST_ALL_R)
	}

	return index
}

/*
 * Entity baselines are used to compress the update messages
 * to the clients -- only the fields that differ from the
 * baseline will be transmitted
 */
func (T *qServer) createBaseline() {

	for entnum := 1; entnum < T.ge.NumEdicts(); entnum++ {
		svent := T.ge.Edict(entnum)

		if !svent.Inuse() {
			continue
		}

		if svent.S().Modelindex == 0 && svent.S().Sound == 0 && svent.S().Effects == 0 {
			continue
		}

		svent.S().Number = entnum

		/* take current state as baseline */
		//  VectorCopy(svent->s.origin, svent->s.old_origin);
		T.sv.baselines[entnum].Copy(*svent.S())
	}
}

/*
 * Change the server to a new map, taking all connected
 * clients along with it.
 */
func (T *qServer) spawnServer(server, spawnpoint string, serverstate server_state_t,
	attractloop, loadgame bool) error {
	//  int i;
	//  unsigned checksum;

	// if attractloop {
	// 	T.common.Cvar_Set("paused", "0")
	// }

	log.Printf("------- server initialization ------\n")
	log.Printf("SpawnServer: %s\n", server)

	//  if (sv.demofile) {
	// 	 FS_FCloseFile(sv.demofile);
	//  }

	T.svs.spawncount++ /* any partially connected client will be restarted */
	T.sv.state = ss_dead
	T.common.SetServerState(int(T.sv.state))

	/* wipe the entire per-level structure */
	T.sv = server_t{}
	T.svs.realtime = 0
	T.sv.loadgame = loadgame
	T.sv.attractloop = attractloop

	/* save name for levels that don't set message */
	T.sv.configstrings[shared.CS_NAME] = server

	if T.common.Cvar_VariableBool("deathmatch") {
		T.sv.configstrings[shared.CS_AIRACCEL] = fmt.Sprintf("%f", T.sv_airaccelerate.Float())
		// T.common.SetAirAccelerate(T.sv_airaccelerate.Float())
	} else {
		T.sv.configstrings[shared.CS_AIRACCEL] = "0"
		// T.common.SetAirAccelerate(0)
	}

	T.sv.multicast = shared.QWritebufCreate(shared.MAX_MSGLEN)

	T.sv.name = string(server)

	/* leave slots at start for clients only */
	for i := range T.svs.clients {
		/* needs to reconnect */
		if T.svs.clients[i].state > cs_connected {
			T.svs.clients[i].state = cs_connected
		}

		T.svs.clients[i].lastframe = -1
	}

	T.sv.time = 1000

	var checksum uint32
	var err error
	if serverstate != ss_game {
		T.sv.models[1], err = T.common.CMLoadMap("", false, &checksum) /* no real map */
	} else {
		T.sv.configstrings[shared.CS_MODELS+1] = fmt.Sprintf("maps/%s.bsp", server)
		T.sv.models[1], err = T.common.CMLoadMap(T.sv.configstrings[shared.CS_MODELS+1],
			false, &checksum)
	}
	if err != nil {
		return err
	}

	//  Com_sprintf(sv.configstrings[CS_MAPCHECKSUM],
	// 		 sizeof(sv.configstrings[CS_MAPCHECKSUM]),
	// 		 "%i", checksum);

	/* clear physics interaction links */
	T.svClearWorld()

	for i := 1; i < T.common.CMNumInlineModels(); i++ {
		T.sv.configstrings[shared.CS_MODELS+1+i] = fmt.Sprintf("*%v", i)
		T.sv.models[i+1], err = T.common.CMInlineModel(T.sv.configstrings[shared.CS_MODELS+1+i])
	}

	/* spawn the rest of the entities on the map */
	T.sv.state = ss_loading
	T.common.SetServerState(int(T.sv.state))

	/* load and spawn all other entities */
	if err := T.ge.SpawnEntities(T.sv.name, T.common.CMEntityString(), spawnpoint); err != nil {
		return err
	}

	/* run two frames to allow everything to settle */
	if err := T.ge.RunFrame(); err != nil {
		return err
	}
	if err := T.ge.RunFrame(); err != nil {
		return err
	}

	//  /* verify game didn't clobber important stuff */
	//  if ((int)checksum !=
	// 	 (int)strtol(sv.configstrings[CS_MAPCHECKSUM], (char **)NULL, 10))
	//  {
	// 	 Com_Error(ERR_DROP, "Game DLL corrupted server configstrings");
	//  }

	/* all precaches are complete */
	T.sv.state = serverstate
	T.common.SetServerState(int(T.sv.state))

	/* create a baseline for more efficient communications */
	T.createBaseline()

	//  /* check for a savegame */
	//  SV_CheckForSavegame();

	/* set serverinfo variable */
	T.common.Cvar_FullSet("mapname", T.sv.name, shared.CVAR_SERVERINFO|shared.CVAR_NOSET)

	log.Printf("------------------------------------\n\n")
	return nil
}

/*
 * A brand new game has been started
 */
func (T *qServer) initGame() error {
	// 	 int i;
	// 	 edict_t *ent;
	// 	 char idmaster[32];

	if T.svs.initialized {
		/* cause any connected clients to reconnect */
		// T.Shutdown("Server restarted\n", true)
	} else {
		// 		 /* make sure the client is down */
		// 		 CL_Drop();
		// 		 SCR_BeginLoadingPlaque();
	}

	/* get any latched variable changes (maxclients, etc) */
	// T.common.Cvar_GetLatchedVars()

	T.svs.initialized = true

	if T.common.Cvar_VariableBool("coop") && T.common.Cvar_VariableBool("deathmatch") {
		log.Printf("Deathmatch and Coop both set, disabling Coop\n")
		T.common.Cvar_FullSet("coop", "0", shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
	}

	/* dedicated servers can't be single player and are usually DM
	so unless they explicity set coop, force it to deathmatch */
	// if !T.common.Cvar_VariableBool("coop") {
	// 	T.common.Cvar_FullSet("deathmatch", "1", shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
	// }

	/* init clients */
	if T.common.Cvar_VariableBool("deathmatch") {
		if T.maxclients.Int() <= 1 {
			T.common.Cvar_FullSet("maxclients", "8", shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
		} else if T.maxclients.Int() > shared.MAX_CLIENTS {
			T.common.Cvar_FullSet("maxclients", strconv.Itoa(shared.MAX_CLIENTS), shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
		}
		T.common.Cvar_FullSet("singleplayer", "0", 0)
	} else if T.common.Cvar_VariableBool("coop") {
		if (T.maxclients.Int() <= 1) || (T.maxclients.Int() > 4) {
			T.common.Cvar_FullSet("maxclients", "4", shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
		}

		T.common.Cvar_FullSet("singleplayer", "0", 0)
	} else { /* non-deathmatch, non-coop is one player */
		T.common.Cvar_FullSet("maxclients", "1", shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
		T.common.Cvar_FullSet("singleplayer", "1", 0)
	}

	T.svs.spawncount = shared.Randk()
	T.svs.clients = make([]client_t, T.maxclients.Int())
	for i := range T.svs.clients {
		T.svs.clients[i].index = i
		T.svs.clients[i].datagram = shared.QWritebufCreate(shared.MAX_MSGLEN)
	}
	T.svs.num_client_entities = T.maxclients.Int() * shared.UPDATE_BACKUP * 64
	T.svs.client_entities = make([]shared.Entity_state_t, T.svs.num_client_entities)

	// 	 /* init network stuff */
	// 	 if (dedicated->value)
	// 	 {
	// 		 if (Cvar_VariableValue("singleplayer"))
	// 		 {
	// 			 NET_Config(true);
	// 		 }
	// 		 else
	// 		 {
	// 			 NET_Config((maxclients->value > 1));
	// 		 }
	// 	 }
	// 	 else
	// 	 {
	// T.common.NET_Config((T.maxclients.Int() > 1))
	// 	 }

	/* heartbeats will always be sent to the id master */
	T.svs.last_heartbeat = -99999 /* send immediately */
	// 	 Com_sprintf(idmaster, sizeof(idmaster), "192.246.40.37:%i", PORT_MASTER);
	// 	 NET_StringToAdr(idmaster, &master_adr[0]);

	/* init game */
	if err := T.svInitGameProgs(); err != nil {
		return err
	}

	for i := 0; i < T.maxclients.Int(); i++ {
		ent := T.ge.Edict(i + 1)
		ent.S().Number = i + 1
		T.svs.clients[i].edict = ent
		T.svs.clients[i].lastcmd.Copy(shared.Usercmd_t{})
	}
	return nil
}

/*
 * the full syntax is:
 *
 * map [*]<map>$<startspot>+<nextserver>
 *
 * command from the console or progs.
 * Map can also be a.cin, .pcx, or .dm2 file
 * Nextserver is used to allow a cinematic to play, then proceed to
 * another level:
 *
 *  map tram.cin+jail_e3
 */
func (T *qServer) svMap(attractloop bool, levelstring string, loadgame bool) error {

	T.sv.loadgame = loadgame
	T.sv.attractloop = attractloop

	if (T.sv.state == ss_dead) && !T.sv.loadgame {
		if err := T.initGame(); err != nil { /* the game is just starting */
			return err
		}
	}

	level := string(levelstring)

	/* if there is a + in the map, set nextserver to the remainder */
	ch := strings.IndexRune(level, '+')
	if ch >= 0 {
		T.common.Cvar_Set("nextserver", fmt.Sprintf("gamemap \"%v\"", level[ch+1:]))
		level = level[:ch]
	} else {
		// use next demo command if list of map commands as empty
		T.common.Cvar_Set("nextserver", T.common.Cvar_VariableString("nextdemo"))
		// and cleanup nextdemo
		T.common.Cvar_Set("nextdemo", "")
	}

	// 	/* hack for end game screen in coop mode */
	// 	if (Cvar_VariableValue("coop") && !Q_stricmp(level, "victory.pcx")) {
	// 		Cvar_Set("nextserver", "gamemap \"*base1\"");
	// 	}

	/* if there is a $, use the remainder as a spawnpoint */
	ch = strings.IndexRune(level, '$')
	spawnpoint := ""
	if ch >= 0 {
		spawnpoint = level[ch+1:]
		level = level[:ch]
	}

	// 	/* skip the end-of-unit flag if necessary */
	// 	l = strlen(level);

	if level[0] == '*' {
		level = level[1:]
	}

	if strings.HasSuffix(level, ".cin") {
		T.svBroadcastCommand("changing\n")
		if err := T.spawnServer(level, spawnpoint, ss_cinematic, attractloop, loadgame); err != nil {
			return err
		}
	} else if strings.HasSuffix(level, ".dm2") {
		T.svBroadcastCommand("changing\n")
		if err := T.spawnServer(level, spawnpoint, ss_demo, attractloop, loadgame); err != nil {
			return err
		}
	} else if strings.HasSuffix(level, ".pcx") {
		T.svBroadcastCommand("changing\n")
		if err := T.spawnServer(level, spawnpoint, ss_pic, attractloop, loadgame); err != nil {
			return err
		}
	} else {
		T.svBroadcastCommand("changing\n")
		// 	// 		SV_SendClientMessages();
		if err := T.spawnServer(level, spawnpoint, ss_game, attractloop, loadgame); err != nil {
			return err
		}
		// 	// 		Cbuf_CopyToDefer();
	}

	T.svBroadcastCommand("reconnect\n")
	return nil
}
