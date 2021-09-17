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
 * Misc. utility functions for the game logic.
 *
 * =======================================================================
 */
package game

import (
	"fmt"
	"math"
	"quake2srv/shared"
	"reflect"
)

const MAXCHOICES = 8

func gProjectSource(point, distance, forward, right, result []float32) {
	result[0] = point[0] + forward[0]*distance[0] + right[0]*distance[1]
	result[1] = point[1] + forward[1]*distance[0] + right[1]*distance[1]
	result[2] = point[2] + forward[2]*distance[0] + right[2]*distance[1] +
		distance[2]
}

/*
 * Searches all active entities for the next
 * one that holds the matching string at fieldofs
 * (use the FOFS() macro) in the structure.
 *
 * Searches beginning at the edict after from, or
 * the beginning. If NULL, NULL will be returned
 * if the end of the list is reached.
 */
func (G *qGame) gFind(from *edict_t, fname, match string) *edict_t {

	var index int = 0
	if from != nil {
		index = from.index + 1
	}

	if len(match) == 0 {
		return nil
	}

	for ; index < G.num_edicts; index++ {
		if !G.g_edicts[index].inuse {
			continue
		}

		b := reflect.ValueOf(&G.g_edicts[index]).Elem()
		f := b.FieldByName(fname)

		if !f.IsValid() || f.Kind() != reflect.String {
			continue
		}

		s := f.String()
		if s == match {
			return &G.g_edicts[index]
		}
	}

	return nil
}

/*
 * Searches all active entities for
 * the next one that holds the matching
 * string at fieldofs (use the FOFS() macro)
 * in the structure.
 *
 * Searches beginning at the edict after from,
 * or the beginning. If NULL, NULL will be
 * returned if the end of the list is reached.
 */
func (G *qGame) gPickTarget(targetname string) *edict_t {

	if len(targetname) == 0 {
		G.gi.Dprintf("G_PickTarget called with NULL targetname\n")
		return nil
	}

	var ent *edict_t = nil
	num_choices := 0
	var choice [MAXCHOICES]*edict_t
	for {
		ent := G.gFind(ent, "Targetname", targetname)
		if ent == nil {
			break
		}

		choice[num_choices] = ent
		num_choices++

		if num_choices == MAXCHOICES {
			break
		}
	}

	if num_choices == 0 {
		G.gi.Dprintf("G_PickTarget: target %s not found\n", targetname)
		return nil
	}

	return choice[shared.Randk()%num_choices]
}

func think_Delay(ent *edict_t, G *qGame) {
	if ent == nil || G == nil {
		return
	}

	G.gUseTargets(ent, ent.activator)
	G.gFreeEdict(ent)
}

/*
 * The global "activator" should be set to
 * the entity that initiated the firing.
 *
 * If self.delay is set, a DelayedUse entity
 * will be created that will actually do the
 * SUB_UseTargets after that many seconds have passed.
 *
 * Centerprints any self.message to the activator.
 *
 * Search for (string)targetname in all entities that
 * match (string)self.target and call their .use function
 */
func (G *qGame) gUseTargets(ent, activator *edict_t) {

	if ent == nil {
		return
	}

	/* check for a delay */
	if ent.Delay != 0 {
		/* create a temp object to fire at a later time */
		t, _ := G.gSpawn()
		t.Classname = "DelayedUse"
		t.nextthink = G.level.time + ent.Delay
		t.think = think_Delay
		t.activator = activator

		if activator == nil {
			G.gi.Dprintf("Think_Delay with no activator\n")
		}

		t.Message = ent.Message
		t.Target = ent.Target
		t.Killtarget = ent.Killtarget
		return
	}

	if activator == nil {
		return
	}

	/* print the message */
	if len(ent.Message) > 0 && (activator.svflags&shared.SVF_MONSTER) == 0 {
		//  gi.centerprintf(activator, "%s", ent->message);

		//  if (ent->noise_index) {
		// 	 gi.sound(activator, CHAN_AUTO, ent->noise_index, 1, ATTN_NORM, 0);
		//  } else {
		// 	 gi.sound(activator, CHAN_AUTO, gi.soundindex(
		// 					 "misc/talk1.wav"), 1, ATTN_NORM, 0);
		//  }
	}

	/* kill killtargets */
	if len(ent.Killtarget) > 0 {
		var t *edict_t

		for {
			t = G.gFind(t, "TargetName", ent.Killtarget)
			if t == nil {
				break
			}
			// 	 while ((t = G_Find(t, FOFS(targetname), ent->killtarget))) {
			/* decrement secret count if target_secret is removed */
			// 		 if (!Q_stricmp(t->classname,"target_secret")) {
			// 			 level.total_secrets--;
			/* same deal with target_goal, but also turn off CD music if applicable */
			// 		 } else if (!Q_stricmp(t->classname,"target_goal")) {
			// 			 level.total_goals--;

			// 			 if (level.found_goals >= level.total_goals) {
			// 				 gi.configstring (CS_CDTRACK, "0");
			// 			 }
			// 		 }

			G.gFreeEdict(t)

			if !ent.inuse {
				G.gi.Dprintf("entity was removed while using killtargets\n")
				return
			}
		}
	}

	/* fire targets */
	if len(ent.Target) > 0 {
		// 	 t = NULL;
		var t *edict_t

		for {
			t = G.gFind(t, "TargetName", ent.Target)
			if t == nil {
				break
			}

			// 	 while ((t = G_Find(t, FOFS(targetname), ent->target)))
			// 	 {
			// 		 /* doors fire area portals in a specific way */
			// 		 if (!Q_stricmp(t->classname, "func_areaportal") &&
			// 			 (!Q_stricmp(ent->classname, "func_door") ||
			// 			  !Q_stricmp(ent->classname, "func_door_rotating")))
			// 		 {
			// 			 continue;
			// 		 }

			if t == ent {
				G.gi.Dprintf("WARNING: Entity used itself.\n")
			} else {
				if t.use != nil {
					t.use(t, ent, activator, G)
				}
			}

			if !ent.inuse {
				G.gi.Dprintf("entity was removed while using targets\n")
				return
			}
		}
	}
}

