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
 * Server side user (player entity) moving.
 *
 * =======================================================================
 */
package server

import (
	"fmt"
	"log"
	"quake2srv/shared"
	"strconv"
)

const maxSTRINGCMDS = 8

/*
 * Sends the first message from the server to a connected client.
 * This will be sent on the initial connection and upon each server load.
 */
func sv_New_f(args []string, T *qServer) error {
	//  static char *gamedir;
	//  int playernum;
	//  edict_t *ent;

	log.Printf("New() from %s\n", T.sv_client.name)

	if T.sv_client.state != cs_connected {
		log.Printf("New not valid -- already spawned\n")
		return nil
	}

	/* demo servers just dump the file message */
	// if T.sv.state == ss_demo {
	// 	return T.beginDemoserver()
	// }

	/* serverdata needs to go over for all types of servers
	to make sure the protocol is right, and to set the gamedir */
	gamedir := T.common.Cvar_VariableString("gamedir")

	/* send the serverdata */
	T.sv_client.netchan.Message.WriteByte(shared.SvcServerdata)
	T.sv_client.netchan.Message.WriteLong(shared.PROTOCOL_VERSION)
	T.sv_client.netchan.Message.WriteLong(T.svs.spawncount)
	if T.sv.attractloop {
		T.sv_client.netchan.Message.WriteByte(1)
	} else {
		T.sv_client.netchan.Message.WriteByte(0)
	}
	T.sv_client.netchan.Message.WriteString(gamedir)

	var playernum int
	if (T.sv.state == ss_cinematic) || (T.sv.state == ss_pic) {
		playernum = -1
	} else {
		playernum = T.sv_client.index
	}

	T.sv_client.netchan.Message.WriteShort(playernum)

	/* send full levelname */
	T.sv_client.netchan.Message.WriteString(T.sv.configstrings[shared.CS_NAME])

	/* game server */
	if T.sv.state == ss_game {
		/* set up the entity for the client */
		ent := T.ge.Edict(playernum + 1)
		ent.S().Number = playernum + 1
		T.sv_client.edict = ent
		T.sv_client.lastcmd.Copy(shared.Usercmd_t{})

		/* begin fetching configstrings */
		T.sv_client.netchan.Message.WriteByte(shared.SvcStufftext)
		T.sv_client.netchan.Message.WriteString(fmt.Sprintf("cmd configstrings %v 0\n", T.svs.spawncount))
	}
	return nil
}

func sv_Configstrings_f(args []string, T *qServer) error {

	log.Printf("Configstrings() from %s\n", T.sv_client.name)

	if T.sv_client.state != cs_connected {
		log.Printf("configstrings not valid -- already spawned\n")
		return nil
	}

	/* handle the case of a level changing while a client was connecting */
	sc, _ := strconv.ParseInt(args[1], 10, 32)
	if int(sc) != T.svs.spawncount {
		log.Printf("SV_Configstrings_f from different level\n")
		sv_New_f([]string{}, T)
		return nil
	}

	start, _ := strconv.ParseInt(args[2], 10, 32)

	/* write a packet full of data */
	for T.sv_client.netchan.Message.Cursize < shared.MAX_MSGLEN/2 &&
		start < shared.MAX_CONFIGSTRINGS {
		if len(T.sv.configstrings[start]) > 0 {
			T.sv_client.netchan.Message.WriteByte(shared.SvcConfigstring)
			T.sv_client.netchan.Message.WriteShort(int(start))
			T.sv_client.netchan.Message.WriteString(T.sv.configstrings[start])
		}

		start++
	}

	/* send next command */
	if start == shared.MAX_CONFIGSTRINGS {
		T.sv_client.netchan.Message.WriteByte(shared.SvcStufftext)
		T.sv_client.netchan.Message.WriteString(fmt.Sprintf("cmd baselines %v 0\n", T.svs.spawncount))
	} else {
		T.sv_client.netchan.Message.WriteByte(shared.SvcStufftext)
		T.sv_client.netchan.Message.WriteString(fmt.Sprintf("cmd configstrings %v %v\n", T.svs.spawncount, start))
	}
	return nil
}

