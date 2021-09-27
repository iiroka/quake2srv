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
 * The basic AI functions like enemy detection, attacking and so on.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

/*
 * Move the specified distance at current facing.
 */
func ai_move(self *edict_t, dist float32, G *qGame) {
	if self == nil || G == nil {
		return
	}

	G.mWalkmove(self, self.s.Angles[shared.YAW], dist)
}

/*
 *
 * Used for standing around and looking
 * for players Distance is for slight
 * position adjustments needed by the
 * animations
 */
func ai_stand(self *edict_t, dist float32, G *qGame) {

	if self == nil || G == nil {
		return
	}

	if dist != 0 {
		G.mWalkmove(self, self.s.Angles[shared.YAW], dist)
	}

	if (self.monsterinfo.aiflags & AI_STAND_GROUND) != 0 {
		if self.enemy != nil {
			v := make([]float32, 3)
			shared.VectorSubtract(self.enemy.s.Origin[:], self.s.Origin[:], v)
			self.ideal_yaw = vectoyaw(v)

			if (self.s.Angles[shared.YAW] != self.ideal_yaw) &&
				(self.monsterinfo.aiflags&AI_TEMP_STAND_GROUND) != 0 {
				self.monsterinfo.aiflags &^= (AI_STAND_GROUND | AI_TEMP_STAND_GROUND)
				self.monsterinfo.run(self, G)
			}

			M_ChangeYaw(self)
			G.ai_checkattack(self)
		} else {
			G.findTarget(self)
		}

		return
	}

	if G.findTarget(self) {
		return
	}

	if G.level.time > self.monsterinfo.pausetime {
		self.monsterinfo.walk(self, G)
		return
	}

	if (self.Spawnflags&1) == 0 && (self.monsterinfo.idle != nil) &&
		(G.level.time > self.monsterinfo.idle_time) {
		if self.monsterinfo.idle_time > 0 {
			self.monsterinfo.idle(self, G)
			self.monsterinfo.idle_time = G.level.time + 15 + shared.Frandk()*15
		} else {
			self.monsterinfo.idle_time = G.level.time + shared.Frandk()*15
		}
	}
}

/*
 * The monster is walking it's beat
 */
func ai_walk(self *edict_t, dist float32, G *qGame) {

	if self == nil || G == nil {
		return
	}

	G.mMoveToGoal(self, dist)

	/* check for noticing a player */
	if G.findTarget(self) {
		return
	}

	if (self.monsterinfo.search != nil) && (G.level.time > self.monsterinfo.idle_time) {
		if self.monsterinfo.idle_time != 0 {
			self.monsterinfo.search(self, G)
			self.monsterinfo.idle_time = G.level.time + 15 + shared.Frandk()*15
		} else {
			self.monsterinfo.idle_time = G.level.time + shared.Frandk()*15
		}
	}
}

/*
 * Turns towards target and advances
 * Use this call with a distance of 0
 * to replace ai_face
 */
func ai_charge(self *edict_t, dist float32, G *qGame) {

	if self == nil || G == nil {
		return
	}

	v := make([]float32, 3)
	if self.enemy != nil {
		shared.VectorSubtract(self.enemy.s.Origin[:], self.s.Origin[:], v)
	}

	self.ideal_yaw = vectoyaw(v)
	M_ChangeYaw(self)

	if dist != 0 {
		G.mWalkmove(self, self.s.Angles[shared.YAW], dist)
	}
}

/* ============================================================================ */

/*
 * .enemy
 * Will be world if not currently angry at anyone.
 *
 * .movetarget
 * The next path spot to walk toward.  If .enemy, ignore .movetarget.
 * When an enemy is killed, the monster will try to return to it's path.
 *
 * .hunt_time
 * Set to time + something when the player is in sight, but movement straight for
 * him is blocked.  This causes the monster to use wall following code for
 * movement direction instead of sighting on the player.
 *
 * .ideal_yaw
 * A yaw angle of the intended direction, which will be turned towards at up
 * to 45 deg / state.  If the enemy is in view and hunt_time is not active,
 * this will be the exact line towards the enemy.
 *
 * .pausetime
 * A monster will leave it's stand state and head towards it's .movetarget when
 * time > .pausetime.
 */

