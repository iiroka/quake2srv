/*
 * Copyright (C) 1997-2001 Id Software, Inc.
 * Copyright (C) 2011 Knightmare
 * Copyright (C) 2011 Yamagi Burmeister
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
 * The savegame system.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

/*
 * This is the Quake 2 savegame system, fixed by Yamagi
 * based on an idea by Knightmare of kmquake2. This major
 * rewrite of the original g_save.c is much more robust
 * and portable since it doesn't use any function pointers.
 *
 * Inner workings:
 * When the game is saved all function pointers are
 * translated into human readable function definition strings.
 * The same way all mmove_t pointers are translated. This
 * human readable strings are then written into the file.
 * At game load the human readable strings are retranslated
 * into the actual function pointers and struct pointers. The
 * pointers are generated at each compilation / start of the
 * client, thus the pointers are always correct.
 *
 * Limitations:
 * While savegames survive recompilations of the game source
 * and bigger changes in the source, there are some limitation
 * which a nearly impossible to fix without a object orientated
 * rewrite of the game.
 *  - If functions or mmove_t structs that a referencenced
 *    inside savegames are added or removed (e.g. the files
 *    in tables/ are altered) the load functions cannot
 *    reconnect all pointers and thus not restore the game.
 *  - If the operating system is changed internal structures
 *    may change in an unrepairable way.
 *  - If the architecture is changed pointer length and
 *    other internal datastructures change in an
 *    incompatible way.
 *  - If the edict_t struct is changed, savegames
 *    will break.
 * This is not so bad as it looks since functions and
 * struct won't be added and edict_t won't be changed
 * if no big, sweeping changes are done. The operating
 * system and architecture are in the hands of the user.
 */

/* ========================================================= */

/*
 * This will be called when the dll is first loaded,
 * which only happens when a new game is started or
 * a save game is loaded.
 */
