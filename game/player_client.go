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
 * Interface between client <-> game and client calculations.
 *
 * =======================================================================
 */
package game

import (
	"fmt"
	"quake2srv/shared"
	"strconv"
)

/* ======================================================================= */

/*
 * This is only called when the game first
 * initializes in single player, but is called
 * after each death and level change in deathmatch
 */
func (G *qGame) initClientPersistant(client *gclient_t) {
	if client == nil {
		return
	}

	client.pers.copy(client_persistant_t{})

	item := G.findItem("Blaster")
	client.pers.selected_item = G.findItemIndex("Blaster")
	client.pers.inventory[client.pers.selected_item] = 1

	client.pers.weapon = item

	client.pers.health = 100
	client.pers.max_health = 100

	client.pers.max_bullets = 200
	client.pers.max_shells = 100
	client.pers.max_rockets = 50
	client.pers.max_grenades = 50
	client.pers.max_cells = 200
	client.pers.max_slugs = 50

	client.pers.connected = true
}

/* ======================================================================= */

/*
 * Returns the distance to the
 * nearest player from the given spot
 */
func (G *qGame) playersRangeFromSpot(spot *edict_t) float32 {
	//  edict_t *player;
	//  float bestplayerdistance;
	//  vec3_t v;
	//  int n;
	//  float playerdistance;

	if spot == nil {
		return 0
	}

	var bestplayerdistance float32 = 9999999

	v := make([]float32, 3)
	for n := 1; n <= G.maxclients.Int(); n++ {
		player := &G.g_edicts[n]

		if !player.inuse {
			continue
		}

		if player.Health <= 0 {
			continue
		}

		shared.VectorSubtract(spot.s.Origin[:], player.s.Origin[:], v)
		playerdistance := shared.VectorLength(v)

		if playerdistance < bestplayerdistance {
			bestplayerdistance = playerdistance
		}
	}

	return bestplayerdistance
}

/*
 * go to a random point, but NOT the two
 * points closest to other players
 */
func (G *qGame) selectRandomDeathmatchSpawnPoint() *edict_t {
	//  edict_t *spot, *spot1, *spot2;
	//  int count = 0;
	//  int selection;
	//  float range, range1, range2;

	var spot *edict_t = nil
	var spot1 *edict_t = nil
	var spot2 *edict_t = nil
	//  spot = NULL;
	//  range1 = range2 = 99999;
	//  spot1 = spot2 = NULL;

	count := 0
	var range1 float32 = 99999
	var range2 float32 = 99999
	for {
		spot = G.gFind(spot, "Classname", "info_player_deathmatch")
		if spot == nil {
			break
		}

		count += 1
		r := G.playersRangeFromSpot(spot)

		if r < range1 {
			range1 = r
			spot1 = spot
		} else if r < range2 {
			range2 = r
			spot2 = spot
		}
	}

	if count == 0 {
		return nil
	}

	if count <= 2 {
		spot1 = nil
		spot2 = nil
	} else {
		if spot1 != nil {
			count--
		}

		if spot2 != nil {
			count--
		}
	}

	selection := shared.Randk() % count

	spot = nil

	for {
		spot = G.gFind(spot, "Classname", "info_player_deathmatch")

		if (spot == spot1) || (spot == spot2) {
			selection++
		}
		if selection <= 0 {
			break
		}
		selection--
	}

	return spot
}

func (G *qGame) selectFarthestDeathmatchSpawnPoint() *edict_t {

	var spot *edict_t = nil
	var bestspot *edict_t = nil
	var bestdistance float32 = 0

	for {
		spot = G.gFind(spot, "Classname", "info_player_deathmatch")
		if spot == nil {
			break
		}
		bestplayerdistance := G.playersRangeFromSpot(spot)

		if bestplayerdistance > bestdistance {
			bestspot = spot
			bestdistance = bestplayerdistance
		}
	}

	if bestspot != nil {
		return bestspot
	}

	/* if there is a player just spawned on each and every start spot/
	we have no choice to turn one into a telefrag meltdown */
	spot = G.gFind(nil, "Classname", "info_player_deathmatch")

	return spot
}

func (G *qGame) selectDeathmatchSpawnPoint() *edict_t {
	if (G.dmflags.Int() & shared.DF_SPAWN_FARTHEST) != 0 {
		return G.selectFarthestDeathmatchSpawnPoint()
	} else {
		return G.selectRandomDeathmatchSpawnPoint()
	}
}

/*
 * Chooses a player start, deathmatch start, coop start, etc
 */
