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
 * Monster utility functions.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

func (G *qGame) mCheckGround(ent *edict_t) {
	// vec3_t point;
	// trace_t trace;

	if ent == nil {
		return
	}

	if (ent.flags & (FL_SWIM | FL_FLY)) != 0 {
		return
	}

	if ent.velocity[2] > 100 {
		ent.groundentity = nil
		return
	}

	/* if the hull point one-quarter unit down
	   is solid the entity is on ground */
	point := []float32{ent.s.Origin[0], ent.s.Origin[1], ent.s.Origin[2] - 0.25}

	trace := G.gi.Trace(ent.s.Origin[:], ent.mins[:], ent.maxs[:], point, ent, shared.MASK_MONSTERSOLID)

	/* check steepness */
	if (trace.Plane.Normal[2] < 0.7) && !trace.Startsolid {
		ent.groundentity = nil
		return
	}

	if !trace.Startsolid && !trace.Allsolid {
		copy(ent.s.Origin[:], trace.Endpos[:])
		ent.groundentity = trace.Ent.(*edict_t)
		ent.groundentity_linkcount = trace.Ent.(*edict_t).linkcount
		ent.velocity[2] = 0
	}
}

func (G *qGame) mCatagorizePosition(ent *edict_t) {
	// vec3_t point;
	// int cont;

	if ent == nil {
		return
	}

	/* get waterlevel */
	point := []float32{(ent.absmax[0] + ent.absmin[0]) / 2, (ent.absmax[1] + ent.absmin[1]) / 2, ent.absmin[2] + 2}
	cont := G.gi.Pointcontents(point)

	if (cont & shared.MASK_WATER) == 0 {
		ent.waterlevel = 0
		ent.watertype = 0
		return
	}

	ent.watertype = cont
	ent.waterlevel = 1
	point[2] += 26
	cont = G.gi.Pointcontents(point)

	if (cont & shared.MASK_WATER) == 0 {
		return
	}

	ent.waterlevel = 2
	point[2] += 22
	cont = G.gi.Pointcontents(point)

	if (cont & shared.MASK_WATER) != 0 {
		ent.waterlevel = 3
	}
}

func (G *qGame) mDroptofloor(ent *edict_t) {

	if ent == nil {
		return
	}

	ent.s.Origin[2] += 1
	end := make([]float32, 3)
	copy(end, ent.s.Origin[:])
	end[2] -= 256

	trace := G.gi.Trace(ent.s.Origin[:], ent.mins[:], ent.maxs[:], end, ent, shared.MASK_MONSTERSOLID)

	if (trace.Fraction == 1) || trace.Allsolid {
		return
	}

	copy(ent.s.Origin[:], trace.Endpos[:])

	G.gi.Linkentity(ent)
	G.mCheckGround(ent)
	G.mCatagorizePosition(ent)
}

func (G *qGame) mMoveFrame(self *edict_t) {
	// mmove_t *move;
	// int index;

	if self == nil {
		return
	}

	move := self.monsterinfo.currentmove
	self.nextthink = G.level.time + FRAMETIME

	if (self.monsterinfo.nextframe != 0) &&
		(self.monsterinfo.nextframe >= move.firstframe) &&
		(self.monsterinfo.nextframe <= move.lastframe) {
		if self.s.Frame != self.monsterinfo.nextframe {
			self.s.Frame = self.monsterinfo.nextframe
			self.monsterinfo.aiflags &^= AI_HOLD_FRAME
		}

		self.monsterinfo.nextframe = 0
	} else {
		/* prevent nextframe from leaking into a future move */
		self.monsterinfo.nextframe = 0

		if self.s.Frame == move.lastframe {
			if move.endfunc != nil {
				move.endfunc(self, G)

				/* regrab move, endfunc is very likely to change it */
				move = self.monsterinfo.currentmove

				/* check for death */
				if (self.svflags & shared.SVF_DEADMONSTER) != 0 {
					return
				}
			}
		}

		if (self.s.Frame < move.firstframe) ||
			(self.s.Frame > move.lastframe) {
			self.monsterinfo.aiflags &^= AI_HOLD_FRAME
			self.s.Frame = move.firstframe
		} else {
			if (self.monsterinfo.aiflags & AI_HOLD_FRAME) == 0 {
				self.s.Frame++

				if self.s.Frame > move.lastframe {
					self.s.Frame = move.firstframe
				}
			}
		}
	}

	index := self.s.Frame - move.firstframe

	if move.frame[index].aifunc != nil {
		if (self.monsterinfo.aiflags & AI_HOLD_FRAME) == 0 {
			move.frame[index].aifunc(self,
				move.frame[index].dist*self.monsterinfo.scale, G)
		} else {
			move.frame[index].aifunc(self, 0, G)
		}
	}

	if move.frame[index].thinkfunc != nil {
		move.frame[index].thinkfunc(self, G)
	}
}