/* ============================================================================ */

/*
 * returns the range categorization of an entity relative to self
 * 0	melee range, will become hostile even if back is turned
 * 1	visibility and infront, or visibility and show hostile
 * 2	infront and show hostile
 * 3	only triggered by damage
 */
func range_(self, other *edict_t) int {

	if self == nil || other == nil {
		return 0
	}

	v := make([]float32, 3)
	shared.VectorSubtract(self.s.Origin[:], other.s.Origin[:], v)
	len := shared.VectorLength(v)

	if len < MELEE_DISTANCE {
		return RANGE_MELEE
	}

	if len < 500 {
		return RANGE_NEAR
	}

	if len < 1000 {
		return RANGE_MID
	}

	return RANGE_FAR
}

/*
 * returns 1 if the entity is visible
 * to self, even if not infront
 */
func (G *qGame) visible(self, other *edict_t) bool {

	if self == nil || other == nil {
		return false
	}

	spot1 := make([]float32, 3)
	spot2 := make([]float32, 3)
	copy(spot1, self.s.Origin[:])
	spot1[2] += float32(self.viewheight)
	copy(spot2, other.s.Origin[:])
	spot2[2] += float32(other.viewheight)
	trace := G.gi.Trace(spot1, []float32{0, 0, 0}, []float32{0, 0, 0}, spot2, self, shared.MASK_OPAQUE)

	if trace.Fraction == 1.0 {
		return true
	}

	return false
}

/*
 * returns 1 if the entity is in
 * front (in sight) of self
 */
func infront(self, other *edict_t) bool {

	if self == nil || other == nil {
		return false
	}

	forward := make([]float32, 3)
	shared.AngleVectors(self.s.Angles[:], forward, nil, nil)

	vec := make([]float32, 3)
	shared.VectorSubtract(other.s.Origin[:], self.s.Origin[:], vec)
	shared.VectorNormalize(vec)
	dot := shared.DotProduct(vec, forward)

	if dot > 0.3 {
		return true
	}

	return false
}

/* ============================================================================ */

func (G *qGame) huntTarget(self *edict_t) {

	if self == nil {
		return
	}

	self.goalentity = self.enemy

	if (self.monsterinfo.aiflags & AI_STAND_GROUND) != 0 {
		self.monsterinfo.stand(self, G)
	} else {
		self.monsterinfo.run(self, G)
	}

	vec := make([]float32, 3)
	if G.visible(self, self.enemy) {
		shared.VectorSubtract(self.enemy.s.Origin[:], self.s.Origin[:], vec)
	}

	self.ideal_yaw = vectoyaw(vec)

	/* wait a while before first attack */
	if (self.monsterinfo.aiflags & AI_STAND_GROUND) == 0 {
		// 	AttackFinished(self, 1);
	}
}

func (G *qGame) foundTarget(self *edict_t) {
	if self == nil || self.enemy == nil || !self.enemy.inuse {
		return
	}

	/* let other monsters see this monster for a while */
	if self.enemy.client != nil {
		G.level.sight_entity = self
		G.level.sight_entity_framenum = G.level.framenum
		// G.level.sight_entity.light_level = 128
	}

	self.show_hostile = G.level.time + 1 /* wake up other monsters */

	copy(self.monsterinfo.last_sighting[:], self.enemy.s.Origin[:])
	// self.monsterinfo.trail_time = level.time

	if len(self.Combattarget) == 0 {
		G.huntTarget(self)
		return
	}

	self.movetarget = G.gPickTarget(self.Combattarget)
	self.goalentity = self.movetarget

	if self.movetarget == nil {
		self.movetarget = self.enemy
		self.goalentity = self.movetarget
		G.huntTarget(self)
		G.gi.Dprintf("%s at %s, combattarget %s not found\n",
			self.Classname,
			vtos(self.s.Origin[:]),
			self.Combattarget)
		return
	}

	/* clear out our combattarget, these are a one shot deal */
	self.Combattarget = ""
	self.monsterinfo.aiflags |= AI_COMBAT_POINT

	/* clear the targetname, that point is ours! */
	self.movetarget.Targetname = ""
	self.monsterinfo.pausetime = 0

	/* run for it */
	self.monsterinfo.run(self, G)
}