func (G *qGame) selectSpawnPoint(ent *edict_t, origin, angles []float32) error {
	//  edict_t *spot = NULL;
	//  edict_t *coopspot = NULL;
	//  int index;
	//  int counter = 0;
	//  vec3_t d;

	if ent == nil {
		return nil
	}

	var spot *edict_t = nil
	if G.deathmatch.Bool() {
		spot = G.selectDeathmatchSpawnPoint()
	} else if G.coop.Bool() {
		// 	 spot = SelectCoopSpawnPoint(ent);
	}

	/* find a single player start spot */
	if spot == nil {
		for {
			spot = G.gFind(spot, "Classname", "info_player_start")
			if spot == nil {
				break
			}
			if len(G.game.spawnpoint) == 0 && len(spot.Targetname) == 0 {
				break
			}

			if len(G.game.spawnpoint) == 0 || len(spot.Targetname) == 0 {
				continue
			}

			if G.game.spawnpoint == spot.Targetname {
				break
			}
		}

		if spot == nil {
			if len(G.game.spawnpoint) == 0 {
				/* there wasn't a spawnpoint without a target, so use any */
				spot = G.gFind(spot, "Classname", "info_player_start")
			}

			if spot == nil {
				return G.gi.Error("Couldn't find spawn point %s\n", G.game.spawnpoint)
			}
		}
	}

	/* If we are in coop and we didn't find a coop
	spawnpoint due to map bugs (not correctly
	connected or the map was loaded via console
	and thus no previously map is known to the
	client) use one in 550 units radius. */
	//  if (coop->value) {
	// 	 index = ent->client - game.clients;

	// 	 if (Q_stricmp(spot->classname, "info_player_start") == 0 && index != 0) {
	// 		 while(counter < 3)
	// 		 {
	// 			 coopspot = G_Find(coopspot, FOFS(classname), "info_player_coop");

	// 			 if (!coopspot)
	// 			 {
	// 				 break;
	// 			 }

	// 			 VectorSubtract(coopspot->s.origin, spot->s.origin, d);

	// 			 if ((VectorLength(d) < 550))
	// 			 {
	// 				 if (index == counter)
	// 				 {
	// 					 spot = coopspot;
	// 					 break;
	// 				 }
	// 				 else
	// 				 {
	// 					 counter++;
	// 				 }
	// 			 }
	// 		 }
	// 	 }
	//  }

	copy(origin, spot.s.Origin[:])
	origin[2] += 9
	copy(angles, spot.s.Angles[:])
	return nil
}

/* ============================================================== */

/*
 * Called when a player connects to
 * a server or respawns in a deathmatch.
 */