func sv_Baselines_f(args []string, T *qServer) error {

	log.Printf("Baselines() from %s\n", T.sv_client.name)

	if T.sv_client.state != cs_connected {
		log.Printf("baselines not valid -- already spawned\n")
		return nil
	}

	/* handle the case of a level changing while a client was connecting */
	sc, _ := strconv.ParseInt(args[1], 10, 32)
	if int(sc) != T.svs.spawncount {
		log.Printf("SV_Baselines_f from different level\n")
		sv_New_f([]string{}, T)
		return nil
	}

	start, _ := strconv.ParseInt(args[2], 10, 32)
	nullstate := shared.Entity_state_t{}

	/* write a packet full of data */
	for T.sv_client.netchan.Message.Cursize < shared.MAX_MSGLEN/2 &&
		start < shared.MAX_EDICTS {
		base := &T.sv.baselines[start]

		if base.Modelindex != 0 || base.Sound != 0 || base.Effects != 0 {
			T.sv_client.netchan.Message.WriteByte(shared.SvcSpawnbaseline)
			T.sv_client.netchan.Message.WriteDeltaEntity(&nullstate, base, true, true)
		}

		start++
	}

	/* send next command */
	if start == shared.MAX_EDICTS {
		T.sv_client.netchan.Message.WriteByte(shared.SvcStufftext)
		T.sv_client.netchan.Message.WriteString(fmt.Sprintf("precache %v\n", T.svs.spawncount))
	} else {
		T.sv_client.netchan.Message.WriteByte(shared.SvcStufftext)
		T.sv_client.netchan.Message.WriteString(fmt.Sprintf("cmd baselines %v %v\n", T.svs.spawncount, start))
	}
	return nil
}

func sv_Begin_f(args []string, T *qServer) error {
	log.Printf("Begin() from %s\n", T.sv_client.name)

	/* handle the case of a level changing while a client was connecting */
	sc, _ := strconv.ParseInt(args[1], 10, 32)
	if int(sc) != T.svs.spawncount {
		log.Printf("SV_Begin_f from different level\n")
		sv_New_f([]string{}, T)
		return nil
	}

	T.sv_client.state = cs_spawned

	/* call the game begin function */
	if err := T.ge.ClientBegin(T.sv_player); err != nil {
		return err
	}

	// Cbuf_InsertFromDefer();
	return nil
}

/*
 * The client is going to disconnect, so remove the connection immediately
 */
func sv_Disconnect_f(args []string, T *qServer) error {
	T.dropClient(T.sv_client)
	return nil
}

func (T *qServer) svNextserver() {

	if (T.sv.state == ss_game) ||
		((T.sv.state == ss_pic) &&
			!T.common.Cvar_VariableBool("coop")) {
		return /* can't nextserver while playing a normal game */
	}

	T.svs.spawncount++ /* make sure another doesn't sneak in */
	v := T.common.Cvar_VariableString("nextserver")

	if len(v) == 0 {
		T.common.Cbuf_AddText("killserver\n")
	} else {
		T.common.Cbuf_AddText(v)
		T.common.Cbuf_AddText("\n")
	}

	T.common.Cvar_Set("nextserver", "")
}

/*
 * A cinematic has completed or been aborted by a client, so move
 * to the next server,
 */
func sv_Nextserver_f(args []string, T *qServer) error {
	sc, _ := strconv.ParseInt(args[1], 10, 32)
	if int(sc) != T.svs.spawncount {
		log.Printf("Nextserver() from wrong level, from %s %v != %v\n", T.sv_client.name, sc, T.svs.spawncount)
		return nil /* leftover from last server */
	}

	log.Printf("Nextserver() from %s\n", T.sv_client.name)

	T.svNextserver()
	return nil
}

var ucmds = map[string](func([]string, *qServer) error){
	/* auto issued */
	"new":           sv_New_f,
	"configstrings": sv_Configstrings_f,
	"baselines":     sv_Baselines_f,
	"begin":         sv_Begin_f,
	"nextserver":    sv_Nextserver_f,
	"disconnect":    sv_Disconnect_f,

	/* issued by hand at client consoles */
	// {"info", SV_ShowServerinfo_f},
}

func (T *qServer) executeUserCommand(s string) error {

	/* Security Fix... This is being set to false so that client's can't
	   macro expand variables on the server.  It seems unlikely that a
	   client ever ought to need to be able to do this... */
	args := shared.Cmd_TokenizeString(s, false)
	T.sv_player = T.sv_client.edict

	if u, ok := ucmds[args[0]]; ok {
		return u(args, T)
	}

	println("executeUserCommand", args[0])
	if T.sv.state == ss_game {
		T.ge.ClientCommand(T.sv_player, args)
	}
	return nil
}