/*
 * Self is currently not attacking anything,
 * so try to find a target
 *
 * Returns TRUE if an enemy was sighted
 *
 * When a player fires a missile, the point
 * of impact becomes a fakeplayer so that
 * monsters that see the impact will respond
 * as if they had seen the player.
 *
 * To avoid spending too much time, only
 * a single client (or fakeclient) is
 * checked each frame. This means multi
 * player games will have slightly
 * slower noticing monsters.
 */
func (G *qGame) findTarget(self *edict_t) bool {

	if self == nil {
		return false
	}

	if (self.monsterinfo.aiflags & AI_GOOD_GUY) != 0 {
		return false
	}

	/* if we're going to a combat point, just proceed */
	if (self.monsterinfo.aiflags & AI_COMBAT_POINT) != 0 {
		return false
	}

	/* if the first spawnflag bit is set, the monster
	will only wake up on really seeing the player,
	not another monster getting angry or hearing
	something */

	heardit := false
	var client *edict_t

	if (G.level.sight_entity_framenum >= (G.level.framenum - 1)) &&
		(self.Spawnflags&1) == 0 {
		client = G.level.sight_entity

		if client.enemy == self.enemy {
			return false
		}
	} else if G.level.sound_entity_framenum >= (G.level.framenum - 1) {
		client = G.level.sound_entity
		heardit = true
	} else if self.enemy == nil &&
		(G.level.sound2_entity_framenum >= (G.level.framenum - 1)) &&
		(self.Spawnflags&1) == 0 {
		client = G.level.sound2_entity
		heardit = true
	} else {
		client = G.level.sight_client
		if client == nil {
			return false /* no clients to get mad at */
		}
	}

	/* if the entity went away, forget it */
	if !client.inuse {
		return false
	}

	if client == self.enemy {
		return true
	}

	if client.client != nil {
		if (client.flags & FL_NOTARGET) != 0 {
			return false
		}
	} else if (client.svflags & shared.SVF_MONSTER) != 0 {
		if client.enemy == nil {
			return false
		}

		if (client.enemy.flags & FL_NOTARGET) != 0 {
			return false
		}
	} else if heardit {
		if (client.owner.flags & FL_NOTARGET) != 0 {
			return false
		}
	} else {
		return false
	}

	if !heardit {
		r := range_(self, client)
		if r == RANGE_FAR {
			return false
		}

		/* is client in an spot too dark to be seen? */
		//  if (client.light_level <= 5) {
		// 	 return false;
		//  }

		if !G.visible(self, client) {
			return false
		}

		if r == RANGE_NEAR {
			if (client.show_hostile < G.level.time) && !infront(self, client) {
				return false
			}
		} else if r == RANGE_MID {
			if !infront(self, client) {
				return false
			}
		}

		self.enemy = client

		if self.enemy.Classname != "player_noise" {
			self.monsterinfo.aiflags &^= AI_SOUND_TARGET

			if self.enemy.client == nil {
				self.enemy = self.enemy.enemy

				if self.enemy.client == nil {
					self.enemy = nil
					return false
				}
			}
		}
	} else { /* heardit */

		if (self.Spawnflags & 1) != 0 {
			if !G.visible(self, client) {
				return false
			}
		} else {
			if !G.gi.InPHS(self.s.Origin[:], client.s.Origin[:]) {
				return false
			}
		}

		temp := make([]float32, 3)
		shared.VectorSubtract(client.s.Origin[:], self.s.Origin[:], temp)

		if shared.VectorLength(temp) > 1000 { /* too far to hear */
			return false
		}

		/* check area portals - if they are different
		and not connected then we can't hear it */
		if client.areanum != self.areanum {
			if !G.gi.AreasConnected(self.areanum, client.areanum) {
				return false
			}
		}

		self.ideal_yaw = vectoyaw(temp)
		M_ChangeYaw(self)

		/* hunt the sound for a bit; hopefully find the real player */
		self.monsterinfo.aiflags |= AI_SOUND_TARGET
		self.enemy = client
	}

	G.foundTarget(self)

	if (self.monsterinfo.aiflags&AI_SOUND_TARGET) == 0 &&
		self.monsterinfo.sight != nil {
		self.monsterinfo.sight(self, self.enemy, G)
	}

	return true
}

