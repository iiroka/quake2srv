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
 * Item spawning.
 *
 * =======================================================================
 */
package game

import (
	"fmt"
	"math"
	"quake2srv/shared"
	"reflect"
	"strconv"
)

var spawns = map[string]func(ent *edict_t, G *qGame) error{
	"item_health":            spItemHealth,
	"item_health_small":      spItemHealthSmall,
	"item_health_large":      spItemHealthLarge,
	"item_health_mega":       spItemHealthMega,
	"info_player_start":      spInfoPlayerStart,
	"info_player_deathmatch": spInfoPlayerDeathmatch,
	"func_door":              spFuncDoor,
	"func_timer":             spFuncTimer,
	"trigger_always":         spTriggerAlways,
	"trigger_once":           spTriggerOnce,
	"trigger_multiple":       spTriggerMultiple,
	"trigger_relay":          spTriggerRelay,
	"target_speaker":         spTargetSpeaker,
	"target_explosion":       spTargetExplosion,
	"target_help":            spTargetHelp,
	"worldspawn":             spWorldspawn,
	"light":                  spLight,
	"path_corner":            spPathCorner,
	"point_combat":           spPointCombat,
	"misc_explobox":          spMiscExplobox,
	"misc_deadsoldier":       spMiscDeadsoldier,
	"misc_teleporter_dest":   spMiscTeleporterDest,
	"monster_soldier":        spMonsterSoldier,
}

/*
 * Finds the spawn function for
 * the entity and calls it
 */
func (G *qGame) edCallSpawn(ent *edict_t) error {

	if ent == nil {
		return nil
	}

	if len(ent.Classname) == 0 {
		G.gi.Dprintf("ED_CallSpawn: NULL classname\n")
		G.gFreeEdict(ent)
		return nil
	}

	/* check item spawn functions */
	for i, item := range gameitemlist {
		if len(item.classname) == 0 {
			continue
		}

		if item.classname == ent.Classname {
			/* found it */
			G.spawnItem(ent, &gameitemlist[i])
			return nil
		}
	}

	/* check normal spawn functions */
	if s, ok := spawns[ent.Classname]; ok {
		/* found it */
		return s(ent, G)
	}

	G.gi.Dprintf("%s doesn't have a spawn function\n", ent.Classname)
	return nil
}

/*
 * Takes a key/value pair and sets
 * the binary values in an edict
 */
func (G *qGame) edParseField(key, value string, ent *edict_t) {

	for _, f := range fields {
		if (f.flags&FFL_NOSPAWN) == 0 && f.name == key {
			/* found it */

			var b reflect.Value
			if (f.flags & FFL_ENTITYSTATE) != 0 {
				b = reflect.ValueOf(&ent.s).Elem()
			} else if (f.flags & FFL_SPAWNTEMP) != 0 {
				b = reflect.ValueOf(&G.st).Elem()
			} else {
				b = reflect.ValueOf(ent).Elem()
			}

			switch f.ftype {
			case F_LSTRING:
				b.FieldByName(f.fname).SetString(value)
			case F_VECTOR:
				var vect [3]float32
				fmt.Sscanf(value, "%f %f %f", &vect[0], &vect[1], &vect[2])
				tgt := b.FieldByName(f.fname)
				tgt.Index(0).SetFloat(float64(vect[0]))
				tgt.Index(1).SetFloat(float64(vect[1]))
				tgt.Index(2).SetFloat(float64(vect[2]))
			case F_INT:
				v, _ := strconv.ParseInt(value, 10, 32)
				b.FieldByName(f.fname).SetInt(v)
			case F_FLOAT:
				v, _ := strconv.ParseFloat(value, 32)
				b.FieldByName(f.fname).SetFloat(v)
			case F_ANGLEHACK:
				v, _ := strconv.ParseFloat(value, 32)
				tgt := b.FieldByName(f.fname)
				tgt.Index(0).SetFloat(float64(0))
				tgt.Index(1).SetFloat(float64(v))
				tgt.Index(2).SetFloat(float64(0))
			case F_IGNORE:
			default:
			}

			return
		}
	}

	G.gi.Dprintf("%s is not a field\n", key)
}

