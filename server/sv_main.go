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
 * Server main function and correspondig stuff
 *
 * =======================================================================
 */
package server

import (
	"log"
	"quake2srv/shared"
	"time"
)

/*
 * Called when the player is totally leaving the server, either willingly
 * or unwillingly.  This is NOT called if the entire server is quiting
 * or crashing.
 */
func (Q *qServer) dropClient(drop *client_t) {
	/* add the disconnect */
	//  MSG_WriteByte(&drop->netchan.message, svc_disconnect);

	//  if (drop->state == cs_spawned) {
	/* call the prog function for removing a client
	this will remove the body, among other things */
	//  ge->ClientDisconnect(drop->edict);
	//  }

	//  if (drop->download)
	//  {
	// 	 FS_FreeFile(drop->download);
	// 	 drop->download = NULL;
	//  }

	drop.state = cs_zombie /* become free in a few seconds */
	drop.name = ""
}

func (Q *qServer) Init() error {
	Q.initOperatorCommands()

	// rcon_password = Cvar_Get("rcon_password", "", 0);
	Q.common.Cvar_Get("skill", "1", 0)
	Q.common.Cvar_Get("singleplayer", "0", 0)
	Q.common.Cvar_Get("deathmatch", "0", shared.CVAR_LATCH)
	Q.common.Cvar_Get("coop", "0", shared.CVAR_LATCH)
	// Q.common.Cvar_Get("dmflags", va("%i", DF_INSTANT_ITEMS), CVAR_SERVERINFO);
	Q.common.Cvar_Get("fraglimit", "0", shared.CVAR_SERVERINFO)
	Q.common.Cvar_Get("timelimit", "0", shared.CVAR_SERVERINFO)
	Q.common.Cvar_Get("cheats", "0", shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
	// Q.common.Cvar_Get("protocol", va("%i", PROTOCOL_VERSION), CVAR_SERVERINFO | CVAR_NOSET);
	Q.maxclients = Q.common.Cvar_Get("maxclients", "1", shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
	Q.hostname = Q.common.Cvar_Get("hostname", "noname", shared.CVAR_SERVERINFO|shared.CVAR_ARCHIVE)
	Q.timeout = Q.common.Cvar_Get("timeout", "125", 0)
	Q.zombietime = Q.common.Cvar_Get("zombietime", "2", 0)
	Q.sv_showclamp = Q.common.Cvar_Get("showclamp", "0", 0)
	Q.sv_paused = Q.common.Cvar_Get("paused", "0", 0)
	Q.sv_timedemo = Q.common.Cvar_Get("timedemo", "0", 0)
	Q.sv_enforcetime = Q.common.Cvar_Get("sv_enforcetime", "0", 0)
	Q.allow_download = Q.common.Cvar_Get("allow_download", "1", shared.CVAR_ARCHIVE)
	Q.allow_download_players = Q.common.Cvar_Get("allow_download_players", "0", shared.CVAR_ARCHIVE)
	Q.allow_download_models = Q.common.Cvar_Get("allow_download_models", "1", shared.CVAR_ARCHIVE)
	Q.allow_download_sounds = Q.common.Cvar_Get("allow_download_sounds", "1", shared.CVAR_ARCHIVE)
	Q.allow_download_maps = Q.common.Cvar_Get("allow_download_maps", "1", shared.CVAR_ARCHIVE)
	Q.sv_downloadserver = Q.common.Cvar_Get("sv_downloadserver", "", 0)

	Q.sv_noreload = Q.common.Cvar_Get("sv_noreload", "0", 0)

	Q.sv_airaccelerate = Q.common.Cvar_Get("sv_airaccelerate", "0", shared.CVAR_LATCH)

	Q.public_server = Q.common.Cvar_Get("public", "0", 0)

	Q.sv_entfile = Q.common.Cvar_Get("sv_entfile", "1", shared.CVAR_ARCHIVE)

	// SZ_Init(&net_message, net_message_buffer, sizeof(net_message_buffer));
	return nil
}

func (T *qServer) readPackets() error {

	for {
		from, data := T.common.NET_GetPacket()
		if data == nil {
			break
		}
		/* check for connectionless packet (0xffffffff) first */
		id := shared.ReadInt32(data)
		if id == -1 {
			T.connectionlessPacket(shared.QReadbufCreate(data), from)
			continue
		}

		msg := shared.QReadbufCreate(data)
		/* read the qport out of the message so we can fix up
		   stupid address translating routers */
		msg.BeginReading()
		msg.ReadLong() /* sequence number */
		msg.ReadLong() /* sequence number */
		qport := msg.ReadShort() & 0xffff

		/* check for packets from connected clients */
		for i, cl := range T.svs.clients {
			if cl.state == cs_free {
				continue
			}

			if cl.addr != from {
				continue
			}

			if cl.netchan.Qport != qport {
				println("Port does not match")
				continue
			}

			if T.svs.clients[i].netchan.Process(msg) {
				/* this is a valid, sequenced packet, so process it */
				if cl.state != cs_zombie {
					cl.lastmessage = T.svs.realtime /* don't timeout */

					// 			if !(T.sv.demofile != nil && (T.sv.state == ss_demo)) {
					if err := T.executeClientMessage(&T.svs.clients[i], msg); err != nil {
						return err
					}
					// 			}
				}
			}

			break
		}
	}

	for {
		disc := T.common.NET_GetDisconnected()
		if len(disc) == 0 {
			break
		}
		for i, cl := range T.svs.clients {
			if cl.state == cs_free {
				continue
			}

			if cl.addr != disc {
				continue
			}

			log.Printf("%v disconnected\n", disc)
			T.svs.clients[i].state = cs_zombie
		}
	}
	return nil
}

/*
 * If a packet has not been received from a client for timeout->value
 * seconds, drop the conneciton.  Server frames are used instead of
 * realtime to avoid dropping the local client while debugging.
 *
 * When a client is normally dropped, the client_t goes into a zombie state
 * for a few seconds to make sure any final reliable message gets resent
 * if necessary
 */
func (T *qServer) checkTimeouts() {
	//  int i;
	//  client_t *cl;
	//  int droppoint;
	//  int zombiepoint;

	// droppoint := T.svs.realtime - 1000*T.timeout.Int()
	zombiepoint := T.svs.realtime - 1000*T.zombietime.Int()
	droppedSome := false

	for i, cl := range T.svs.clients {
		/* message times may be wrong across a changelevel */
		if cl.lastmessage > T.svs.realtime {
			T.svs.clients[i].lastmessage = T.svs.realtime
		}

		if (cl.state == cs_zombie) &&
			(cl.lastmessage < zombiepoint) {
			T.svs.clients[i].state = cs_free /* can now be reused */
			droppedSome = true
			continue
		}

		// 	 if (((cl->state == cs_connected) || (cl->state == cs_spawned)) &&
		// 		 (cl->lastmessage < droppoint))
		// 	 {
		// 		 SV_BroadcastPrintf(PRINT_HIGH, "%s timed out\n", cl->name);
		// 		 SV_DropClient(cl);
		// 		 cl->state = cs_free; /* don't bother with zombie state */
		// 	 }
	}
	if droppedSome {
		stillAlive := false
		for _, cl := range T.svs.clients {
			if cl.state != cs_free {
				stillAlive = true
			}
		}
		if !stillAlive {
			T.common.Quit()
		}
	}
}

func (T *qServer) runGameFrame() error {
	// #ifndef DEDICATED_ONLY
	// 	if (host_speeds->value)
	// 	{
	// 		time_before_game = Sys_Milliseconds();
	// 	}
	// #endif

	/* we always need to bump framenum, even if we
	   don't run the world, otherwise the delta
	   compression can get confused when a client
	   has the "current" frame */
	T.sv.framenum++
	T.sv.time = uint(T.sv.framenum * 100)

	/* don't run if paused */
	if !T.sv_paused.Bool() || (T.maxclients.Int() > 1) {
		if err := T.ge.RunFrame(); err != nil {
			return err
		}

		/* never get more than one tic behind */
		if int(T.sv.time) < T.svs.realtime {
			if T.sv_showclamp.Bool() {
				log.Printf("sv highclamp\n")
			}

			T.svs.realtime = int(T.sv.time)
		}
	}

	// #ifndef DEDICATED_ONLY
	// 	if (host_speeds->value)
	// 	{
	// 		time_after_game = Sys_Milliseconds();
	// 	}
	// #endif
	return nil
}

func (T *qServer) Frame(usec int) error {
	// time_before_game = time_after_game = 0;

	/* if server is not active, do nothing */
	if !T.svs.initialized {
		return nil
	}

	T.svs.realtime += usec / 1000

	/* keep the random time dependent */
	shared.Randk()

	/* check timeouts */
	T.checkTimeouts()

	/* get packets from clients */
	if err := T.readPackets(); err != nil {
		return err
	}

	/* move autonomous things around if enough time has passed */
	if !T.sv_timedemo.Bool() && (T.svs.realtime < int(T.sv.time)) {
		/* never let the time get too far off */
		if int(T.sv.time)-T.svs.realtime > 100 {
			if T.sv_showclamp.Bool() {
				log.Printf("sv lowclamp\n")
			}

			T.svs.realtime = int(T.sv.time - 100)
		}

		time.Sleep(time.Duration(int(T.sv.time)-T.svs.realtime) * time.Millisecond)
		return nil
	}

	/* update ping based on the last known frame from all clients */
	// SV_CalcPings();

	/* give the clients some timeslices */
	// SV_GiveMsec();

	/* let everything in the world think and move */
	if err := T.runGameFrame(); err != nil {
		return err
	}

	/* send messages back to the clients that had packets read this frame */
	T.svSendClientMessages()

	/* save the entire world state if recording a serverdemo */
	// SV_RecordDemoMessage();

	/* send a heartbeat to the master if needed */
	// Master_Heartbeat();

	/* clear teleport flags, etc for next frame */
	// SV_PrepWorldFrame();
	return nil
}