func monster_think(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	G.mMoveFrame(self)

	if self.linkcount != self.monsterinfo.linkcount {
		self.monsterinfo.linkcount = self.linkcount
		G.mCheckGround(self)
	}

	G.mCatagorizePosition(self)
	// M_WorldEffects(self);
	// M_SetEffects(self);
}

func (G *qGame) monster_triggered_start(self *edict_t) {
	if self == nil {
		return
	}

	self.solid = shared.SOLID_NOT
	self.movetype = MOVETYPE_NONE
	self.svflags |= shared.SVF_NOCLIENT
	self.nextthink = 0
	// self.use = monster_triggered_spawn_use
}

/*
 * When a monster dies, it fires all of its targets
 * with the current enemy as activator.
 */
func (G *qGame) monster_death_use(self *edict_t) {
	if self == nil {
		return
	}

	self.flags &^= (FL_FLY | FL_SWIM)
	self.monsterinfo.aiflags &= AI_GOOD_GUY

	if self.item != nil {
		// Drop_Item(self, self.item)
		self.item = nil
	}

	if len(self.Deathtarget) > 0 {
		self.Target = self.Deathtarget
	}

	if len(self.Target) == 0 {
		return
	}

	G.gUseTargets(self, self.enemy)
}

/* ================================================================== */

func (G *qGame) monster_start(self *edict_t) bool {
	if self == nil {
		return false
	}

	if G.deathmatch.Bool() {
		G.gFreeEdict(self)
		return false
	}

	if (self.Spawnflags&4) != 0 && (self.monsterinfo.aiflags&AI_GOOD_GUY) == 0 {
		self.Spawnflags &^= 4
		self.Spawnflags |= 1
	}

	if (self.Spawnflags&2) != 0 && len(self.Targetname) == 0 {
		// 	if (g_fix_triggered->value) {
		// 		self->spawnflags &= ~2;
		// 	}

		G.gi.Dprintf("triggered %s at %s has no targetname\n", self.Classname, vtos(self.s.Origin[:]))
	}

	if (self.monsterinfo.aiflags & AI_GOOD_GUY) == 0 {
		G.level.total_monsters++
	}

	self.nextthink = G.level.time + FRAMETIME
	self.svflags |= shared.SVF_MONSTER
	self.s.Renderfx |= shared.RF_FRAMELERP
	self.takedamage = DAMAGE_AIM
	// self.air_finished = level.time + 12
	// self.use = monster_use

	if self.max_health == 0 {
		self.max_health = self.Health
	}

	self.clipmask = shared.MASK_MONSTERSOLID

	self.s.Skinnum = 0
	self.deadflag = DEAD_NO
	self.svflags &^= shared.SVF_DEADMONSTER

	if self.monsterinfo.checkattack == nil {
		self.monsterinfo.checkattack = mCheckAttack
	}

	copy(self.s.Old_origin[:], self.s.Origin[:])

	// if (st.item) {
	// 	self->item = FindItemByClassname(st.item);

	// 	if (!self->item) {
	// 		gi.dprintf("%s at %s has bad item: %s\n", self->classname,
	// 				vtos(self->s.origin), st.item);
	// 	}
	// }

	/* randomize what frame they start on */
	// if (self.monsterinfo.currentmove)
	// {
	// 	self->s.frame = self->monsterinfo.currentmove->firstframe +
	// 		(randk() % (self->monsterinfo.currentmove->lastframe -
	// 				   self->monsterinfo.currentmove->firstframe + 1));
	// }

	return true
}