/*
 * Parses an edict out of the given string,
 * returning the new position ed should be
 * a properly initialized empty edict.
 */
func (G *qGame) edParseEdict(data string, index int, ent *edict_t) (int, error) {

	if ent == nil {
		return -1, nil
	}

	init := false
	G.st = spawn_temp_t{}

	/* go through all the dictionary pairs */
	for {
		/* parse key */
		var token string
		token, index = shared.COM_Parse(data, index)

		if token[0] == '}' {
			break
		}

		if index < 0 {
			return -1, G.gi.Error("ED_ParseEntity: EOF without closing brace")
		}

		keyname := string(token)

		/* parse value */
		token, index = shared.COM_Parse(data, index)

		if index < 0 {
			return -1, G.gi.Error("ED_ParseEntity: EOF without closing brace")
		}

		if token[0] == '}' {
			return -1, G.gi.Error("ED_ParseEntity: closing brace without data")
		}

		init = true

		/* keynames with a leading underscore are
		used for utility comments, and are
		immediately discarded by quake */
		if keyname[0] == '_' {
			continue
		}

		G.edParseField(keyname, token, ent)
	}

	if !init {
		ent.copy(edict_t{})
	}

	return index, nil
}

/*
 * Chain together all entities with a matching team field.
 *
 * All but the first will have the FL_TEAMSLAVE flag set.
 * All but the last will have the teamchain field set to the next one
 */
func (G *qGame) gFindTeams() {

	c := 0
	c2 := 0

	for i := 1; i < G.num_edicts; i++ {
		e := &G.g_edicts[i]
		if !e.inuse {
			continue
		}

		if len(e.Team) == 0 {
			continue
		}

		if (e.flags & FL_TEAMSLAVE) != 0 {
			continue
		}

		chain := e
		e.teammaster = e
		c++
		c2++

		for j := i + 1; j < G.num_edicts; j++ {
			e2 := &G.g_edicts[j]
			if !e2.inuse {
				continue
			}

			if len(e2.Team) == 0 {
				continue
			}

			if (e2.flags & FL_TEAMSLAVE) != 0 {
				continue
			}

			if e.Team == e2.Team {
				c2++
				chain.teamchain = e2
				e2.teammaster = e
				chain = e2
				e2.flags |= FL_TEAMSLAVE
			}
		}
	}

	G.gi.Dprintf("%v teams with %v entities.\n", c, c2)
}

/*
 * Creates a server's entity / program execution context by
 * parsing textual entity definitions out of an ent file.
 */
