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
 * Main header file for the client
 *
 * =======================================================================
 */
package server

import "quake2srv/shared"

/* MAX_CHALLENGES is made large to prevent a denial
   of service attack that could cycle all of them
   out before legitimate users connected */
const MAX_CHALLENGES = 1024

type server_state_t int

const (
	ss_dead      server_state_t = 0 /* no map loaded */
	ss_loading   server_state_t = 1 /* spawning level edicts */
	ss_game      server_state_t = 2 /* actively running */
	ss_cinematic server_state_t = 3
	ss_demo      server_state_t = 4
	ss_pic       server_state_t = 5
)

type server_t struct {
	state server_state_t /* precache commands are only valid during load */

	attractloop bool /* running cinematics and demos for the local system only */
	loadgame    bool /* client begins should reuse existing entity */

	time     uint /* always sv.framenum * 100 msec */
	framenum int

	name   string /* map name, or cinematic name */
	models [shared.MAX_MODELS]*shared.Cmodel_t

	configstrings [shared.MAX_CONFIGSTRINGS]string
	baselines     [shared.MAX_EDICTS]shared.Entity_state_t

	/* the multicast buffer is used to send a message to a set of clients
	it is only used to marshall data until SV_Multicast is called */
	multicast *shared.QWritebuf

	/* demo server information */
	// demofile shared.QFileHandle
	// qboolean timedemo; /* don't time sync */
}

type client_state_t int

const (
	cs_free   client_state_t = 0 /* can be reused for a new connection */
	cs_zombie client_state_t = 1 /* client has been disconnected, but don't reuse
	connection for a couple seconds */
	cs_connected client_state_t = 2 /* has been assigned to a client_t, but not in game yet */
	cs_spawned   client_state_t = 3 /* client is fully in game */
)

type client_frame_t struct {
	areabytes    int
	areabits     [shared.MAX_MAP_AREAS / 8]byte /* portalarea visibility bits */
	ps           shared.Player_state_t
	num_entities int
	first_entity int /* into the circular sv_packet_entities[] */
	senttime     int /* for ping calculations */
}

type client_t struct {
	index int
	state client_state_t
	addr  string

	userinfo string /* name, etc */

	lastframe int              /* for delta compression */
	lastcmd   shared.Usercmd_t /* for filling in big drops */

	commandMsec int /* every seconds this is reset, if user */
	/* commands exhaust it, assume time cheating */

	// int frame_latency[LATENCY_COUNTS];
	ping int

	// int message_size[RATE_MESSAGES];    /* used to rate drop packets */
	rate          int
	surpressCount int /* number of messages rate supressed */

	edict        shared.Edict_s /* EDICT_NUM(clientnum+1) */
	name         string         /* extracted from userinfo, high bits masked */
	messagelevel int            /* for filtering printed messages */

	/* The datagram is written to by sound calls, prints,
	temp ents, etc. It can be harmlessly overflowed. */
	// sizebuf_t datagram;
	// byte datagram_buf[MAX_MSGLEN];

	frames [shared.UPDATE_BACKUP]client_frame_t /* updates can be delta'd from here */

	lastmessage int /* sv.framenum when packet was last received */
	lastconnect int

	challenge int /* challenge of this user, randomly generated */

	netchan shared.Netchan_t
}

type challenge_t struct {
	adr       string
	challenge int
	time      int
}

type server_static_t struct {
	initialized bool /* sv_init has completed */
	realtime    int  /* always increasing, no clamping, etc */

	mapcmd string /* ie: *intro.cin+base */

	spawncount int /* incremented each server start */
	/* used to check late spawns */

	clients              []client_t              /* [maxclients->value]; */
	num_client_entities  int                     /* maxclients->value*UPDATE_BACKUP*MAX_PACKET_ENTITIES */
	next_client_entities int                     /* next client_entity to use */
	client_entities      []shared.Entity_state_t /* [num_client_entities] */

	last_heartbeat int

	challenges [MAX_CHALLENGES]challenge_t /* to prevent invalid IPs from connecting */

	// /* serverrecord values */
	// FILE *demofile;
	// sizebuf_t demo_multicast;
	// byte demo_multicast_buf[MAX_MSGLEN];
}

type qServer struct {
	common shared.QCommon

	sv_paused              *shared.CvarT
	sv_timedemo            *shared.CvarT
	sv_enforcetime         *shared.CvarT
	timeout                *shared.CvarT /* seconds without any message */
	zombietime             *shared.CvarT /* seconds to sink messages after disconnect */
	rcon_password          *shared.CvarT /* password for remote server commands */
	allow_download         *shared.CvarT
	allow_download_players *shared.CvarT
	allow_download_models  *shared.CvarT
	allow_download_sounds  *shared.CvarT
	allow_download_maps    *shared.CvarT
	sv_airaccelerate       *shared.CvarT
	sv_noreload            *shared.CvarT /* don't reload level state when reentering */
	maxclients             *shared.CvarT /* rename sv_maxclients */
	sv_showclamp           *shared.CvarT
	hostname               *shared.CvarT
	public_server          *shared.CvarT /* should heartbeats be sent */
	sv_entfile             *shared.CvarT /* External entity files. */
	sv_downloadserver      *shared.CvarT /* Download server. */

	sv  server_t
	svs server_static_t

	sv_client *client_t

	ge shared.Game_export_t

	fatpvs [65536 / 8]byte

	sv_areanodes    [AREA_NODES]areanode_t
	sv_numareanodes int

	area_mins, area_maxs      []float32
	area_list                 []shared.Edict_s
	area_count, area_maxcount int
	area_type                 int

	sv_player shared.Edict_s
}

func CreateQServer(common shared.QCommon) shared.QServer {
	q := &qServer{}
	q.common = common
	return q
}
