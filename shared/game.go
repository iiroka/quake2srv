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
 * Here are the client, server and game are tied together.
 *
 * =======================================================================
 */
package shared

/*
 * !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
 *
 * THIS FILE IS _VERY_ FRAGILE AND THERE'S NOTHING IN IT THAT CAN OR
 * MUST BE CHANGED. IT'S MOST LIKELY A VERY GOOD IDEA TO CLOSE THE
 * EDITOR NOW AND NEVER LOOK BACK. OTHERWISE YOU MAY SCREW UP EVERYTHING!
 *
 * !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
 */

const GAME_API_VERSION = 3

const SVF_NOCLIENT = 0x00000001    /* don't send entity to clients, even if it has effects */
const SVF_DEADMONSTER = 0x00000002 /* treat as CONTENTS_DEADMONSTER for collision */
const SVF_MONSTER = 0x00000004     /* treat as CONTENTS_MONSTER for collision */

const MAX_ENT_CLUSTERS = 16

type Solid_t int

const (
	SOLID_NOT     Solid_t = 0 /* no interaction with other objects */
	SOLID_TRIGGER Solid_t = 1 /* only touch when inside, after moving */
	SOLID_BBOX    Solid_t = 2 /* touch on edge */
	SOLID_BSP     Solid_t = 3 /* bsp clip, touch on edge */
)

/* =============================================================== */

/* link_t is only used for entity area links now */
type Link_t struct {
	Prev, Next *Link_t
	Self       Edict_s
}

type Gclient_s interface {
	// player_state_t ps;      /* communicated by server to clients */
	// int ping;
	Ps() *Player_state_t
	Ping() int
	/* the game dll can add anything it wants
	after  this point in the structure */
}

type Edict_s interface {
	S() *Entity_state_t
	Client() Gclient_s
	Inuse() bool
	Linkcount() int
	SetLinkcount(v int)

	Area() *Link_t /* linked to a division node or leaf */

	NumClusters() int /* if -1, uLink_tse headnode instead */
	SetNumClusters(v int)
	Clusternums() []int
	Headnode() int /* unused if num_clusters != -1 */
	SetHeadnode(v int)
	Areanum() int
	SetAreanum(v int)
	Areanum2() int
	SetAreanum2(v int)

	Svflags() int /* SVF_NOCLIENT, SVF_DEADMONSTER, SVF_MONSTER, etc */
	Mins() []float32
	Maxs() []float32
	Absmin() []float32
	Absmax() []float32
	Size() []float32
	Solid() Solid_t
	// int clipmask;
	Owner() Edict_s
	// edict_t *owner;

	/* the game dll can add anything it wants
	after this point in the structure */
}

/* =============================================================== */

/* functions provided by the main engine */
type Game_import_t interface {
	/* special messages */
	// void (*bprintf)(int printlevel, char *fmt, ...);
	Dprintf(format string, a ...interface{})
	// void (*cprintf)(edict_t *ent, int printlevel, char *fmt, ...);
	// void (*centerprintf)(edict_t *ent, char *fmt, ...);
	// void (*sound)(edict_t *ent, int channel, int soundindex, float volume,
	// 		float attenuation, float timeofs);
	// void (*positioned_sound)(vec3_t origin, edict_t *ent, int channel,
	// 		int soundinedex, float volume, float attenuation, float timeofs);

	/* config strings hold all the index strings, the lightstyles,
	and misc data like the sky definition and cdtrack.
	All of the current configstrings are sent to clients when
	they connect, and changes are sent to all connected clients. */
	Configstring(num int, str string) error

	Error(format string, a ...interface{}) error

	/* the *index functions create configstrings
	and some internal server state */
	Modelindex(name string) int
	Soundindex(name string) int
	Imageindex(name string) int

	Setmodel(ent Edict_s, name string) error

	// /* collision detection */
	Trace(start, mins, maxs, end []float32, passent Edict_s, contentmask int) Trace_t
	Pointcontents(point []float32) int
	InPVS(p1, p2 []float32) bool
	InPHS(p1, p2 []float32) bool
	SetAreaPortalState(portalnum int, open bool)
	AreasConnected(area1, area2 int) bool

	/* an entity will never be sent to a client or used for collision
	if it is not passed to linkentity. If the size, position, or
	solidity changes, it must be relinked. */
	Linkentity(ent Edict_s)
	Unlinkentity(ent Edict_s) /* call before removing an interactive edict */
	BoxEdicts(mins, maxs []float32, edicts []Edict_s, maxcount, areatype int) int
	Pmove(pmove *Pmove_t) /* player movement code common with client prediction */

	// /* network messaging */
	Multicast(origin []float32, to Multicast_t)
	// void (*unicast)(edict_t *ent, qboolean reliable);
	// void (*WriteChar)(int c);
	WriteByte(c int)
	WriteShort(c int)
	WriteLong(c int)
	WriteFloat(c float32)
	WriteString(c string)
	WritePosition(pos []float32) /* some fractional bits */
	WriteDir(pos []float32)      /* single byte encoded, very coarse */
	// void (*WriteAngle)(float f);

	// /* managed memory allocation */
	// void *(*TagMalloc)(int size, int tag);
	// void (*TagFree)(void *block);
	// void (*FreeTags)(int tag);

	/* console variable interaction */
	Cvar(var_name, value string, flags int) *CvarT
	CvarSet(var_name, value string) *CvarT
	CvarForceSet(var_name, value string) *CvarT

	// /* ClientCommand and ServerCommand parameter access */
	// int (*argc)(void);
	// char *(*argv)(int n);
	// char *(*args)(void); /* concatenation of all argv >= 1 */

	// /* add commands to the server console as if
	//    they were typed in for map changing, etc */
	// void (*AddCommandString)(char *text);

	// void (*DebugGraph)(float value, int color);
}

/* functions exported by the game subsystem */
type Game_export_t interface {
	// int apiversion;

	/* the init function will only be called when a game starts,
	not each time a level is loaded.  Persistant data for clients
	and the server can be allocated in init */
	Init()
	Shutdown()

	/* each new level entered will cause a call to SpawnEntities */
	SpawnEntities(mapname, entstring, spawnpoint string) error

	// /* Read/Write Game is for storing persistant cross level information
	//    about the world state and the clients.
	//    WriteGame is called every time a level is exited.
	//    ReadGame is called on a loadgame. */
	// void (*WriteGame)(char *filename, qboolean autosave);
	// void (*ReadGame)(char *filename);

	// /* ReadLevel is called after the default
	//    map information has been loaded with
	//    SpawnEntities */
	// void (*WriteLevel)(char *filename);
	// void (*ReadLevel)(char *filename);

	ClientConnect(ent Edict_s, userinfo string) bool
	ClientBegin(ent Edict_s) error
	// void (*ClientUserinfoChanged)(edict_t *ent, char *userinfo);
	// void (*ClientDisconnect)(edict_t *ent);
	ClientCommand(ent Edict_s, args []string)
	ClientThink(ent Edict_s, cmd *Usercmd_t)

	RunFrame() error

	// /* ServerCommand will be called when an "sv <command>"
	//    command is issued on the  server console. The game can
	//    issue gi.argc() / gi.argv() commands to get the rest
	//    of the parameters */
	// void (*ServerCommand)(void);

	/* global variables shared between game and server */

	/* The edict array is allocated in the game dll so it
	can vary in size from one game to another.
	The size will be fixed when ge->Init() is called */
	Edict(index int) Edict_s
	NumEdicts() int /* current number, <= max_edicts */
	MaxEdicts() int
}