func (G *qGame) SpawnEntities(mapname, entities, spawnpoint string) error {
	//  edict_t *ent;
	//  int inhibit;
	//  const char *com_token;
	//  int i;
	//  float skill_level;
	//  static qboolean monster_count_city2 = false;
	//  static qboolean monster_count_city3 = false;
	//  static qboolean monster_count_cool1 = false;
	//  static qboolean monster_count_lab = false;

	//  if (!mapname || !entities || !spawnpoint)
	//  {
	// 	 return;
	//  }

	skill_level := math.Floor(float64(G.skill.Float()))

	if skill_level < 0 {
		skill_level = 0
	}

	if skill_level > 3 {
		skill_level = 3
	}

	if float64(G.skill.Float()) != skill_level {
		//  gi.cvar_forceset("skill", va("%f", skill_level));
	}

	G.saveClientData()

	//  gi.FreeTags(TAG_LEVEL);

	G.level = level_locals_t{}
	G.g_edicts = make([]edict_t, G.maxentities.Int())
	for i := range G.g_edicts {
		G.g_edicts[i].index = i
	}

	G.level.mapname = mapname
	G.game.spawnpoint = spawnpoint

	/* set client fields on player ents */
	for i := 0; i < G.game.maxclients; i++ {
		G.g_edicts[i+1].client = &G.game.clients[i]
	}

	var ent *edict_t
	inhibit := 0

	/* parse ents */
	index := 0
	var err error
	for index >= 0 && index < len(entities) {
		/* parse the opening brace */
		var token string
		token, index = shared.COM_Parse(entities, index)
		if index < 0 {
			break
		}

		if token[0] != '{' {
			return G.gi.Error("ED_LoadFromFile: found %s when expecting {", token)
		}

		if ent == nil {
			ent = &G.g_edicts[0]
		} else {
			ent, err = G.gSpawn()
			if err != nil {
				return err
			}
		}

		index, err = G.edParseEdict(entities, index, ent)
		if err != nil {
			return err
		}

		// 	 /* yet another map hack */
		// 	 if (!Q_stricmp(level.mapname, "command") &&
		// 		 !Q_stricmp(ent->classname, "trigger_once") &&
		// 			!Q_stricmp(ent->model, "*27")) {
		// 		 ent->spawnflags &= ~SPAWNFLAG_NOT_HARD;
		// 	 }

		/*
		 * The 'monsters' count in city3.bsp is wrong.
		 * There're two monsters triggered in a hidden
		 * and unreachable room next to the security
		 * pass.
		 *
		 * We need to make sure that this hack is only
		 * applied once!
		 */
		// 	 if (!Q_stricmp(level.mapname, "city3") && !monster_count_city3)
		// 	 {
		// 		 level.total_monsters = level.total_monsters - 2;
		// 		 monster_count_city3 = true;
		// 	 }

		/* A slightly other problem in city2.bsp. There's a floater
		 * with missing trigger on the right gallery above the data
		 * spinner console, right before the door to the staircase.
		 */
		// 	 if ((skill->value > 0) && !Q_stricmp(level.mapname, "city2") && !monster_count_city2)
		// 	 {
		// 		 level.total_monsters = level.total_monsters - 1;
		// 		 monster_count_city2 = true;
		// 	 }

		/*
		 * Nearly the same problem exists in cool1.bsp.
		 * On medium skill a gladiator is spawned in a
		 * crate that's never triggered.
		 */
		// 	 if ((skill->value == 1) && !Q_stricmp(level.mapname, "cool1") && !monster_count_cool1)
		// 	 {
		// 		 level.total_monsters = level.total_monsters - 1;
		// 		 monster_count_cool1 = true;
		// 	 }

		/*
		 * Nearly the same problem exists in lab.bsp.
		 * On medium skill two parasites are spawned
		 * in a hidden place that never triggers.
		 */
		// 	 if ((skill->value == 1) && !Q_stricmp(level.mapname, "lab") && !monster_count_lab)
		// 	 {
		// 		 level.total_monsters = level.total_monsters - 2;
		// 		 monster_count_lab = true;
		// 	 }

		/* remove things (except the world) from
		different skill levels or deathmatch */
		if ent != &G.g_edicts[0] {
			if G.deathmatch.Bool() {
				if (ent.Spawnflags & SPAWNFLAG_NOT_DEATHMATCH) != 0 {
					G.gFreeEdict(ent)
					inhibit++
					continue
				}
			} else {
				if ((G.skill.Int() == SKILL_EASY) &&
					(ent.Spawnflags&SPAWNFLAG_NOT_EASY) != 0) ||
					((G.skill.Int() == SKILL_MEDIUM) &&
						(ent.Spawnflags&SPAWNFLAG_NOT_MEDIUM) != 0) ||
					(((G.skill.Int() == SKILL_HARD) ||
						(G.skill.Int() == SKILL_HARDPLUS)) &&
						(ent.Spawnflags&SPAWNFLAG_NOT_HARD) != 0) {
					G.gFreeEdict(ent)
					inhibit++
					continue
				}
			}

			ent.Spawnflags &^=
				(SPAWNFLAG_NOT_EASY | SPAWNFLAG_NOT_MEDIUM |
					SPAWNFLAG_NOT_HARD |
					SPAWNFLAG_NOT_COOP | SPAWNFLAG_NOT_DEATHMATCH)
		}

		if err := G.edCallSpawn(ent); err != nil {
			return err
		}
	}

	G.gi.Dprintf("%v entities inhibited.\n", inhibit)

	G.gFindTeams()

	//  PlayerTrail_Init();
	return nil
}