func (G *qGame) putClientInServer(ent *edict_t) error {
	//  char userinfo[MAX_INFO_STRING];

	if ent == nil {
		return nil
	}

	mins := []float32{-16, -16, -24}
	maxs := []float32{16, 16, 32}
	//  int index;
	//  gclient_t *client;
	//  int i;
	//  client_persistant_t saved;
	//  client_respawn_t resp;

	/* find a spawn point do it before setting
	health back up, so farthest ranging
	doesn't count this client */
	spawn_origin := make([]float32, 3)
	spawn_angles := make([]float32, 3)
	if err := G.selectSpawnPoint(ent, spawn_origin, spawn_angles); err != nil {
		return err
	}

	index := ent.index - 1
	client := ent.client

	resp := client_respawn_t{}
	/* deathmatch wipes most client data every spawn */
	if G.deathmatch.Bool() {
		resp.copy(client.resp)
		userinfo := string(client.pers.userinfo)
		G.initClientPersistant(client)
		G.clientUserinfoChanged(ent, userinfo)
	}
	//  else if (coop->value)
	//  {
	// 	 resp = client->resp;
	// 	 memcpy(userinfo, client->pers.userinfo, sizeof(userinfo));
	// 	 resp.coop_respawn.game_helpchanged = client->pers.game_helpchanged;
	// 	 resp.coop_respawn.helpchanged = client->pers.helpchanged;
	// 	 client->pers = resp.coop_respawn;
	// 	 ClientUserinfoChanged(ent, userinfo);

	// 	 if (resp.score > client->pers.score)
	// 	 {
	// 		 client->pers.score = resp.score;
	// 	 }
	//  }

	userinfo := string(client.pers.userinfo)
	G.clientUserinfoChanged(ent, userinfo)

	/* clear everything but the persistant data */
	var saved client_persistant_t
	saved.copy(client.pers)
	client.copy(gclient_t{})
	client.pers.copy(saved)

	if client.pers.health <= 0 {
		G.initClientPersistant(client)
	}

	client.resp = resp

	/* copy some data from the client to the entity */
	G.fetchClientEntData(ent)

	/* clear entity values */
	ent.groundentity = nil
	ent.client = &G.game.clients[index]
	ent.takedamage = DAMAGE_AIM
	ent.movetype = MOVETYPE_WALK
	ent.viewheight = 22
	ent.inuse = true
	ent.Classname = "player"
	ent.Mass = 200
	ent.solid = shared.SOLID_BBOX
	ent.deadflag = DEAD_NO
	//  ent->air_finished = level.time + 12;
	ent.clipmask = shared.MASK_PLAYERSOLID
	ent.Model = "players/male/tris.md2"
	//  ent->pain = player_pain;
	//  ent->die = player_die;
	ent.waterlevel = 0
	ent.watertype = 0
	ent.flags &^= FL_NO_KNOCKBACK
	ent.svflags = 0

	copy(ent.mins[:], mins)
	copy(ent.maxs[:], maxs)
	copy(ent.velocity[:], []float32{0, 0, 0})

	/* clear playerstate values */
	client.ps.Copy(shared.Player_state_t{})

	client.ps.Pmove.Origin[0] = int16(spawn_origin[0] * 8)
	client.ps.Pmove.Origin[1] = int16(spawn_origin[1] * 8)
	client.ps.Pmove.Origin[2] = int16(spawn_origin[2] * 8)

	if G.deathmatch.Bool() && (G.dmflags.Int()&shared.DF_FIXED_FOV) != 0 {
		client.ps.Fov = 90
	} else {
		fv, _ := strconv.ParseInt(shared.Info_ValueForKey(client.pers.userinfo, "fov"), 10, 32)
		client.ps.Fov = float32(fv)
		if client.ps.Fov < 1 {
			client.ps.Fov = 90
		} else if client.ps.Fov > 160 {
			client.ps.Fov = 160
		}
	}

	client.ps.Gunindex = G.gi.Modelindex(client.pers.weapon.view_model)

	/* clear entity state values */
	ent.s.Effects = 0
	ent.s.Modelindex = 255  /* will use the skin specified model */
	ent.s.Modelindex2 = 255 /* custom gun model */

	/* sknum is player num and weapon number
	weapon number will be added in changeweapon */
	ent.s.Skinnum = ent.index - 1

	ent.s.Frame = 0
	copy(ent.s.Origin[:], spawn_origin)
	ent.s.Origin[2] += 1 /* make sure off ground */
	copy(ent.s.Old_origin[:], ent.s.Origin[:])

	/* set the delta angle */
	for i := 0; i < 3; i++ {
		client.ps.Pmove.Delta_angles[i] = shared.ANGLE2SHORT(
			spawn_angles[i] - client.resp.cmd_angles[i])
	}

	ent.s.Angles[shared.PITCH] = 0
	ent.s.Angles[shared.YAW] = spawn_angles[shared.YAW]
	ent.s.Angles[shared.ROLL] = 0
	copy(client.ps.Viewangles[:], ent.s.Angles[:])
	copy(client.v_angle[:], ent.s.Angles[:])

	//  /* spawn a spectator */
	//  if (client->pers.spectator)
	//  {
	// 	 client->chase_target = NULL;

	// 	 client->resp.spectator = true;

	// 	 ent->movetype = MOVETYPE_NOCLIP;
	// 	 ent->solid = SOLID_NOT;
	// 	 ent->svflags |= SVF_NOCLIENT;
	// 	 ent->client->ps.gunindex = 0;
	// 	 gi.linkentity(ent);
	// 	 return;
	//  }
	//  else
	//  {
	client.resp.spectator = false
	//  }

	//  if (!KillBox(ent))
	//  {
	// 	 /* could't spawn in? */
	//  }

	G.gi.Linkentity(ent)

	/* force the current weapon up */
	client.newweapon = client.pers.weapon
	G.changeWeapon(ent)
	return nil
}

/*
 * Some maps have no unnamed (e.g. generic)
 * info_player_start. This is no problem in
 * normal gameplay, but if the map is loaded
 * via console there is a huge chance that
 * the player will spawn in the wrong point.
 * Therefore create an unnamed info_player_start
 * at the correct point.
 */