/* ============================================================================= */

func FacingIdeal(self *edict_t) bool {

	if self == nil {
		return false
	}

	delta := shared.Anglemod(self.s.Angles[shared.YAW] - self.ideal_yaw)

	if (delta > 45) && (delta < 315) {
		return false
	}

	return true
}

/* ============================================================================= */

func mCheckAttack(self *edict_t, G *qGame) bool {
	// vec3_t spot1, spot2;
	// float chance;
	// trace_t tr;

	if self == nil || self.enemy == nil || !self.enemy.inuse || G == nil {
		return false
	}

	if self.enemy.Health > 0 {
		/* see if any entities are in the way of the shot */
		spot1 := make([]float32, 3)
		copy(spot1, self.s.Origin[:])
		spot1[2] += float32(self.viewheight)
		spot2 := make([]float32, 3)
		copy(spot2, self.s.Origin[:])
		spot2[2] += float32(self.enemy.viewheight)

		tr := G.gi.Trace(spot1, nil, nil, spot2, self,
			shared.CONTENTS_SOLID|shared.CONTENTS_MONSTER|shared.CONTENTS_SLIME|
				shared.CONTENTS_LAVA|shared.CONTENTS_WINDOW)

		/* do we have a clear shot? */
		if tr.Ent != self.enemy {
			return false
		}
	}

	/* melee attack */
	if G.enemy_range == RANGE_MELEE {
		/* don't always melee in easy mode */
		if (G.skill.Int() == SKILL_EASY) && (shared.Randk()&3) != 0 {
			return false
		}

		if self.monsterinfo.melee != nil {
			self.monsterinfo.attack_state = AS_MELEE
		} else {
			self.monsterinfo.attack_state = AS_MISSILE
		}

		return true
	}

	/* missile attack */
	if self.monsterinfo.attack == nil {
		return false
	}

	if G.level.time < self.monsterinfo.attack_finished {
		return false
	}

	if G.enemy_range == RANGE_FAR {
		return false
	}

	var chance float32
	if (self.monsterinfo.aiflags & AI_STAND_GROUND) != 0 {
		chance = 0.4
	} else if G.enemy_range == RANGE_NEAR {
		chance = 0.1
	} else if G.enemy_range == RANGE_MID {
		chance = 0.02
	} else {
		return false
	}

	if G.skill.Int() == SKILL_EASY {
		chance *= 0.5
	} else if G.skill.Int() >= SKILL_HARD {
		chance *= 2
	}

	if shared.Frandk() < chance {
		self.monsterinfo.attack_state = AS_MISSILE
		self.monsterinfo.attack_finished = G.level.time + 2*shared.Frandk()
		return true
	}

	if (self.flags & FL_FLY) != 0 {
		if shared.Frandk() < 0.3 {
			self.monsterinfo.attack_state = AS_SLIDING
		} else {
			self.monsterinfo.attack_state = AS_STRAIGHT
		}
	}

	return false
}

/*
 * Turn and close until within an
 * angle to launch a melee attack
 */
func (G *qGame) ai_run_melee(self *edict_t) {
	if self == nil {
		return
	}

	self.ideal_yaw = G.enemy_yaw
	M_ChangeYaw(self)

	if FacingIdeal(self) {
		if self.monsterinfo.melee != nil {
			self.monsterinfo.melee(self, G)
			self.monsterinfo.attack_state = AS_STRAIGHT
		}
	}
}

/*
 * Turn in place until within an
 * angle to launch a missile attack
 */
func (G *qGame) ai_run_missile(self *edict_t) {
	if self == nil {
		return
	}

	self.ideal_yaw = G.enemy_yaw
	M_ChangeYaw(self)

	if FacingIdeal(self) {
		if self.monsterinfo.attack != nil {
			self.monsterinfo.attack(self, G)
			self.monsterinfo.attack_state = AS_STRAIGHT
		}
	}
}