/* =================================================================== */

const single_statusbar = "yb	-24 " +

	/* health */
	"xv	0 " +
	"hnum " +
	"xv	50 " +
	"pic 0 " +

	/* ammo */
	"if 2 " +
	"	xv	100 " +
	"	anum " +
	"	xv	150 " +
	"	pic 2 " +
	"endif " +

	/* armor */
	"if 4 " +
	"	xv	200 " +
	"	rnum " +
	"	xv	250 " +
	"	pic 4 " +
	"endif " +

	/* selected item */
	"if 6 " +
	"	xv	296 " +
	"	pic 6 " +
	"endif " +

	"yb	-50 " +

	/* picked up item */
	"if 7 " +
	"	xv	0 " +
	"	pic 7 " +
	"	xv	26 " +
	"	yb	-42 " +
	"	stat_string 8 " +
	"	yb	-50 " +
	"endif " +

	/* timer */
	"if 9 " +
	"	xv	262 " +
	"	num	2	10 " +
	"	xv	296 " +
	"	pic	9 " +
	"endif " +

	/*  help / weapon icon */
	"if 11 " +
	"	xv	148 " +
	"	pic	11 " +
	"endif "

const dm_statusbar = "yb	-24 " +

	/* health */
	"xv	0 " +
	"hnum " +
	"xv	50 " +
	"pic 0 " +

	/* ammo */
	"if 2 " +
	"	xv	100 " +
	"	anum " +
	"	xv	150 " +
	"	pic 2 " +
	"endif " +

	/* armor */
	"if 4 " +
	"	xv	200 " +
	"	rnum " +
	"	xv	250 " +
	"	pic 4 " +
	"endif " +

	/* selected item */
	"if 6 " +
	"	xv	296 " +
	"	pic 6 " +
	"endif " +

	"yb	-50 " +

	/* picked up item */
	"if 7 " +
	"	xv	0 " +
	"	pic 7 " +
	"	xv	26 " +
	"	yb	-42 " +
	"	stat_string 8 " +
	"	yb	-50 " +
	"endif " +

	/* timer */
	"if 9 " +
	"	xv	246 " +
	"	num	2	10 " +
	"	xv	296 " +
	"	pic	9 " +
	"endif " +

	/*  help / weapon icon */
	"if 11 " +
	"	xv	148 " +
	"	pic	11 " +
	"endif " +

	/*  frags */
	"xr	-50 " +
	"yt 2 " +
	"num 3 14 " +

	/* spectator */
	"if 17 " +
	"xv 0 " +
	"yb -58 " +
	"string2 \"SPECTATOR MODE\" " +
	"endif " +

	/* chase camera */
	"if 16 " +
	"xv 0 " +
	"yb -68 " +
	"string \"Chasing\" " +
	"xv 64 " +
	"stat_string 16 " +
	"endif "

/*QUAKED worldspawn (0 0 0) ?
 *
 * Only used for the world.
 *  "sky"		environment map name
 *  "skyaxis"	vector axis for rotating sky
 *  "skyrotate"	speed of rotation in degrees/second
 *  "sounds"	music cd track number
 *  "gravity"	800 is default gravity
 *  "message"	text to print at user logon
 */