func spCreateUnnamedSpawn(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	spot, _ := G.gSpawn()

	/* mine1 */
	if G.level.mapname == "mine1" {
		if self.Targetname == "mintro" {
			spot.Classname = self.Classname
			spot.s.Origin[0] = self.s.Origin[0]
			spot.s.Origin[1] = self.s.Origin[1]
			spot.s.Origin[2] = self.s.Origin[2]
			spot.s.Angles[1] = self.s.Angles[1]
			spot.Targetname = ""

			return
		}
	}

	/* mine2 */
	//  if (Q_stricmp(level.mapname, "mine2") == 0) {
	// 	 if (Q_stricmp(self->targetname, "mine1") == 0) {
	// 		 spot->classname = self->classname;
	// 		 spot->s.origin[0] = self->s.origin[0];
	// 		 spot->s.origin[1] = self->s.origin[1];
	// 		 spot->s.origin[2] = self->s.origin[2];
	// 		 spot->s.angles[1] = self->s.angles[1];
	// 		 spot->targetname = NULL;

	// 		 return;
	// 	 }
	//  }

	//  /* mine3 */
	//  if (Q_stricmp(level.mapname, "mine3") == 0) {
	// 	 if (Q_stricmp(self->targetname, "mine2a") == 0) {
	// 		 spot->classname = self->classname;
	// 		 spot->s.origin[0] = self->s.origin[0];
	// 		 spot->s.origin[1] = self->s.origin[1];
	// 		 spot->s.origin[2] = self->s.origin[2];
	// 		 spot->s.angles[1] = self->s.angles[1];
	// 		 spot->targetname = NULL;

	// 		 return;
	// 	 }
	//  }

	//  /* mine4 */
	//  if (Q_stricmp(level.mapname, "mine4") == 0) {
	// 	 if (Q_stricmp(self->targetname, "mine3") == 0) {
	// 		 spot->classname = self->classname;
	// 		 spot->s.origin[0] = self->s.origin[0];
	// 		 spot->s.origin[1] = self->s.origin[1];
	// 		 spot->s.origin[2] = self->s.origin[2];
	// 		 spot->s.angles[1] = self->s.angles[1];
	// 		 spot->targetname = NULL;

	// 		 return;
	// 	 }
	//  }

	//   /* power2 */
	//  if (Q_stricmp(level.mapname, "power2") == 0) {
	// 	 if (Q_stricmp(self->targetname, "power1") == 0) {
	// 		 spot->classname = self->classname;
	// 		 spot->s.origin[0] = self->s.origin[0];
	// 		 spot->s.origin[1] = self->s.origin[1];
	// 		 spot->s.origin[2] = self->s.origin[2];
	// 		 spot->s.angles[1] = self->s.angles[1];
	// 		 spot->targetname = NULL;

	// 		 return;
	// 	 }
	//  }

	//  /* waste1 */
	//  if (Q_stricmp(level.mapname, "waste1") == 0) {
	// 	 if (Q_stricmp(self->targetname, "power2") == 0) {
	// 		 spot->classname = self->classname;
	// 		 spot->s.origin[0] = self->s.origin[0];
	// 		 spot->s.origin[1] = self->s.origin[1];
	// 		 spot->s.origin[2] = self->s.origin[2];
	// 		 spot->s.angles[1] = self->s.angles[1];
	// 		 spot->targetname = NULL;

	// 		 return;
	// 	 }
	//  }

	//  /* waste2 */
	//  if (Q_stricmp(level.mapname, "waste2") == 0) {
	// 	 if (Q_stricmp(self->targetname, "waste1") == 0) {
	// 		 spot->classname = self->classname;
	// 		 spot->s.origin[0] = self->s.origin[0];
	// 		 spot->s.origin[1] = self->s.origin[1];
	// 		 spot->s.origin[2] = self->s.origin[2];
	// 		 spot->s.angles[1] = self->s.angles[1];
	// 		 spot->targetname = NULL;

	// 		 return;
	// 	 }
	//  }

	//  /* waste3 */
	//  if (Q_stricmp(level.mapname, "waste3") == 0) {
	// 	 if (Q_stricmp(self->targetname, "waste2") == 0) {
	// 		 spot->classname = self->classname;
	// 		 spot->s.origin[0] = self->s.origin[0];
	// 		 spot->s.origin[1] = self->s.origin[1];
	// 		 spot->s.origin[2] = self->s.origin[2];
	// 		 spot->s.angles[1] = self->s.angles[1];
	// 		 spot->targetname = NULL;

	// 		 return;
	// 	 }
	//  }

	//  /* city3 */
	//  if (Q_stricmp(level.mapname, "city2") == 0) {
	// 	 if (Q_stricmp(self->targetname, "city2NL") == 0) {
	// 		 spot->classname = self->classname;
	// 		 spot->s.origin[0] = self->s.origin[0];
	// 		 spot->s.origin[1] = self->s.origin[1];
	// 		 spot->s.origin[2] = self->s.origin[2];
	// 		 spot->s.angles[1] = self->s.angles[1];
	// 		 spot->targetname = NULL;

	// 		 return;
	// 	 }
	//  }
}