/*
 * This is just a convenience function
 * for printing vectors
 */
func vtos(v []float32) string {
	return fmt.Sprintf("(%v %v %v)", int(v[0]), int(v[1]), int(v[2]))
}

var VEC_UP = []float32{0, -1, 0}
var MOVEDIR_UP = []float32{0, 0, 1}
var VEC_DOWN = []float32{0, -2, 0}
var MOVEDIR_DOWN = []float32{0, 0, -1}

func gSetMovedir(angles, movedir []float32) {
	if shared.VectorCompare(angles, VEC_UP) != 0 {
		copy(movedir, MOVEDIR_UP)
	} else if shared.VectorCompare(angles, VEC_DOWN) != 0 {
		copy(movedir, MOVEDIR_DOWN)
	} else {
		shared.AngleVectors(angles, movedir, nil, nil)
	}

	angles[0] = 0
	angles[1] = 0
	angles[2] = 0
}

func vectoyaw(vec []float32) float32 {

	var yaw float32 = 0
	if vec[shared.PITCH] == 0 {
		yaw = 0

		if vec[shared.YAW] > 0 {
			yaw = 90
		} else if vec[shared.YAW] < 0 {
			yaw = -90
		}
	} else {
		yaw = float32(int(math.Atan2(float64(vec[shared.YAW]), float64(vec[shared.PITCH])) * 180 / math.Pi))
		if yaw < 0 {
			yaw += 360
		}
	}

	return yaw
}

func vectoangles(value1, angles []float32) {

	var yaw float32 = 0
	var pitch float32 = 0
	if (value1[1] == 0) && (value1[0] == 0) {
		yaw = 0

		if value1[2] > 0 {
			pitch = 90
		} else {
			pitch = 270
		}
	} else {
		if value1[0] != 0 {
			yaw = float32(int(math.Atan2(float64(value1[1]), float64(value1[0])) * 180 / math.Pi))
		} else if value1[1] > 0 {
			yaw = 90
		} else {
			yaw = -90
		}

		if yaw < 0 {
			yaw += 360
		}

		forward := math.Sqrt(float64(value1[0]*value1[0]) + float64(value1[1]*value1[1]))
		pitch = float32(int(math.Atan2(float64(value1[2]), forward) * 180 / math.Pi))

		if pitch < 0 {
			pitch += 360
		}
	}

	angles[shared.PITCH] = -pitch
	angles[shared.YAW] = yaw
	angles[shared.ROLL] = 0
}

func G_InitEdict(e *edict_t, index int) {
	e.inuse = true
	e.Classname = "noclass"
	e.gravity = 1.0
	e.s.Number = index
	e.area.Self = e
}

/*
 * Either finds a free edict, or allocates a
 * new one.  Try to avoid reusing an entity
 * that was recently freed, because it can
 * cause the client to think the entity
 * morphed into something else instead of
 * being removed and recreated, which can
 * cause interpolated angles and bad trails.
 */
func (G *qGame) gSpawn() (*edict_t, error) {

	for i := G.maxclients.Int() + 1; i < G.num_edicts; i++ {
		e := &G.g_edicts[i]
		/* the first couple seconds of
		server time can involve a lot of
		freeing and allocating, so relax
		the replacement policy */
		if !e.inuse && ((e.freetime < 2) || (G.level.time-e.freetime > 0.5)) {
			G_InitEdict(e, i)
			return e, nil
		}
	}

	if G.num_edicts == G.game.maxentities {
		return nil, G.gi.Error("ED_Alloc: no free edicts")
	}

	e := &G.g_edicts[G.num_edicts]
	G_InitEdict(e, G.num_edicts)
	G.num_edicts++
	return e, nil
}

func gFreeEdictFunc(ed *edict_t, G *qGame) {
	G.gFreeEdict(ed)
}

/*
 * Marks the edict as free
 */
func (G *qGame) gFreeEdict(ed *edict_t) {
	G.gi.Unlinkentity(ed) /* unlink from world */

	if G.deathmatch.Bool() || G.coop.Bool() {
		if ed.index <= (G.maxclients.Int() + BODY_QUEUE_SIZE) {
			return
		}
	} else {
		if ed.index <= G.maxclients.Int() {
			return
		}
	}

	index := ed.index
	ed.copy(edict_t{})
	ed.index = index
	ed.Classname = "freed"
	ed.freetime = G.level.time
	ed.inuse = false
}

func (G *qGame) gTouchTriggers(ent *edict_t) {

	if ent == nil {
		return
	}

	/* dead things don't activate triggers! */
	if (ent.client != nil || (ent.svflags&shared.SVF_MONSTER) != 0) && (ent.Health <= 0) {
		return
	}

	touch := make([]shared.Edict_s, shared.MAX_EDICTS)
	num := G.gi.BoxEdicts(ent.absmin[:], ent.absmax[:], touch, shared.MAX_EDICTS, shared.AREA_TRIGGERS)

	/* be careful, it is possible to have an entity in this
	   list removed before we get to it (killtriggered) */
	for i := 0; i < num; i++ {
		hit := touch[i].(*edict_t)

		if !hit.inuse {
			continue
		}

		if hit.touch == nil {
			continue
		}

		hit.touch(hit, ent, nil, nil, G)
	}
}