func spWorldspawn(ent *edict_t, G *qGame) error {
	if ent == nil {
		return nil
	}

	ent.movetype = MOVETYPE_PUSH
	ent.solid = shared.SOLID_BSP
	ent.inuse = true     /* since the world doesn't use G_Spawn() */
	ent.s.Modelindex = 1 /* world model is always index 1 */

	/* --------------- */

	/* reserve some spots for dead
	player bodies for coop / deathmatch */
	//  InitBodyQue();

	/* set configstrings for items */
	G.setItemNames()

	if len(G.st.Nextmap) > 0 {
		G.level.nextmap = string(G.st.Nextmap)
	}

	/* make some data visible to the server */
	if len(ent.Message) > 0 {
		G.gi.Configstring(shared.CS_NAME, ent.Message)
		G.level.level_name = string(ent.Message)
	} else {
		G.level.level_name = string(G.level.mapname)
	}

	if len(G.st.Sky) > 0 {
		G.gi.Configstring(shared.CS_SKY, G.st.Sky)
	} else {
		G.gi.Configstring(shared.CS_SKY, "unit1_")
	}

	G.gi.Configstring(shared.CS_SKYROTATE, fmt.Sprintf("%f", G.st.Skyrotate))

	G.gi.Configstring(shared.CS_SKYAXIS, fmt.Sprintf("%f %f %f",
		G.st.Skyaxis[0], G.st.Skyaxis[1], G.st.Skyaxis[2]))

	//  gi.configstring(CS_CDTRACK, va("%i", ent->sounds));

	G.gi.Configstring(shared.CS_MAXCLIENTS, fmt.Sprintf("%v", G.maxclients.Int()))

	/* status bar program */
	if G.deathmatch.Bool() {
		G.gi.Configstring(shared.CS_STATUSBAR, dm_statusbar)
	} else {
		G.gi.Configstring(shared.CS_STATUSBAR, single_statusbar)
	}

	/* --------------- */

	/* help icon for statusbar */
	//  gi.imageindex("i_help");
	G.level.pic_health = G.gi.Imageindex("i_health")
	//  gi.imageindex("help");
	//  gi.imageindex("field_3");

	// if !G.st.Gravity {
	G.gi.CvarSet("sv_gravity", "800")
	// } else {
	// 	G.gi.CvarSet("sv_gravity", st.gravity)
	// }

	//  snd_fry = gi.soundindex("player/fry.wav"); /* standing in lava / slime */

	//  PrecacheItem(FindItem("Blaster"));

	G.gi.Soundindex("player/lava1.wav")
	G.gi.Soundindex("player/lava2.wav")

	G.gi.Soundindex("misc/pc_up.wav")
	G.gi.Soundindex("misc/talk1.wav")

	G.gi.Soundindex("misc/udeath.wav")

	/* gibs */
	G.gi.Soundindex("items/respawn1.wav")

	/* sexed sounds */
	G.gi.Soundindex("*death1.wav")
	G.gi.Soundindex("*death2.wav")
	G.gi.Soundindex("*death3.wav")
	G.gi.Soundindex("*death4.wav")
	G.gi.Soundindex("*fall1.wav")
	G.gi.Soundindex("*fall2.wav")
	G.gi.Soundindex("*gurp1.wav") /* drowning damage */
	G.gi.Soundindex("*gurp2.wav")
	G.gi.Soundindex("*jump1.wav") /* player jump */
	G.gi.Soundindex("*pain25_1.wav")
	G.gi.Soundindex("*pain25_2.wav")
	G.gi.Soundindex("*pain50_1.wav")
	G.gi.Soundindex("*pain50_2.wav")
	G.gi.Soundindex("*pain75_1.wav")
	G.gi.Soundindex("*pain75_2.wav")
	G.gi.Soundindex("*pain100_1.wav")
	G.gi.Soundindex("*pain100_2.wav")

	/* sexed models: THIS ORDER MUST MATCH THE DEFINES IN g_local.h
	you can add more, max 19 (pete change)these models are only
	loaded in coop or deathmatch. not singleplayer. */
	if G.coop.Bool() || G.deathmatch.Bool() {
		G.gi.Modelindex("#w_blaster.md2")
		G.gi.Modelindex("#w_shotgun.md2")
		G.gi.Modelindex("#w_sshotgun.md2")
		G.gi.Modelindex("#w_machinegun.md2")
		G.gi.Modelindex("#w_chaingun.md2")
		G.gi.Modelindex("#a_grenades.md2")
		G.gi.Modelindex("#w_glauncher.md2")
		G.gi.Modelindex("#w_rlauncher.md2")
		G.gi.Modelindex("#w_hyperblaster.md2")
		G.gi.Modelindex("#w_railgun.md2")
		G.gi.Modelindex("#w_bfg.md2")
	}

	/* ------------------- */

	G.gi.Soundindex("player/gasp1.wav") /* gasping for air */
	G.gi.Soundindex("player/gasp2.wav") /* head breaking surface, not gasping */

	G.gi.Soundindex("player/watr_in.wav")  /* feet hitting water */
	G.gi.Soundindex("player/watr_out.wav") /* feet leaving water */

	G.gi.Soundindex("player/watr_un.wav") /* head going underwater */

	G.gi.Soundindex("player/u_breath1.wav")
	G.gi.Soundindex("player/u_breath2.wav")

	G.gi.Soundindex("items/pkup.wav")   /* bonus item pickup */
	G.gi.Soundindex("world/land.wav")   /* landing thud */
	G.gi.Soundindex("misc/h2ohit1.wav") /* landing splash */

	G.gi.Soundindex("items/damage.wav")
	G.gi.Soundindex("items/protect.wav")
	G.gi.Soundindex("items/protect4.wav")
	G.gi.Soundindex("weapons/noammo.wav")

	G.gi.Soundindex("infantry/inflies1.wav")

	//  sm_meat_index = gi.modelindex("models/objects/gibs/sm_meat/tris.md2");
	G.gi.Modelindex("models/objects/gibs/arm/tris.md2")
	G.gi.Modelindex("models/objects/gibs/bone/tris.md2")
	G.gi.Modelindex("models/objects/gibs/bone2/tris.md2")
	G.gi.Modelindex("models/objects/gibs/chest/tris.md2")
	G.gi.Modelindex("models/objects/gibs/skull/tris.md2")
	G.gi.Modelindex("models/objects/gibs/head2/tris.md2")

	/* Setup light animation tables. 'a'
	is total darkness, 'z' is doublebright. */

	/* 0 normal */
	G.gi.Configstring(shared.CS_LIGHTS+0, "m")

	/* 1 FLICKER (first variety) */
	G.gi.Configstring(shared.CS_LIGHTS+1, "mmnmmommommnonmmonqnmmo")

	/* 2 SLOW STRONG PULSE */
	G.gi.Configstring(shared.CS_LIGHTS+2, "abcdefghijklmnopqrstuvwxyzyxwvutsrqponmlkjihgfedcba")

	/* 3 CANDLE (first variety) */
	G.gi.Configstring(shared.CS_LIGHTS+3, "mmmmmaaaaammmmmaaaaaabcdefgabcdefg")

	/* 4 FAST STROBE */
	G.gi.Configstring(shared.CS_LIGHTS+4, "mamamamamama")

	/* 5 GENTLE PULSE 1 */
	G.gi.Configstring(shared.CS_LIGHTS+5, "jklmnopqrstuvwxyzyxwvutsrqponmlkj")

	/* 6 FLICKER (second variety) */
	G.gi.Configstring(shared.CS_LIGHTS+6, "nmonqnmomnmomomno")

	/* 7 CANDLE (second variety) */
	G.gi.Configstring(shared.CS_LIGHTS+7, "mmmaaaabcdefgmmmmaaaammmaamm")

	/* 8 CANDLE (third variety) */
	G.gi.Configstring(shared.CS_LIGHTS+8, "mmmaaammmaaammmabcdefaaaammmmabcdefmmmaaaa")

	/* 9 SLOW STROBE (fourth variety) */
	G.gi.Configstring(shared.CS_LIGHTS+9, "aaaaaaaazzzzzzzz")

	/* 10 FLUORESCENT FLICKER */
	G.gi.Configstring(shared.CS_LIGHTS+10, "mmamammmmammamamaaamammma")

	/* 11 SLOW PULSE NOT FADE TO BLACK */
	G.gi.Configstring(shared.CS_LIGHTS+11, "abcdefghijklmnopqrrqponmlkjihgfedcba")

	/* styles 32-62 are assigned by the light program for switchable lights */

	/* 63 testing */
	G.gi.Configstring(shared.CS_LIGHTS+63, "a")
	return nil
}