/*
 * Decides if we're going to attack
 * or do something else used by
 * ai_run and ai_stand
 */
func (G *qGame) ai_checkattack(self *edict_t) bool {
	//  vec3_t temp;
	//  qboolean hesDeadJim;

	if self == nil {
		G.enemy_vis = false

		return false
	}

	/* this causes monsters to run blindly
	to the combat point w/o firing */
	if self.goalentity != nil {
		if (self.monsterinfo.aiflags & AI_COMBAT_POINT) != 0 {
			return false
		}

		if (self.monsterinfo.aiflags&AI_SOUND_TARGET) != 0 && !G.visible(self, self.goalentity) {
			// 		 if ((level.time - self->enemy->last_sound_time) > 5.0)
			// 		 {
			// 			 if (self->goalentity == self->enemy)
			// 			 {
			// 				 if (self->movetarget)
			// 				 {
			// 					 self->goalentity = self->movetarget;
			// 				 }
			// 				 else
			// 				 {
			// 					 self->goalentity = NULL;
			// 				 }
			// 			 }

			// 			 self->monsterinfo.aiflags &= ~AI_SOUND_TARGET;

			// 			 if (self->monsterinfo.aiflags & AI_TEMP_STAND_GROUND)
			// 			 {
			// 				 self->monsterinfo.aiflags &=
			// 						 ~(AI_STAND_GROUND | AI_TEMP_STAND_GROUND);
			// 			 }
			// 		 }
			// 		 else
			// 		 {
			// 			 self->show_hostile = level.time + 1;
			// 			 return false;
			//  }
		}
	}

	G.enemy_vis = false

	/* see if the enemy is dead */
	hesDeadJim := false

	if (self.enemy == nil) || (!self.enemy.inuse) {
		hesDeadJim = true
	} else if (self.monsterinfo.aiflags & AI_MEDIC) != 0 {
		if self.enemy.Health > 0 {
			hesDeadJim = true
			self.monsterinfo.aiflags &^= AI_MEDIC
		}
	} else {
		if (self.monsterinfo.aiflags & AI_BRUTAL) != 0 {
			if self.enemy.Health <= -80 {
				hesDeadJim = true
			}
		} else {
			if self.enemy.Health <= 0 {
				hesDeadJim = true
			}
		}
	}

	if hesDeadJim {
		self.enemy = nil

		if self.oldenemy != nil && (self.oldenemy.Health > 0) {
			self.enemy = self.oldenemy
			self.oldenemy = nil
			// 		 HuntTarget(self);
		} else {
			// 		 if (self.movetarget != nil) {
			// 			 self.goalentity = self.movetarget;
			// 			 self->monsterinfo.walk(self);
			// 		 } else {
			// 			 /* we need the pausetime otherwise the stand code
			// 				will just revert to walking with no target and
			// 				the monsters will wonder around aimlessly trying
			// 				to hunt the world entity */
			// 			 self.monsterinfo.pausetime = G.level.time + 100000000;
			// 			 self->monsterinfo.stand(self);
			// 		 }

			return true
		}
	}

	/* wake up other monsters */
	self.show_hostile = G.level.time + 1

	/* check knowledge of enemy */
	G.enemy_vis = G.visible(self, self.enemy)

	if G.enemy_vis {
		self.monsterinfo.search_time = G.level.time + 5
		copy(self.monsterinfo.last_sighting[:], self.enemy.s.Origin[:])
	}

	//  /* look for other coop players here */
	//  if (coop->value && (self->monsterinfo.search_time < level.time))
	//  {
	// 	 if (FindTarget(self))
	// 	 {
	// 		 return true;
	// 	 }
	//  }

	if self.enemy != nil {
		G.enemy_infront = infront(self, self.enemy)
		G.enemy_range = range_(self, self.enemy)
		temp := make([]float32, 3)
		shared.VectorSubtract(self.enemy.s.Origin[:], self.s.Origin[:], temp)
		G.enemy_yaw = vectoyaw(temp)
	}

	if self.monsterinfo.attack_state == AS_MISSILE {
		G.ai_run_missile(self)
		return true
	}

	if self.monsterinfo.attack_state == AS_MELEE {
		G.ai_run_melee(self)
		return true
	}

	/* if enemy is not currently visible,
	we will never attack */
	if !G.enemy_vis {
		return false
	}

	return self.monsterinfo.checkattack(self, G)
}