/*
 * QUAKED info_player_start (1 0 0) (-16 -16 -24) (16 16 32)
 * The normal starting point for a level.
 */
func spInfoPlayerStart(self *edict_t, G *qGame) error {
	if self == nil {
		return nil
	}

	/* Call function to hack unnamed spawn points */
	self.think = spCreateUnnamedSpawn
	self.nextthink = G.level.time + FRAMETIME

	if !G.coop.Bool() {
		return nil
	}

	if G.level.mapname == "security" {
		/* invoke one of our gross, ugly, disgusting hacks */
		// 	self->think = SP_CreateCoopSpots;
		// 	self->nextthink = level.time + FRAMETIME;
	}
	return nil
}

/*
 * QUAKED info_player_deathmatch (1 0 1) (-16 -16 -24) (16 16 32)
 * potential spawning position for deathmatch games
 */
func spInfoPlayerDeathmatch(self *edict_t, G *qGame) error {
	if self == nil || G == nil {
		return nil
	}

	if !G.deathmatch.Bool() {
		G.gFreeEdict(self)
		return nil
	}

	return spMiscTeleporterDest(self, G)
}

func (G *qGame) initClientResp(client *gclient_t) {
	if client == nil {
		return
	}

	client.resp = client_respawn_t{}
	client.resp.enterframe = G.level.framenum
	client.resp.coop_respawn = client.pers
}

func (G *qGame) fetchClientEntData(ent *edict_t) {
	if ent == nil {
		return
	}

	ent.Health = ent.client.pers.health
	ent.max_health = ent.client.pers.max_health
	ent.flags |= ent.client.pers.savedFlags

	if G.coop.Bool() {
		ent.client.resp.score = ent.client.pers.score
	}
}

/*
 * called when a client has finished connecting, and is ready
 * to be placed into the game.  This will happen every level load.
 */
func (G *qGame) ClientBegin(sent shared.Edict_s) error {
	//  int i;

	ent := sent.(*edict_t)
	if ent == nil {
		return nil
	}

	ent.client = &G.game.clients[ent.index-1]

	//  if (deathmatch->value) {
	// 	 ClientBeginDeathmatch(ent);
	// 	 return;
	//  }

	/* if there is already a body waiting for us (a loadgame),
	just take it, otherwise spawn one from scratch */
	if ent.inuse == true {
		/* the client has cleared the client side viewangles upon
		connecting to the server, which is different than the
		state when the game is saved, so we need to compensate
		with deltaangles */
		//  for i := 0; i < 3; i++ {
		// 	 ent->client->ps.pmove.delta_angles[i] = ANGLE2SHORT(
		// 			 ent->client->ps.viewangles[i]);
		//  }
	} else {
		/* a spawn point will completely reinitialize the entity
		except for the persistant data that was initialized at
		ClientConnect() time */
		G_InitEdict(ent, ent.index)
		ent.Classname = "player"
		G.initClientResp(ent.client)
		if err := G.putClientInServer(ent); err != nil {
			return err
		}
	}

	//  if (level.intermissiontime) {
	// 	 MoveClientToIntermission(ent);
	//  } else {
	// 	 /* send effect if in a multiplayer game */
	// 	 if (game.maxclients > 1) {
	// 		 gi.WriteByte(svc_muzzleflash);
	// 		 gi.WriteShort(ent - g_edicts);
	// 		 gi.WriteByte(MZ_LOGIN);
	// 		 gi.multicast(ent->s.origin, MULTICAST_PVS);

	// 		 gi.bprintf(PRINT_HIGH, "%s entered the game\n",
	// 				 ent->client->pers.netname);
	// 	 }
	//  }

	/* make sure all view stuff is valid */
	G.clientEndServerFrame(ent)
	return nil
}

/*
 * Called whenever the player updates a userinfo variable.
 * The game can override any of the settings in place
 * (forcing skins or names, etc) before copying it off.
 */
