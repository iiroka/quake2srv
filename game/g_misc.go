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
 * Miscellaneos entities, functs and functions.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

/* ===================================================== */

/*
 * QUAKED path_corner (.5 .3 0) (-8 -8 -8) (8 8 8) TELEPORT
 * Target: next path corner
 * Pathtarget: gets used when an entity that has
 *             this path_corner targeted touches it
 */
func path_corner_touch(self, other *edict_t, plane *shared.Cplane_t,
	surf *shared.Csurface_t, G *qGame) {

	if self == nil || other == nil || G == nil {
		return
	}

	if other.movetarget != self {
		return
	}

	if other.enemy != nil {
		return
	}

	if len(self.Pathtarget) > 0 {
		savetarget := self.Target
		self.Target = self.Pathtarget
		G.gUseTargets(self, other)
		self.Target = savetarget
	}

	var next *edict_t
	if len(self.Target) > 0 {
		next = G.gPickTarget(self.Target)
	} else {
		next = nil
	}

	if (next != nil) && (next.Spawnflags&1) != 0 {
		v := make([]float32, 3)
		copy(v, next.s.Origin[:])
		v[2] += next.mins[2]
		v[2] -= other.mins[2]
		copy(other.s.Origin[:], v)
		next = G.gPickTarget(next.Target)
		other.s.Event = shared.EV_OTHER_TELEPORT
	}

	other.goalentity = next
	other.movetarget = next

	if self.Wait != 0 {
		other.monsterinfo.pausetime = G.level.time + self.Wait
		other.monsterinfo.stand(other, G)
		return
	}

	if other.movetarget == nil {
		other.monsterinfo.pausetime = G.level.time + 100000000
		other.monsterinfo.stand(other, G)
	} else {
		v := make([]float32, 3)
		shared.VectorSubtract(other.goalentity.s.Origin[:], other.s.Origin[:], v)
		other.ideal_yaw = vectoyaw(v)
	}
}

func spPathCorner(self *edict_t, G *qGame) error {
	if self == nil || G == nil {
		return nil
	}

	if len(self.Targetname) == 0 {
		G.gi.Dprintf("path_corner with no targetname at %s\n",
			vtos(self.s.Origin[:]))
		G.gFreeEdict(self)
		return nil
	}

	self.solid = shared.SOLID_TRIGGER
	self.touch = path_corner_touch
	self.mins = [3]float32{-8, -8, -8}
	self.maxs = [3]float32{8, 8, 8}
	self.svflags |= shared.SVF_NOCLIENT
	G.gi.Linkentity(self)
	return nil
}

/* ===================================================== */

/*
 * QUAKED point_combat (0.5 0.3 0) (-8 -8 -8) (8 8 8) Hold
 *
 * Makes this the target of a monster and it will head here
 * when first activated before going after the activator.  If
 * hold is selected, it will stay here.
 */
func point_combat_touch(self, other *edict_t, plane *shared.Cplane_t,
	surf *shared.Csurface_t, G *qGame) {

	if self == nil || other == nil || G == nil {
		return
	}

	if other.movetarget != self {
		return
	}

	if len(self.Target) > 0 {
		other.Target = self.Target
		other.movetarget = G.gPickTarget(other.Target)
		other.goalentity = other.movetarget

		// 		 if (!other->goalentity)
		// 		 {
		// 			 gi.dprintf("%s at %s target %s does not exist\n",
		// 					 self->classname,
		// 					 vtos(self->s.origin),
		// 					 self->target);
		// 			 other->movetarget = self;
		// 		 }

		self.Target = ""
	} else if (self.Spawnflags&1) != 0 && (other.flags&(FL_SWIM|FL_FLY)) == 0 {
		other.monsterinfo.pausetime = G.level.time + 100000000
		other.monsterinfo.aiflags |= AI_STAND_GROUND
		other.monsterinfo.stand(other, G)
	}

	if other.movetarget == self {
		other.Target = ""
		other.movetarget = nil
		other.goalentity = other.enemy
		other.monsterinfo.aiflags &^= AI_COMBAT_POINT
	}

	if len(self.Pathtarget) > 0 {

		savetarget := self.Target
		self.Target = self.Pathtarget

		var activator *edict_t
		if other.enemy != nil && other.enemy.client != nil {
			activator = other.enemy
		} else if other.oldenemy != nil && other.oldenemy.client != nil {
			activator = other.oldenemy
		} else if other.activator != nil && other.activator.client != nil {
			activator = other.activator
		} else {
			activator = other
		}

		G.gUseTargets(self, activator)
		self.Target = savetarget
	}
}

func spPointCombat(self *edict_t, G *qGame) error {
	if self == nil || G == nil {
		return nil
	}

	if G.deathmatch.Bool() {
		G.gFreeEdict(self)
		return nil
	}

	self.solid = shared.SOLID_TRIGGER
	self.touch = point_combat_touch
	self.mins = [3]float32{-8, -8, -16}
	self.maxs = [3]float32{8, 8, 16}
	self.svflags = shared.SVF_NOCLIENT
	G.gi.Linkentity(self)
	return nil
}

const START_OFF = 1

func spLight(self *edict_t, G *qGame) error {
	if self == nil {
		return nil
	}

	/* no targeted lights in deathmatch, because they cause global messages */
	if len(self.Targetname) == 0 || G.deathmatch.Bool() {
		G.gFreeEdict(self)
		return nil
	}

	if self.Style >= 32 {
		// self.use = light_use;

		if (self.Spawnflags & START_OFF) != 0 {
			return G.gi.Configstring(shared.CS_LIGHTS+self.Style, "a")
		} else {
			return G.gi.Configstring(shared.CS_LIGHTS+self.Style, "m")
		}
	}
	return nil
}

/*
 * QUAKED misc_teleporter_dest (1 0 0) (-32 -32 -24) (32 32 -16)
 * Point teleporters at these.
 */
func spMiscTeleporterDest(ent *edict_t, G *qGame) error {
	if ent == nil || G == nil {
		return nil
	}

	G.gi.Setmodel(ent, "models/objects/dmspot/tris.md2")
	ent.s.Skinnum = 0
	ent.solid = shared.SOLID_BBOX
	copy(ent.mins[:], []float32{-32, -32, -24})
	copy(ent.maxs[:], []float32{32, 32, -16})
	G.gi.Linkentity(ent)
	return nil
}