/*
 * The monster has an enemy
 * it is trying to kill
 */
func ai_run(self *edict_t, dist float32, G *qGame) {
	//  vec3_t v;
	//  edict_t *tempgoal;
	//  edict_t *save;
	//  qboolean new;
	//  edict_t *marker;
	//  float d1, d2;
	//  trace_t tr;
	//  vec3_t v_forward, v_right;
	//  float left, center, right;
	//  vec3_t left_target, right_target;

	if self == nil || G == nil {
		return
	}

	/* if we're going to a combat point, just proceed */
	if (self.monsterinfo.aiflags & AI_COMBAT_POINT) != 0 {
		G.mMoveToGoal(self, dist)
		return
	}

	if (self.monsterinfo.aiflags & AI_SOUND_TARGET) != 0 {
		/* Special case: Some projectiles like grenades or rockets are
		classified as an enemy. When they explode they generate a
		sound entity, triggering this code path. Since they're gone
		after the explosion their entity pointer is NULL. Therefor
		self->enemy is also NULL and we're crashing. Work around
		this by predending that the enemy is still there, and move
		to it. */
		// 	 if (self->enemy) {
		// 		 VectorSubtract(self->s.origin, self->enemy->s.origin, v);

		// 		 if (VectorLength(v) < 64) {
		// 			 self->monsterinfo.aiflags |= (AI_STAND_GROUND | AI_TEMP_STAND_GROUND);
		// 			 self->monsterinfo.stand(self);
		// 			 return;
		// 		 }
		// 	 }

		G.mMoveToGoal(self, dist)

		if !G.findTarget(self) {
			return
		}
	}

	if G.ai_checkattack(self) {
		return
	}

	//  if (self->monsterinfo.attack_state == AS_SLIDING) {
	// 	 ai_run_slide(self, dist);
	// 	 return;
	//  }

	if G.enemy_vis {
		G.mMoveToGoal(self, dist)
		self.monsterinfo.aiflags &^= AI_LOST_SIGHT
		copy(self.monsterinfo.last_sighting[:], self.enemy.s.Origin[:])
		self.monsterinfo.trail_time = G.level.time
		return
	}

	if (self.monsterinfo.search_time != 0) &&
		(G.level.time > (self.monsterinfo.search_time + 20)) {
		G.mMoveToGoal(self, dist)
		self.monsterinfo.search_time = 0
		return
	}

	save := self.goalentity
	tempgoal, _ := G.gSpawn()
	self.goalentity = tempgoal

	isNew := false

	if (self.monsterinfo.aiflags & AI_LOST_SIGHT) == 0 {
		/* just lost sight of the player, decide where to go first */
		self.monsterinfo.aiflags |= (AI_LOST_SIGHT | AI_PURSUIT_LAST_SEEN)
		self.monsterinfo.aiflags &^= (AI_PURSUE_NEXT | AI_PURSUE_TEMP)
		isNew = true
	}

	if (self.monsterinfo.aiflags & AI_PURSUE_NEXT) != 0 {
		self.monsterinfo.aiflags &^= AI_PURSUE_NEXT

		/* give ourself more time since we got this far */
		self.monsterinfo.search_time = G.level.time + 5

		var marker *edict_t = nil
		if (self.monsterinfo.aiflags & AI_PURSUE_TEMP) != 0 {
			self.monsterinfo.aiflags &^= AI_PURSUE_TEMP
			// 		 marker = NULL;
			// copy(self.monsterinfo.last_sighting[:], self.owner.monsterinfo.saved_goal[:])
			isNew = true
		} else if (self.monsterinfo.aiflags & AI_PURSUIT_LAST_SEEN) != 0 {
			self.monsterinfo.aiflags &^= AI_PURSUIT_LAST_SEEN
			// 		 marker = PlayerTrail_PickFirst(self);
		} else {
			// 		 marker = PlayerTrail_PickNext(self);
		}

		if marker != nil {
			// 		 VectorCopy(marker->s.origin, self->monsterinfo.last_sighting);
			// 		 self->monsterinfo.trail_time = marker->timestamp;
			// 		 self->s.angles[YAW] = self->ideal_yaw = marker->s.angles[YAW];
			isNew = true
		}
	}

	v := make([]float32, 3)
	shared.VectorSubtract(self.s.Origin[:], self.monsterinfo.last_sighting[:], v)
	d1 := shared.VectorLength(v)

	if d1 <= dist {
		self.monsterinfo.aiflags |= AI_PURSUE_NEXT
		dist = d1
	}

	copy(self.goalentity.s.Origin[:], self.monsterinfo.last_sighting[:])

	if isNew {
		println("isNew")
		// 	 tr = gi.trace(self->s.origin, self->mins, self->maxs,
		// 			 self->monsterinfo.last_sighting, self,
		// 			 MASK_PLAYERSOLID);

		// 	 if (tr.fraction < 1)
		// 	 {
		// 		 VectorSubtract(self->goalentity->s.origin, self->s.origin, v);
		// 		 d1 = VectorLength(v);
		// 		 center = tr.fraction;
		// 		 d2 = d1 * ((center + 1) / 2);
		// 		 self->s.angles[YAW] = self->ideal_yaw = vectoyaw(v);
		// 		 AngleVectors(self->s.angles, v_forward, v_right, NULL);

		// 		 VectorSet(v, d2, -16, 0);
		// 		 G_ProjectSource(self->s.origin, v, v_forward, v_right, left_target);
		// 		 tr = gi.trace(self->s.origin, self->mins, self->maxs, left_target,
		// 				 self, MASK_PLAYERSOLID);
		// 		 left = tr.fraction;

		// 		 VectorSet(v, d2, 16, 0);
		// 		 G_ProjectSource(self->s.origin, v, v_forward, v_right, right_target);
		// 		 tr = gi.trace(self->s.origin, self->mins, self->maxs, right_target,
		// 				 self, MASK_PLAYERSOLID);
		// 		 right = tr.fraction;

		// 		 center = (d1 * center) / d2;

		// 		 if ((left >= center) && (left > right))
		// 		 {
		// 			 if (left < 1)
		// 			 {
		// 				 VectorSet(v, d2 * left * 0.5, -16, 0);
		// 				 G_ProjectSource(self->s.origin, v, v_forward,
		// 						 v_right, left_target);
		// 			 }

		// 			 VectorCopy(self->monsterinfo.last_sighting,
		// 					 self->monsterinfo.saved_goal);
		// 			 self->monsterinfo.aiflags |= AI_PURSUE_TEMP;
		// 			 VectorCopy(left_target, self->goalentity->s.origin);
		// 			 VectorCopy(left_target, self->monsterinfo.last_sighting);
		// 			 VectorSubtract(self->goalentity->s.origin, self->s.origin, v);
		// 			 self->s.angles[YAW] = self->ideal_yaw = vectoyaw(v);
		// 		 }
		// 		 else if ((right >= center) && (right > left))
		// 		 {
		// 			 if (right < 1)
		// 			 {
		// 				 VectorSet(v, d2 * right * 0.5, 16, 0);
		// 				 G_ProjectSource(self->s.origin, v, v_forward, v_right,
		// 						 right_target);
		// 			 }

		// 			 VectorCopy(self->monsterinfo.last_sighting,
		// 					 self->monsterinfo.saved_goal);
		// 			 self->monsterinfo.aiflags |= AI_PURSUE_TEMP;
		// 			 VectorCopy(right_target, self->goalentity->s.origin);
		// 			 VectorCopy(right_target, self->monsterinfo.last_sighting);
		// 			 VectorSubtract(self->goalentity->s.origin, self->s.origin, v);
		// 			 self->s.angles[YAW] = self->ideal_yaw = vectoyaw(v);
		// 		 }
		// 	 }
	}

	G.mMoveToGoal(self, dist)

	G.gFreeEdict(tempgoal)

	self.goalentity = save
}