func (G *qGame) clientUserinfoChanged(ent *edict_t, userinfo string) {

	if ent == nil {
		return
	}

	/* check for malformed or illegal info strings */
	// if !Info_Validate(userinfo) {
	// 	strcpy(userinfo, "\\name\\badinfo\\skin\\male/grunt")
	// }

	/* set name */
	s := shared.Info_ValueForKey(userinfo, "name")
	ent.client.pers.netname = s

	/* set spectator */
	//  s = shared.Info_ValueForKey(userinfo, "spectator");

	/* spectators are only supported in deathmatch */
	//  if (deathmatch.value && *s && strcmp(s, "0")) {
	// 	 ent->client->pers.spectator = true;
	//  } else {
	ent.client.pers.spectator = false
	//  }

	/* set skin */
	s = shared.Info_ValueForKey(userinfo, "skin")

	playernum := ent.index - 1

	/* combine name and skin into a configstring */
	G.gi.Configstring(shared.CS_PLAYERSKINS+playernum,
		fmt.Sprintf("%s\\%s", ent.client.pers.netname, s))

	/* fov */
	if G.deathmatch.Bool() && (G.dmflags.Int()&shared.DF_FIXED_FOV) != 0 {
		ent.client.ps.Fov = 90
	} else {
		fov, _ := strconv.ParseInt(shared.Info_ValueForKey(userinfo, "fov"), 10, 32)

		ent.client.ps.Fov = float32(fov)
		if ent.client.ps.Fov < 1 {
			ent.client.ps.Fov = 90
		} else if ent.client.ps.Fov > 160 {
			ent.client.ps.Fov = 160
		}
	}

	/* handedness */
	s = shared.Info_ValueForKey(userinfo, "hand")

	if len(s) > 0 {
		h, _ := strconv.ParseInt(s, 10, 32)
		ent.client.pers.hand = int(h)
	}

	/* save off the userinfo in case we want to check something later */
	ent.client.pers.userinfo = userinfo
}

/*
 * Called when a player begins connecting to the server.
 * The game can refuse entrance to a client by returning false.
 * If the client is allowed, the connection process will continue
 * and eventually get to ClientBegin(). Changing levels will NOT
 * cause this to be called again, but loadgames will.
 */
func (G *qGame) ClientConnect(sent shared.Edict_s, userinfo string) bool {

	ent := sent.(*edict_t)
	if ent == nil {
		return false
	}

	/* check to see if they are on the banned IP list */
	// value := shared.Info_ValueForKey(userinfo, "ip")

	//  if (SV_FilterPacket(value)) {
	// 	 Info_SetValueForKey(userinfo, "rejmsg", "Banned.");
	// 	 return false;
	//  }

	//  /* check for a spectator */
	//  value = Info_ValueForKey(userinfo, "spectator");

	//  if (deathmatch->value && *value && strcmp(value, "0"))
	//  {
	// 	 int i, numspec;

	// 	 if (*spectator_password->string &&
	// 		 strcmp(spectator_password->string, "none") &&
	// 		 strcmp(spectator_password->string, value))
	// 	 {
	// 		 Info_SetValueForKey(userinfo, "rejmsg",
	// 				 "Spectator password required or incorrect.");
	// 		 return false;
	// 	 }

	// 	 /* count spectators */
	// 	 for (i = numspec = 0; i < maxclients->value; i++)
	// 	 {
	// 		 if (g_edicts[i + 1].inuse && g_edicts[i + 1].client->pers.spectator)
	// 		 {
	// 			 numspec++;
	// 		 }
	// 	 }

	// 	 if (numspec >= maxspectators->value)
	// 	 {
	// 		 Info_SetValueForKey(userinfo, "rejmsg",
	// 				 "Server spectator limit is full.");
	// 		 return false;
	// 	 }
	//  }
	//  else
	//  {
	// 	 /* check for a password */
	// 	 value = Info_ValueForKey(userinfo, "password");

	// 	 if (*password->string && strcmp(password->string, "none") &&
	// 		 strcmp(password->string, value))
	// 	 {
	// 		 Info_SetValueForKey(userinfo, "rejmsg",
	// 				 "Password required or incorrect.");
	// 		 return false;
	// 	 }
	//  }

	/* they can connect */
	ent.client = &G.game.clients[ent.index-1]

	/* if there is already a body waiting for us (a loadgame),
	just take it, otherwise spawn one from scratch */
	if ent.inuse == false {
		/* clear the respawning variables */
		G.initClientResp(ent.client)

		if !G.game.autosaved || ent.client.pers.weapon == nil {
			G.initClientPersistant(ent.client)
		}
	}

	G.clientUserinfoChanged(ent, userinfo)

	if G.game.maxclients > 1 {
		G.gi.Dprintf("%s connected\n", ent.client.pers.netname)
	}

	ent.svflags = 0 /* make sure we start with known default */
	ent.client.pers.connected = true
	return true
}

/* ============================================================== */

// edict_t *pm_passent;
func pmPointcontents(point []float32, a interface{}) int {
	G := a.(*qGame)
	return G.gi.Pointcontents(point)
}

/*
 * pmove doesn't need to know
 * about passent and contentmask
 */