func (G *qGame) monster_start_go(self *edict_t) {
	// vec3_t v;

	if self == nil {
		return
	}

	if self.Health <= 0 {
		return
	}

	/* check for target to combat_point and change to combattarget */
	if len(self.Target) > 0 {

		var target *edict_t = nil
		notcombat := false
		fixup := false

		for {
			target = G.gFind(target, "Targetname", self.Target)
			if target == nil {
				break
			}
			if target.Classname == "point_combat" {
				self.Combattarget = self.Target
				fixup = true
			} else {
				notcombat = true
			}
		}

		if notcombat {
			G.gi.Dprintf("%s at %s has target with mixed types\n",
				self.Classname, vtos(self.s.Origin[:]))
		}

		if fixup {
			self.Target = ""
		}
	}

	/* validate combattarget */
	if len(self.Combattarget) > 0 {

		var target *edict_t = nil

		for {
			target = G.gFind(target, "Targetname", self.Combattarget)
			if target == nil {
				break
			}
			if target.Classname != "point_combat" {
				G.gi.Dprintf("%s at (%v %v %v) has a bad combattarget %s : %s at (%v %v %v)\n",
					self.Classname, int(self.s.Origin[0]), int(self.s.Origin[1]),
					int(self.s.Origin[2]), self.Combattarget, target.Classname,
					int(target.s.Origin[0]), int(target.s.Origin[1]),
					int(target.s.Origin[2]))
			}
		}
	}

	if len(self.Target) > 0 {
		self.movetarget = G.gPickTarget(self.Target)
		self.goalentity = self.movetarget

		if self.movetarget == nil {
			G.gi.Dprintf("%s can't find target %s at %s\n", self.Classname,
				self.Target, vtos(self.s.Origin[:]))
			self.Target = ""
			self.monsterinfo.pausetime = 100000000
			self.monsterinfo.stand(self, G)
		} else if self.movetarget.Classname == "path_corner" {
			v := make([]float32, 3)
			shared.VectorSubtract(self.goalentity.s.Origin[:], self.s.Origin[:], v)
			self.s.Angles[shared.YAW] = vectoyaw(v)
			self.ideal_yaw = self.s.Angles[shared.YAW]
			self.monsterinfo.walk(self, G)
			self.Target = ""
		} else {
			self.goalentity = nil
			self.movetarget = nil
			self.monsterinfo.pausetime = 100000000
			self.monsterinfo.stand(self, G)
		}
	} else {
		self.monsterinfo.pausetime = 100000000
		self.monsterinfo.stand(self, G)
	}

	self.think = monster_think
	self.nextthink = G.level.time + FRAMETIME
}

func walkmonster_start_go(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if (self.Spawnflags&2) == 0 && (G.level.time < 1) {
		G.mDroptofloor(self)

		if self.groundentity != nil {
			if !G.mWalkmove(self, 0, 0) {
				G.gi.Dprintf("%s in solid at %s\n", self.Classname,
					vtos(self.s.Origin[:]))
			}
		}
	}

	if self.yaw_speed == 0 {
		self.yaw_speed = 20
	}

	if self.viewheight == 0 {
		self.viewheight = 25
	}

	if (self.Spawnflags & 2) != 0 {
		G.monster_triggered_start(self)
	} else {
		G.monster_start_go(self)
	}
}

func (G *qGame) walkmonster_start(self *edict_t) {
	if self == nil {
		return
	}

	self.think = walkmonster_start_go
	G.monster_start(self)
}