func (G *qGame) Init() {
	G.gi.Dprintf("Game is starting up.\n")
	// G.gi.Dprintf("Game is %s built on %s.\n", GAMEVERSION, BUILD_DATE);

	G.gun_x = G.gi.Cvar("gun_x", "0", 0)
	G.gun_y = G.gi.Cvar("gun_y", "0", 0)
	G.gun_z = G.gi.Cvar("gun_z", "0", 0)
	G.sv_rollspeed = G.gi.Cvar("sv_rollspeed", "200", 0)
	G.sv_rollangle = G.gi.Cvar("sv_rollangle", "2", 0)
	G.sv_maxvelocity = G.gi.Cvar("sv_maxvelocity", "2000", 0)
	G.sv_gravity = G.gi.Cvar("sv_gravity", "800", 0)

	/* noset vars */
	G.dedicated = G.gi.Cvar("dedicated", "0", shared.CVAR_NOSET)

	/* latched vars */
	G.sv_cheats = G.gi.Cvar("cheats", "0", shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
	// G.gi.Cvar("gamename", GAMEVERSION, CVAR_SERVERINFO | CVAR_LATCH);
	// G.gi.Cvar("gamedate", BUILD_DATE, CVAR_SERVERINFO | CVAR_LATCH);
	G.maxclients = G.gi.Cvar("maxclients", "4", shared.CVAR_SERVERINFO|shared.CVAR_LATCH)
	G.maxspectators = G.gi.Cvar("maxspectators", "4", shared.CVAR_SERVERINFO)
	G.deathmatch = G.gi.Cvar("deathmatch", "0", shared.CVAR_LATCH)
	G.coop = G.gi.Cvar("coop", "0", shared.CVAR_LATCH)
	G.coop_pickup_weapons = G.gi.Cvar("coop_pickup_weapons", "1", shared.CVAR_ARCHIVE)
	G.coop_elevator_delay = G.gi.Cvar("coop_elevator_delay", "1.0", shared.CVAR_ARCHIVE)
	G.skill = G.gi.Cvar("skill", "1", shared.CVAR_LATCH)
	G.maxentities = G.gi.Cvar("maxentities", "1024", shared.CVAR_LATCH)
	G.g_footsteps = G.gi.Cvar("g_footsteps", "1", shared.CVAR_ARCHIVE)
	G.g_fix_triggered = G.gi.Cvar("g_fix_triggered", "0", 0)
	G.g_commanderbody_nogod = G.gi.Cvar("g_commanderbody_nogod", "0", shared.CVAR_ARCHIVE)

	println("deathmatch", G.deathmatch.String)

	/* change anytime vars */
	G.dmflags = G.gi.Cvar("dmflags", "0", shared.CVAR_SERVERINFO)
	G.fraglimit = G.gi.Cvar("fraglimit", "0", shared.CVAR_SERVERINFO)
	G.timelimit = G.gi.Cvar("timelimit", "0", shared.CVAR_SERVERINFO)
	G.password = G.gi.Cvar("password", "", shared.CVAR_USERINFO)
	G.spectator_password = G.gi.Cvar("spectator_password", "", shared.CVAR_USERINFO)
	G.needpass = G.gi.Cvar("needpass", "0", shared.CVAR_SERVERINFO)
	G.filterban = G.gi.Cvar("filterban", "1", 0)
	G.g_select_empty = G.gi.Cvar("g_select_empty", "0", shared.CVAR_ARCHIVE)
	G.run_pitch = G.gi.Cvar("run_pitch", "0.002", 0)
	G.run_roll = G.gi.Cvar("run_roll", "0.005", 0)
	G.bob_up = G.gi.Cvar("bob_up", "0.005", 0)
	G.bob_pitch = G.gi.Cvar("bob_pitch", "0.002", 0)
	G.bob_roll = G.gi.Cvar("bob_roll", "0.002", 0)

	/* flood control */
	G.flood_msgs = G.gi.Cvar("flood_msgs", "4", 0)
	G.flood_persecond = G.gi.Cvar("flood_persecond", "4", 0)
	G.flood_waitdelay = G.gi.Cvar("flood_waitdelay", "10", 0)

	/* dm map list */
	G.sv_maplist = G.gi.Cvar("sv_maplist", "", 0)

	/* others */
	G.aimfix = G.gi.Cvar("aimfix", "0", shared.CVAR_ARCHIVE)

	// /* items */
	// InitItems();

	G.game.helpmessage1 = ""
	G.game.helpmessage2 = ""

	/* initialize all entities for this game */
	G.game.maxentities = G.maxentities.Int()
	G.g_edicts = make([]edict_t, G.maxentities.Int()) //gi.TagMalloc(game.maxentities * sizeof(g_edicts[0]), TAG_GAME);
	for i := range G.g_edicts {
		G.g_edicts[i].index = i
		G.g_edicts[i].area.Self = &G.g_edicts[i]
	}

	// /* initialize all clients for this game */
	G.game.maxclients = G.maxclients.Int()
	G.game.clients = make([]gclient_t, G.game.maxclients) //gi.TagMalloc(game.maxclients*sizeof(game.clients[0]), TAG_GAME)
	G.num_edicts = G.game.maxclients + 1
}

/*
 * Fields to be saved
 */
var fields = []field_t{
	{"classname", "Classname", F_LSTRING, 0},
	{"model", "Model", F_LSTRING, 0},
	{"spawnflags", "Spawnflags", F_INT, 0},
	{"speed", "Speed", F_FLOAT, 0},
	{"accel", "Accel", F_FLOAT, 0},
	{"decel", "Decel", F_FLOAT, 0},
	{"target", "Target", F_LSTRING, 0},
	{"targetname", "Targetname", F_LSTRING, 0},
	{"pathtarget", "Pathtarget", F_LSTRING, 0},
	{"deathtarget", "Deathtarget", F_LSTRING, 0},
	{"killtarget", "Killtarget", F_LSTRING, 0},
	{"combattarget", "Combattarget", F_LSTRING, 0},
	{"message", "Message", F_LSTRING, 0},
	{"team", "Team", F_LSTRING, 0},
	{"wait", "Wait", F_FLOAT, 0},
	{"delay", "Delay", F_FLOAT, 0},
	{"random", "Random", F_FLOAT, 0},
	// {"move_origin", FOFS(move_origin), F_VECTOR},
	// {"move_angles", FOFS(move_angles), F_VECTOR},
	{"style", "Style", F_INT, 0},
	// {"count", FOFS(count), F_INT},
	{"health", "Health", F_INT, 0},
	{"sounds", "Sounds", F_INT, 0},
	{"light", "", F_IGNORE, 0},
	{"dmg", "Dmg", F_INT, 0},
	{"mass", "Mass", F_INT, 0},
	{"volume", "Volume", F_FLOAT, 0},
	{"attenuation", "Attenuation", F_FLOAT, 0},
	{"map", "Map", F_LSTRING, 0},
	{"origin", "Origin", F_VECTOR, FFL_ENTITYSTATE},
	{"angles", "Angles", F_VECTOR, FFL_ENTITYSTATE},
	{"angle", "Angles", F_ANGLEHACK, FFL_ENTITYSTATE},
	// {"goalentity", FOFS(goalentity), F_EDICT, FFL_NOSPAWN},
	// {"movetarget", FOFS(movetarget), F_EDICT, FFL_NOSPAWN},
	// {"enemy", FOFS(enemy), F_EDICT, FFL_NOSPAWN},
	// {"oldenemy", FOFS(oldenemy), F_EDICT, FFL_NOSPAWN},
	// {"activator", FOFS(activator), F_EDICT, FFL_NOSPAWN},
	// {"groundentity", FOFS(groundentity), F_EDICT, FFL_NOSPAWN},
	// {"teamchain", FOFS(teamchain), F_EDICT, FFL_NOSPAWN},
	// {"teammaster", FOFS(teammaster), F_EDICT, FFL_NOSPAWN},
	// {"owner", FOFS(owner), F_EDICT, FFL_NOSPAWN},
	// {"mynoise", FOFS(mynoise), F_EDICT, FFL_NOSPAWN},
	// {"mynoise2", FOFS(mynoise2), F_EDICT, FFL_NOSPAWN},
	// {"target_ent", FOFS(target_ent), F_EDICT, FFL_NOSPAWN},
	// {"chain", FOFS(chain), F_EDICT, FFL_NOSPAWN},
	// {"prethink", FOFS(prethink), F_FUNCTION, FFL_NOSPAWN},
	// {"think", FOFS(think), F_FUNCTION, FFL_NOSPAWN},
	// {"blocked", FOFS(blocked), F_FUNCTION, FFL_NOSPAWN},
	// {"touch", FOFS(touch), F_FUNCTION, FFL_NOSPAWN},
	// {"use", FOFS(use), F_FUNCTION, FFL_NOSPAWN},
	// {"pain", FOFS(pain), F_FUNCTION, FFL_NOSPAWN},
	// {"die", FOFS(die), F_FUNCTION, FFL_NOSPAWN},
	// {"stand", FOFS(monsterinfo.stand), F_FUNCTION, FFL_NOSPAWN},
	// {"idle", FOFS(monsterinfo.idle), F_FUNCTION, FFL_NOSPAWN},
	// {"search", FOFS(monsterinfo.search), F_FUNCTION, FFL_NOSPAWN},
	// {"walk", FOFS(monsterinfo.walk), F_FUNCTION, FFL_NOSPAWN},
	// {"run", FOFS(monsterinfo.run), F_FUNCTION, FFL_NOSPAWN},
	// {"dodge", FOFS(monsterinfo.dodge), F_FUNCTION, FFL_NOSPAWN},
	// {"attack", FOFS(monsterinfo.attack), F_FUNCTION, FFL_NOSPAWN},
	// {"melee", FOFS(monsterinfo.melee), F_FUNCTION, FFL_NOSPAWN},
	// {"sight", FOFS(monsterinfo.sight), F_FUNCTION, FFL_NOSPAWN},
	// {"checkattack", FOFS(monsterinfo.checkattack), F_FUNCTION, FFL_NOSPAWN},
	// {"currentmove", FOFS(monsterinfo.currentmove), F_MMOVE, FFL_NOSPAWN},
	// {"endfunc", FOFS(moveinfo.endfunc), F_FUNCTION, FFL_NOSPAWN},
	{"lip", "Lip", F_INT, FFL_SPAWNTEMP},
	// {"distance", STOFS(distance), F_INT, FFL_SPAWNTEMP},
	// {"height", STOFS(height), F_INT, FFL_SPAWNTEMP},
	{"noise", "Noise", F_LSTRING, FFL_SPAWNTEMP},
	// {"pausetime", STOFS(pausetime), F_FLOAT, FFL_SPAWNTEMP},
	// {"item", STOFS(item), F_LSTRING, FFL_SPAWNTEMP},
	// {"item", FOFS(item), F_ITEM},
	{"gravity", "Gravity", F_LSTRING, FFL_SPAWNTEMP},
	{"sky", "Sky", F_LSTRING, FFL_SPAWNTEMP},
	{"skyrotate", "Skyrotate", F_FLOAT, FFL_SPAWNTEMP},
	{"skyaxis", "Skyaxis", F_VECTOR, FFL_SPAWNTEMP},
	// {"minyaw", STOFS(minyaw), F_FLOAT, FFL_SPAWNTEMP},
	// {"maxyaw", STOFS(maxyaw), F_FLOAT, FFL_SPAWNTEMP},
	// {"minpitch", STOFS(minpitch), F_FLOAT, FFL_SPAWNTEMP},
	// {"maxpitch", STOFS(maxpitch), F_FLOAT, FFL_SPAWNTEMP},
	{"nextmap", "Nextmap", F_LSTRING, FFL_SPAWNTEMP},
}