func PM_trace(start, mins, maxs, end []float32, a interface{}) shared.Trace_t {
	G := a.(*qGame)
	if G.pm_passent.Health > 0 {
		return G.gi.Trace(start, mins, maxs, end, G.pm_passent, shared.MASK_PLAYERSOLID)
	} else {
		return G.gi.Trace(start, mins, maxs, end, G.pm_passent, shared.MASK_DEADSOLID)
	}
}

/*
 * This will be called once for each client frame, which will
 * usually be a couple times for each server frame.
 */
func (G *qGame) ClientThink(sent shared.Edict_s, ucmd *shared.Usercmd_t) {
	//  gclient_t *client;
	//  edict_t *other;
	//  int i, j;
	//  pmove_t pm;

	if sent == nil || ucmd == nil {
		return
	}

	ent := sent.(*edict_t)

	G.level.current_entity = ent
	client := ent.client

	if G.level.intermissiontime != 0 {
		client.ps.Pmove.Pm_type = shared.PM_FREEZE

		// 	 /* can exit intermission after five seconds */
		// 	 if ((level.time > level.intermissiontime + 5.0) &&
		// 		 (ucmd->buttons & BUTTON_ANY)) {
		// 		 level.exitintermission = true;
		// 	 }

		return
	}

	G.pm_passent = ent

	if ent.client.chase_target != nil {
		client.resp.cmd_angles[0] = shared.SHORT2ANGLE(int(ucmd.Angles[0]))
		client.resp.cmd_angles[1] = shared.SHORT2ANGLE(int(ucmd.Angles[1]))
		client.resp.cmd_angles[2] = shared.SHORT2ANGLE(int(ucmd.Angles[2]))
	} else {
		/* set up for pmove */
		pm := shared.Pmove_t{}

		if ent.movetype == MOVETYPE_NOCLIP {
			client.ps.Pmove.Pm_type = shared.PM_SPECTATOR
		} else if ent.s.Modelindex != 255 {
			client.ps.Pmove.Pm_type = shared.PM_GIB
		} else if ent.deadflag != 0 {
			client.ps.Pmove.Pm_type = shared.PM_DEAD
		} else {
			client.ps.Pmove.Pm_type = shared.PM_NORMAL
		}

		client.ps.Pmove.Gravity = int16(G.sv_gravity.Int())
		pm.S = client.ps.Pmove

		for i := 0; i < 3; i++ {
			pm.S.Origin[i] = int16(ent.s.Origin[i] * 8)
			/* save to an int first, in case the short overflows
			 * so we get defined behavior (at least with -fwrapv) */
			tmpVel := int16(ent.velocity[i] * 8)
			pm.S.Velocity[i] = tmpVel
		}

		if !client.old_pmove.Equals(pm.S) {
			pm.Snapinitial = true
		}

		pm.Cmd.Copy(*ucmd)

		pm.TraceArg = G
		pm.Trace = PM_trace /* adds default parms */
		pm.PCArg = G
		pm.Pointcontents = pmPointcontents

		/* perform a pmove */
		G.gi.Pmove(&pm)

		/* save results of pmove */
		client.ps.Pmove.Copy(pm.S)
		client.old_pmove.Copy(pm.S)

		for i := 0; i < 3; i++ {
			ent.s.Origin[i] = float32(pm.S.Origin[i]) * 0.125
			ent.velocity[i] = float32(pm.S.Velocity[i]) * 0.125
		}

		copy(ent.mins[:], pm.Mins[:])
		copy(ent.maxs[:], pm.Maxs[:])

		client.resp.cmd_angles[0] = shared.SHORT2ANGLE(int(ucmd.Angles[0]))
		client.resp.cmd_angles[1] = shared.SHORT2ANGLE(int(ucmd.Angles[1]))
		client.resp.cmd_angles[2] = shared.SHORT2ANGLE(int(ucmd.Angles[2]))

		// 	 if (ent->groundentity && !pm.groundentity && (pm.cmd.upmove >= 10) &&
		// 		 (pm.waterlevel == 0)) {
		// 		 gi.sound(ent, CHAN_VOICE, gi.soundindex(
		// 						 "*jump1.wav"), 1, ATTN_NORM, 0);
		// 		 PlayerNoise(ent, ent->s.origin, PNOISE_SELF);
		// 	 }

		ent.viewheight = int(pm.Viewheight)
		ent.waterlevel = pm.Waterlevel
		ent.watertype = pm.Watertype
		if pm.Groundentity == nil {
			ent.groundentity = nil
		} else {
			ent.groundentity = pm.Groundentity.(*edict_t)
			ent.groundentity_linkcount = pm.Groundentity.(*edict_t).linkcount
		}

		if ent.deadflag != 0 {
			client.ps.Viewangles[shared.ROLL] = 40
			client.ps.Viewangles[shared.PITCH] = -15
			client.ps.Viewangles[shared.YAW] = client.killer_yaw
		} else {
			copy(client.v_angle[:], pm.Viewangles[:])
			copy(client.ps.Viewangles[:], pm.Viewangles[:])
		}

		G.gi.Linkentity(ent)

		if ent.movetype != MOVETYPE_NOCLIP {
			G.gTouchTriggers(ent)
		}

		/* touch other objects */
		for i := 0; i < pm.Numtouch; i++ {
			other, ok := pm.Touchents[i].(*edict_t)
			if !ok {
				continue
			}

			var found = false
			for j := 0; j < i; j++ {
				if pm.Touchents[j] == other {
					found = true
					break
				}
			}

			if found {
				continue /* duplicated */
			}

			if other.touch == nil {
				continue
			}

			other.touch(other, ent, nil, nil, G)
		}
	}

	client.oldbuttons = client.buttons
	client.buttons = int(ucmd.Buttons)
	//  client.latched_buttons |= client.buttons & ~client.oldbuttons;

	/* save light level the player is standing
	on for monster sighting AI */
	//  ent->light_level = ucmd->lightlevel;

	/* fire weapon from final position if needed */
	//  if (client->latched_buttons & BUTTON_ATTACK) != 0 {
	// 	 if (client->resp.spectator) {
	// 		 client->latched_buttons = 0;

	// 		 if (client->chase_target) {
	// 			 client->chase_target = NULL;
	// 			 client->ps.pmove.pm_flags &= ~PMF_NO_PREDICTION;
	// 		 } else {
	// 			 GetChaseTarget(ent);
	// 		 }
	// 	 } else if (!client->weapon_thunk) {
	// 		 client->weapon_thunk = true;
	// 		 Think_Weapon(ent);
	// 	 }
	//  }

	//  if (client->resp.spectator) {
	// 	 if (ucmd->upmove >= 10) {
	// 		 if (!(client->ps.pmove.pm_flags & PMF_JUMP_HELD)) {
	// 			 client->ps.pmove.pm_flags |= PMF_JUMP_HELD;

	// 			 if (client->chase_target) {
	// 				 ChaseNext(ent);
	// 			 } else {
	// 				 GetChaseTarget(ent);
	// 			 }
	// 		 }
	// 	 } else {
	// 		 client->ps.pmove.pm_flags &= ~PMF_JUMP_HELD;
	// 	 }
	//  }

	/* update chase cam if being followed */
	//  for i := 1; i <= maxclients->value; i++ {
	// 	 other = g_edicts + i;

	// 	 if (other->inuse && (other->client->chase_target == ent)) {
	// 		 UpdateChaseCam(other);
	// 	 }
	//  }
}