func (T *qServer) svClientThink(cl *client_t, cmd *shared.Usercmd_t) {
	cl.commandMsec -= int(cmd.Msec)

	if (cl.commandMsec < 0) && T.sv_enforcetime.Bool() {
		log.Printf("commandMsec underflow from %s\n", cl.name)
		return
	}

	T.ge.ClientThink(cl.edict, cmd)
}

/*
 * The current net_message is parsed for the given client
 */
func (T *qServer) executeClientMessage(cl *client_t, msg *shared.QReadbuf) error {
	//  int c;
	//  char *s;

	//  usercmd_t nullcmd;
	//  usercmd_t oldest, oldcmd, newcmd;
	//  int net_drop;
	//  int stringCmdCount;
	//  int checksum, calculatedChecksum;
	//  int checksumIndex;
	//  qboolean move_issued;
	//  int lastframe;

	T.sv_client = cl
	T.sv_player = T.sv_client.edict

	/* only allow one move command */
	move_issued := false
	stringCmdCount := 0

	for {
		if msg.IsOver() {
			log.Printf("SV_ReadClientMessage: badread\n")
			// SV_DropClient(cl)
			return nil
		}

		c := msg.ReadByte()

		if c == -1 {
			break
		}

		switch c {

		case shared.ClcNop:

		case shared.ClcUserinfo:
			cl.userinfo = msg.ReadString()
			// 			 SV_UserinfoChanged(cl);

		case shared.ClcMove:

			if move_issued {
				return nil /* someone is trying to cheat... */
			}

			move_issued = true
			// 			 checksumIndex = net_message.readcount;
			_ = msg.ReadByte()
			lastframe := msg.ReadLong()

			if lastframe != cl.lastframe {
				cl.lastframe = lastframe

				// if cl.lastframe > 0 {
				// 	cl.frame_latency[cl.lastframe&(LATENCY_COUNTS-1)] =
				// 		svs.realtime - cl.frames[cl.lastframe&UPDATE_MASK].senttime
				// }
			}

			// 			 memset(&nullcmd, 0, sizeof(nullcmd));
			oldest := shared.Usercmd_t{}
			msg.ReadDeltaUsercmd(&shared.Usercmd_t{}, &oldest)
			oldcmd := shared.Usercmd_t{}
			msg.ReadDeltaUsercmd(&oldest, &oldcmd)
			newcmd := shared.Usercmd_t{}
			msg.ReadDeltaUsercmd(&oldcmd, &newcmd)

			if cl.state != cs_spawned {
				cl.lastframe = -1
				break
			}

			// 			 /* if the checksum fails, ignore the rest of the packet */
			// 			 calculatedChecksum = COM_BlockSequenceCRCByte(
			// 				 net_message.data + checksumIndex + 1,
			// 				 net_message.readcount - checksumIndex - 1,
			// 				 cl->netchan.incoming_sequence);

			// 			 if (calculatedChecksum != checksum) {
			// 				 Com_DPrintf("Failed command checksum for %s (%d != %d)/%d\n",
			// 						 cl->name, calculatedChecksum, checksum,
			// 						 cl->netchan.incoming_sequence);
			// 				 return;
			// 			 }

			if !T.sv_paused.Bool() {
				net_drop := cl.netchan.Dropped

				if net_drop < 20 {
					for net_drop > 2 {
						T.svClientThink(cl, &cl.lastcmd)

						net_drop--
					}

					if net_drop > 1 {
						T.svClientThink(cl, &oldest)
					}

					if net_drop > 0 {
						T.svClientThink(cl, &oldcmd)
					}
				}

				T.svClientThink(cl, &newcmd)
			}

			cl.lastcmd.Copy(newcmd)

		case shared.ClcStringcmd:
			s := msg.ReadString()

			/* malicious users may try using too many string commands */
			stringCmdCount++
			if stringCmdCount < maxSTRINGCMDS {
				if err := T.executeUserCommand(s); err != nil {
					return err
				}
			}

			if cl.state == cs_zombie {
				return nil /* disconnect command */
			}

		default:
			log.Printf("SV_ReadClientMessage: unknown command char\n")
			// 			 SV_DropClient(cl);
			return nil
		}
	}
	return nil
}