/*
 * This will be called once for each server
 * frame, before running any other entities
 * in the world.
 */
func (G *qGame) clientBeginServerFrame(ent *edict_t) {

	if ent == nil {
		return
	}

	if G.level.intermissiontime != 0 {
		return
	}

	client := ent.client

	//  if (deathmatch->value &&
	// 	 (client->pers.spectator != client->resp.spectator) &&
	// 	 ((level.time - client->respawn_time) >= 5))
	//  {
	// 	 spectator_respawn(ent);
	// 	 return;
	//  }

	/* run weapon animations if it hasn't been done by a ucmd_t */
	if !client.weapon_thunk && !client.resp.spectator {
		G.thinkWeapon(ent)
	} else {
		client.weapon_thunk = false
	}

	if ent.deadflag != 0 {
		// 	 /* wait for any button just going down */
		// 	 if (level.time > client->respawn_time)
		// 	 {
		// 		 /* in deathmatch, only wait for attack button */
		// 		 if (deathmatch->value)
		// 		 {
		// 			 buttonMask = BUTTON_ATTACK;
		// 		 }
		// 		 else
		// 		 {
		// 			 buttonMask = -1;
		// 		 }

		// 		 if ((client->latched_buttons & buttonMask) ||
		// 			 (deathmatch->value && ((int)dmflags->value & DF_FORCE_RESPAWN)))
		// 		 {
		// 			 respawn(ent);
		// 			 client->latched_buttons = 0;
		// 		 }
		// 	 }

		return
	}

	//  /* add player trail so monsters can follow */
	//  if (!deathmatch->value) {
	// 	 if (!visible(ent, PlayerTrail_LastSpot())) {
	// 		 PlayerTrail_Add(ent->s.old_origin);
	// 	 }
	//  }

	client.latched_buttons = 0
}
