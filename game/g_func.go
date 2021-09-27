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
 * Level functions. Platforms, buttons, dooors and so on.
 *
 * =======================================================================
 */
package game

import (
	"math"
	"quake2srv/shared"
)

const (
	STATE_TOP    = 0
	STATE_BOTTOM = 1
	STATE_UP     = 2
	STATE_DOWN   = 3

	DOOR_START_OPEN = 1
	DOOR_REVERSE    = 2
	DOOR_CRUSHER    = 4
	DOOR_NOMONSTER  = 8
	DOOR_TOGGLE     = 32
	DOOR_X_AXIS     = 64
	DOOR_Y_AXIS     = 128
)

func move_Done(ent *edict_t, G *qGame) {
	if ent == nil || G == nil {
		return
	}

	copy(ent.velocity[:], []float32{0, 0, 0})
	ent.moveinfo.endfunc(ent, G)
}

func move_Final(ent *edict_t, G *qGame) {
	if ent == nil || G == nil {
		return
	}

	if ent.moveinfo.remaining_distance == 0 {
		move_Done(ent, G)
		return
	}

	shared.VectorScale(ent.moveinfo.dir[:],
		ent.moveinfo.remaining_distance/FRAMETIME,
		ent.velocity[:])

	ent.think = move_Done
	ent.nextthink = G.level.time + FRAMETIME
}

func move_Begin(ent *edict_t, G *qGame) {

	if ent == nil || G == nil {
		return
	}

	if (ent.moveinfo.speed * FRAMETIME) >= ent.moveinfo.remaining_distance {
		move_Final(ent, G)
		return
	}

	shared.VectorScale(ent.moveinfo.dir[:], ent.moveinfo.speed, ent.velocity[:])
	frames := float32(math.Floor(float64(
		(ent.moveinfo.remaining_distance /
			ent.moveinfo.speed) / FRAMETIME)))
	ent.moveinfo.remaining_distance -= frames * ent.moveinfo.speed *
		FRAMETIME
	ent.nextthink = G.level.time + (frames * FRAMETIME)
	ent.think = move_Final
}

func (G *qGame) move_Calc(ent *edict_t, dest []float32, f func(*edict_t, *qGame)) {
	if ent == nil || f == nil {
		return
	}

	copy(ent.velocity[:], []float32{0, 0, 0})
	shared.VectorSubtract(dest, ent.s.Origin[:], ent.moveinfo.dir[:])
	ent.moveinfo.remaining_distance = shared.VectorNormalize(ent.moveinfo.dir[:])
	ent.moveinfo.endfunc = f

	if (ent.moveinfo.speed == ent.moveinfo.accel) &&
		(ent.moveinfo.speed == ent.moveinfo.decel) {
		if ((ent.flags&FL_TEAMSLAVE) != 0 && G.level.current_entity == ent.teammaster) ||
			((ent.flags&FL_TEAMSLAVE) == 0 && G.level.current_entity == ent) {
			move_Begin(ent, G)
		} else {
			ent.nextthink = G.level.time + FRAMETIME
			ent.think = move_Begin
		}
	} else {
		/* accelerative */
		ent.moveinfo.current_speed = 0
		ent.think = think_AccelMove
		ent.nextthink = G.level.time + FRAMETIME
	}
}

/* Support routines for angular movement (changes in angle using avelocity) */

func angleMove_Done(ent *edict_t, G *qGame) {
	if ent == nil || G == nil {
		return
	}

	copy(ent.avelocity[:], []float32{0, 0, 0})
	ent.moveinfo.endfunc(ent, G)
}

func angleMove_Final(ent *edict_t, G *qGame) {

	if ent == nil || G == nil {
		return
	}

	move := make([]float32, 3)
	if ent.moveinfo.state == STATE_UP {
		shared.VectorSubtract(ent.moveinfo.end_angles[:], ent.s.Angles[:], move)
	} else {
		shared.VectorSubtract(ent.moveinfo.start_angles[:], ent.s.Angles[:], move)
	}

	if shared.VectorCompare(move, []float32{0, 0, 0}) != 0 {
		angleMove_Done(ent, G)
		return
	}

	shared.VectorScale(move, 1.0/FRAMETIME, ent.avelocity[:])

	ent.think = angleMove_Done
	ent.nextthink = G.level.time + FRAMETIME
}

func angleMove_Begin(ent *edict_t, G *qGame) {

	if ent == nil || G == nil {
		return
	}

	/* set destdelta to the vector needed to move */
	destdelta := make([]float32, 3)
	if ent.moveinfo.state == STATE_UP {
		shared.VectorSubtract(ent.moveinfo.end_angles[:], ent.s.Angles[:], destdelta)
	} else {
		shared.VectorSubtract(ent.moveinfo.start_angles[:], ent.s.Angles[:], destdelta)
	}

	/* calculate length of vector */
	len := shared.VectorLength(destdelta)

	/* divide by speed to get time to reach dest */
	traveltime := len / ent.moveinfo.speed

	if traveltime < FRAMETIME {
		angleMove_Final(ent, G)
		return
	}

	frames := float32(math.Floor(float64(traveltime / FRAMETIME)))

	/* scale the destdelta vector by the time spent traveling to get velocity */
	shared.VectorScale(destdelta, 1.0/traveltime, ent.avelocity[:])

	/* set nextthink to trigger a think when dest is reached */
	ent.nextthink = G.level.time + frames*FRAMETIME
	ent.think = angleMove_Final
}

func (G *qGame) angleMove_Calc(ent *edict_t, f func(*edict_t, *qGame)) {

	if ent == nil || f == nil {
		return
	}

	copy(ent.avelocity[:], []float32{0, 0, 0})
	ent.moveinfo.endfunc = f

	if (((ent.flags & FL_TEAMSLAVE) != 0) && G.level.current_entity == ent.teammaster) ||
		(((ent.flags & FL_TEAMSLAVE) == 0) && G.level.current_entity == ent) {
		// if (level.current_entity == ((ent.flags & FL_TEAMSLAVE) ? ent.teammaster : ent)) {
		angleMove_Begin(ent, G)
	} else {
		ent.nextthink = G.level.time + FRAMETIME
		ent.think = angleMove_Begin
	}
}

func accelerationDistance(target, rate float32) float32 { return (target * ((target / rate) + 1) / 2) }

func (G *qGame) plat_CalcAcceleratedMove(moveinfo *moveinfo_t) {
	// float accel_dist;
	// float decel_dist;

	if moveinfo == nil {
		return
	}

	moveinfo.move_speed = moveinfo.speed

	if moveinfo.remaining_distance < moveinfo.accel {
		moveinfo.current_speed = moveinfo.remaining_distance
		return
	}

	accel_dist := accelerationDistance(moveinfo.speed, moveinfo.accel)
	decel_dist := accelerationDistance(moveinfo.speed, moveinfo.decel)

	if (moveinfo.remaining_distance - accel_dist - decel_dist) < 0 {
		// float f;

		f :=
			(moveinfo.accel +
				moveinfo.decel) / (moveinfo.accel * moveinfo.decel)
		moveinfo.move_speed =
			(-2 +
				float32(math.Sqrt(float64(4-4*f*(-2*moveinfo.remaining_distance))))) / (2 * f)
		decel_dist = accelerationDistance(moveinfo.move_speed, moveinfo.decel)
	}

	moveinfo.decel_distance = decel_dist
}

func (G *qGame) plat_Accelerate(moveinfo *moveinfo_t) {
	if moveinfo == nil {
		return
	}

	/* are we decelerating? */
	if moveinfo.remaining_distance <= moveinfo.decel_distance {
		if moveinfo.remaining_distance < moveinfo.decel_distance {
			if moveinfo.next_speed != 0 {
				moveinfo.current_speed = moveinfo.next_speed
				moveinfo.next_speed = 0
				return
			}

			if moveinfo.current_speed > moveinfo.decel {
				moveinfo.current_speed -= moveinfo.decel
			}
		}

		return
	}

	/* are we at full speed and need to start decelerating during this move? */
	if moveinfo.current_speed == moveinfo.move_speed {
		if (moveinfo.remaining_distance - moveinfo.current_speed) <
			moveinfo.decel_distance {

			p1_distance := moveinfo.remaining_distance -
				moveinfo.decel_distance
			p2_distance := moveinfo.move_speed *
				(1.0 - (p1_distance / moveinfo.move_speed))
			distance := p1_distance + p2_distance
			moveinfo.current_speed = moveinfo.move_speed
			moveinfo.next_speed = moveinfo.move_speed - moveinfo.decel*
				(p2_distance/distance)
			return
		}
	}

	/* are we accelerating? */
	if moveinfo.current_speed < moveinfo.speed {

		old_speed := moveinfo.current_speed

		/* figure simple acceleration up to move_speed */
		moveinfo.current_speed += moveinfo.accel

		if moveinfo.current_speed > moveinfo.speed {
			moveinfo.current_speed = moveinfo.speed
		}

		/* are we accelerating throughout this entire move? */
		if (moveinfo.remaining_distance - moveinfo.current_speed) >=
			moveinfo.decel_distance {
			return
		}

		/* during this move we will accelrate from current_speed to move_speed
		   and cross over the decel_distance; figure the average speed for the
		   entire move */
		p1_distance := moveinfo.remaining_distance - moveinfo.decel_distance
		p1_speed := (old_speed + moveinfo.move_speed) / 2.0
		p2_distance := moveinfo.move_speed * (1.0 - (p1_distance / p1_speed))
		distance := p1_distance + p2_distance
		moveinfo.current_speed =
			(p1_speed *
				(p1_distance /
					distance)) + (moveinfo.move_speed * (p2_distance / distance))
		moveinfo.next_speed = moveinfo.move_speed - moveinfo.decel*
			(p2_distance/distance)
		return
	}

	/* we are at constant velocity (move_speed) */
	return
}

/*
 * The team has completed a frame of movement,
 * so change the speed for the next frame
 */
func think_AccelMove(ent *edict_t, G *qGame) {
	if ent == nil || G == nil {
		return
	}

	ent.moveinfo.remaining_distance -= ent.moveinfo.current_speed

	if ent.moveinfo.current_speed == 0 { /* starting or blocked */
		G.plat_CalcAcceleratedMove(&ent.moveinfo)
	}

	G.plat_Accelerate(&ent.moveinfo)

	/* will the entire move complete on next frame? */
	if ent.moveinfo.remaining_distance <= ent.moveinfo.current_speed {
		move_Final(ent, G)
		return
	}

	shared.VectorScale(ent.moveinfo.dir[:], ent.moveinfo.current_speed*10,
		ent.velocity[:])
	ent.nextthink = G.level.time + FRAMETIME
	ent.think = think_AccelMove
}

/* ==================================================================== */

/*
 * DOORS
 *
 * spawn a trigger surrounding the entire team
 * unless it is already targeted by another
 */

//  void door_go_down(edict_t *self);

/*
 * QUAKED func_door (0 .5 .8) ? START_OPEN x CRUSHER NOMONSTER ANIMATED TOGGLE ANIMATED_FAST
 *
 * TOGGLE		wait in both the start and end states for a trigger event.
 * START_OPEN	the door to moves to its destination when spawned, and operate in reverse.
 *              It is used to temporarily or permanently close off an area when triggered
 *              (not useful for touch or takedamage doors).
 * NOMONSTER	monsters will not trigger this door
 *
 * "message"	is printed when the door is touched if it is a trigger door and it hasn't been fired yet
 * "angle"		determines the opening direction
 * "targetname" if set, no touch field will be spawned and a remote button or trigger field activates the door.
 * "health"	    if set, door must be shot open
 * "speed"		movement speed (100 default)
 * "wait"		wait before returning (3 default, -1 = never return)
 * "lip"		lip remaining at end of move (8 default)
 * "dmg"		damage to inflict when blocked (2 default)
 * "sounds"
 *    1)	silent
 *    2)	light
 *    3)	medium
 *    4)	heavy
 */

func (G *qGame) door_use_areaportals(self *edict_t, open bool) {

	if self == nil {
		return
	}

	if len(self.Target) == 0 {
		return
	}

	var t *edict_t = nil
	for {
		t = G.gFind(t, "Targetname", self.Target)
		if t == nil {
			break
		}
		if t.Classname == "func_areaportal" {
			G.gi.SetAreaPortalState(t.Style, open)
		}
	}
}

func door_hit_top(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if (self.flags & FL_TEAMSLAVE) == 0 {
		//  if (self->moveinfo.sound_end) {
		// 	 gi.sound(self, CHAN_NO_PHS_ADD + CHAN_VOICE, self->moveinfo.sound_end,
		// 			 1, ATTN_STATIC, 0);
		//  }

		//  self->s.sound = 0;
	}

	self.moveinfo.state = STATE_TOP

	if (self.Spawnflags & DOOR_TOGGLE) != 0 {
		return
	}

	if self.moveinfo.wait >= 0 {
		self.think = door_go_down
		self.nextthink = G.level.time + self.moveinfo.wait
	}
}

func door_hit_bottom(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if (self.flags & FL_TEAMSLAVE) == 0 {
		//  if (self->moveinfo.sound_end) {
		// 	 gi.sound(self, CHAN_NO_PHS_ADD + CHAN_VOICE,
		// 			 self->moveinfo.sound_end, 1,
		// 			 ATTN_STATIC, 0);
		//  }

		//  self->s.sound = 0;
	}

	self.moveinfo.state = STATE_BOTTOM
	G.door_use_areaportals(self, false)
}

func door_go_down(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if (self.flags & FL_TEAMSLAVE) == 0 {
		// if (self->moveinfo.sound_start) {
		// 	gi.sound(self, CHAN_NO_PHS_ADD + CHAN_VOICE,
		// 			self->moveinfo.sound_start, 1,
		// 			ATTN_STATIC, 0);
		// }

		// self->s.sound = self->moveinfo.sound_middle;
	}

	if self.max_health != 0 {
		self.takedamage = DAMAGE_YES
		self.Health = self.max_health
	}

	self.moveinfo.state = STATE_DOWN

	if self.Classname == "func_door" {
		G.move_Calc(self, self.moveinfo.start_origin[:], door_hit_bottom)
	} else if self.Classname == "func_door_rotating" {
		G.angleMove_Calc(self, door_hit_bottom)
	}
}

func (G *qGame) door_go_up(self, activator *edict_t) {
	if self == nil || activator == nil {
		return
	}

	if self.moveinfo.state == STATE_UP {
		return /* already going up */
	}

	if self.moveinfo.state == STATE_TOP {
		/* reset top wait time */
		if self.moveinfo.wait >= 0 {
			self.nextthink = G.level.time + self.moveinfo.wait
		}

		return
	}

	if (self.flags & FL_TEAMSLAVE) == 0 {
		// if (self->moveinfo.sound_start) {
		// 	gi.sound(self, CHAN_NO_PHS_ADD + CHAN_VOICE,
		// 			self->moveinfo.sound_start, 1,
		// 			ATTN_STATIC, 0);
		// }

		// self->s.sound = self->moveinfo.sound_middle;
	}

	self.moveinfo.state = STATE_UP

	if self.Classname == "func_door" {
		G.move_Calc(self, self.moveinfo.end_origin[:], door_hit_top)
	} else if self.Classname == "func_door_rotating" {
		G.angleMove_Calc(self, door_hit_top)
	}

	G.gUseTargets(self, activator)
	G.door_use_areaportals(self, true)
}

func door_use(self, other, activator *edict_t, G *qGame) {
	if self == nil || activator == nil || G == nil {
		return
	}

	if (self.flags & FL_TEAMSLAVE) != 0 {
		return
	}

	if (self.Spawnflags & DOOR_TOGGLE) != 0 {
		if (self.moveinfo.state == STATE_UP) ||
			(self.moveinfo.state == STATE_TOP) {
			/* trigger all paired doors */
			for ent := self; ent != nil; ent = ent.teamchain {
				ent.Message = ""
				ent.touch = nil
				door_go_down(ent, G)
			}

			return
		}
	}

	/* trigger all paired doors */
	for ent := self; ent != nil; ent = ent.teamchain {
		ent.Message = ""
		ent.touch = nil
		G.door_go_up(ent, activator)
	}
}

func touch_DoorTrigger(self, other *edict_t, plane *shared.Cplane_t, surf *shared.Csurface_t, G *qGame) {
	if self == nil || other == nil || G == nil {
		return
	}

	if other.Health <= 0 {
		return
	}

	if (other.svflags&shared.SVF_MONSTER) == 0 && (other.client == nil) {
		return
	}

	if (self.owner.Spawnflags&DOOR_NOMONSTER) != 0 &&
		(other.svflags&shared.SVF_MONSTER) != 0 {
		return
	}

	if G.level.time < self.touch_debounce_time {
		return
	}

	self.touch_debounce_time = G.level.time + 1.0

	door_use(self.owner, other, other, G)
}

func think_CalcMoveSpeed(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	if (self.flags & FL_TEAMSLAVE) != 0 {
		return /* only the team master does this */
	}

	/* find the smallest distance any member of the team will be moving */
	min := float32(math.Abs(float64(self.moveinfo.distance)))

	for ent := self.teamchain; ent != nil; ent = ent.teamchain {
		dist := float32(math.Abs(float64(ent.moveinfo.distance)))

		if dist < min {
			min = dist
		}
	}

	time := min / self.moveinfo.speed

	/* adjust speeds so they will all complete at the same time */
	for ent := self; ent != nil; ent = ent.teamchain {
		newspeed := float32(math.Abs(float64(ent.moveinfo.distance))) / time
		ratio := newspeed / ent.moveinfo.speed

		if ent.moveinfo.accel == ent.moveinfo.speed {
			ent.moveinfo.accel = newspeed
		} else {
			ent.moveinfo.accel *= ratio
		}

		if ent.moveinfo.decel == ent.moveinfo.speed {
			ent.moveinfo.decel = newspeed
		} else {
			ent.moveinfo.decel *= ratio
		}

		ent.moveinfo.speed = newspeed
	}
}

func think_SpawnDoorTrigger(ent *edict_t, G *qGame) {

	if ent == nil || G == nil {
		return
	}

	if (ent.flags & FL_TEAMSLAVE) != 0 {
		return /* only the team leader spawns a trigger */
	}

	mins := make([]float32, 3)
	maxs := make([]float32, 3)
	copy(mins, ent.absmin[:])
	copy(maxs, ent.absmax[:])

	// for other := ent.teamchain; other != nil; other = other.teamchain {
	// 	AddPointToBounds(other->absmin, mins, maxs);
	// 	AddPointToBounds(other->absmax, mins, maxs);
	// }

	/* expand */
	mins[0] -= 60
	mins[1] -= 60
	maxs[0] += 60
	maxs[1] += 60

	other, _ := G.gSpawn()
	copy(other.mins[:], mins)
	copy(other.maxs[:], maxs)
	other.owner = ent
	other.solid = shared.SOLID_TRIGGER
	other.movetype = MOVETYPE_NONE
	other.touch = touch_DoorTrigger
	G.gi.Linkentity(other)

	if (ent.Spawnflags & DOOR_START_OPEN) != 0 {
		G.door_use_areaportals(ent, true)
	}

	think_CalcMoveSpeed(ent, G)
}

/*
 * =========================================================
 *
 * PLATS
 *
 * movement options:
 *
 * linear
 * smooth start, hard stop
 * smooth start, smooth stop
 *
 * start
 * end
 * acceleration
 * speed
 * deceleration
 * begin sound
 * end sound
 * target fired when reaching end
 * wait at end
 *
 * object characteristics that use move segments
 * ---------------------------------------------
 * movetype_push, or movetype_stop
 * action when touched
 * action when blocked
 * action when used
 *  disabled?
 * auto trigger spawning
 *
 *
 * =========================================================
 */

func spFuncDoor(ent *edict_t, G *qGame) error {

	if ent == nil || G == nil {
		return nil
	}

	// if (ent.sounds != 1)
	// {
	// 	ent->moveinfo.sound_start = gi.soundindex("doors/dr1_strt.wav");
	// 	ent->moveinfo.sound_middle = gi.soundindex("doors/dr1_mid.wav");
	// 	ent->moveinfo.sound_end = gi.soundindex("doors/dr1_end.wav");
	// }

	gSetMovedir(ent.s.Angles[:], ent.movedir[:])
	ent.movetype = MOVETYPE_PUSH
	ent.solid = shared.SOLID_BSP
	G.gi.Setmodel(ent, ent.Model)

	// ent->blocked = door_blocked;
	ent.use = door_use

	if ent.Speed == 0 {
		ent.Speed = 100
	}

	// if (deathmatch->value)
	// {
	// 	ent->speed *= 2;
	// }

	if ent.Accel == 0 {
		ent.Accel = ent.Speed
	}

	if ent.Decel == 0 {
		ent.Decel = ent.Speed
	}

	if ent.Wait == 0 {
		ent.Wait = 3
	}

	if G.st.Lip == 0 {
		G.st.Lip = 8
	}

	if ent.Dmg == 0 {
		ent.Dmg = 2
	}

	/* calculate second position */
	copy(ent.pos1[:], ent.s.Origin[:])
	abs_movedir := []float32{
		float32(math.Abs(float64(ent.movedir[0]))),
		float32(math.Abs(float64(ent.movedir[1]))),
		float32(math.Abs(float64(ent.movedir[2])))}
	ent.moveinfo.distance = abs_movedir[0]*ent.size[0] + abs_movedir[1]*
		ent.size[1] + abs_movedir[2]*ent.size[2] - float32(G.st.Lip)
	shared.VectorMA(ent.pos1[:], ent.moveinfo.distance, ent.movedir[:], ent.pos2[:])

	/* if it starts open, switch the positions */
	if (ent.Spawnflags & DOOR_START_OPEN) != 0 {
		copy(ent.s.Origin[:], ent.pos2[:])
		copy(ent.pos2[:], ent.pos1[:])
		copy(ent.pos1[:], ent.s.Origin[:])
	}

	ent.moveinfo.state = STATE_BOTTOM

	if ent.Health != 0 {
		ent.takedamage = DAMAGE_YES
		// 	ent->die = door_killed;
		ent.max_health = ent.Health
		// } else if (ent->targetname && ent->message) {
		// 	gi.soundindex("misc/talk.wav");
		// 	ent->touch = door_touch;
	}

	ent.moveinfo.speed = ent.Speed
	ent.moveinfo.accel = ent.Accel
	ent.moveinfo.decel = ent.Decel
	ent.moveinfo.wait = ent.Wait
	copy(ent.moveinfo.start_origin[:], ent.pos1[:])
	copy(ent.moveinfo.start_angles[:], ent.s.Angles[:])
	copy(ent.moveinfo.end_origin[:], ent.pos2[:])
	copy(ent.moveinfo.end_angles[:], ent.s.Angles[:])

	if (ent.Spawnflags & 16) != 0 {
		ent.s.Effects |= shared.EF_ANIM_ALL
	}

	if (ent.Spawnflags & 64) != 0 {
		ent.s.Effects |= shared.EF_ANIM_ALLFAST
	}

	/* to simplify logic elsewhere, make non-teamed doors into a team of one */
	if len(ent.Team) == 0 {
		ent.teammaster = ent
	}

	G.gi.Linkentity(ent)

	ent.nextthink = G.level.time + FRAMETIME

	if ent.Health != 0 || len(ent.Targetname) > 0 {
		ent.think = think_CalcMoveSpeed
	} else {
		ent.think = think_SpawnDoorTrigger
	}

	// /* Map quirk for waste3 (to make that secret armor behind
	//  * the secret wall - this func_door - count, #182) */
	// if (Q_stricmp(level.mapname, "waste3") == 0 && Q_stricmp(ent->model, "*12") == 0)
	// {
	// 	ent->target = "t117";
	// }
	return nil
}

/* ==================================================================== */

/*
 * QUAKED func_timer (0.3 0.1 0.6) (-8 -8 -8) (8 8 8) START_ON
 *
 * "wait"	base time between triggering all targets, default is 1
 * "random"	wait variance, default is 0
 *
 * so, the basic time between firing is a random time
 * between (wait - random) and (wait + random)
 *
 * "delay"			delay before first firing when turned on, default is 0
 * "pausetime"		additional delay used only the very first time
 *                  and only if spawned with START_ON
 *
 * These can used but not touched.
 */
func func_timer_think(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	G.gUseTargets(self, self.activator)
	self.nextthink = G.level.time + self.Wait + shared.Crandk()*self.Random
}

func spFuncTimer(self *edict_t, G *qGame) error {
	if self == nil || G == nil {
		return nil
	}

	if self.Wait == 0 {
		self.Wait = 1.0
	}

	// self.use = func_timer_use
	self.think = func_timer_think

	if self.Random >= self.Wait {
		self.Random = self.Wait - FRAMETIME
		G.gi.Dprintf("func_timer at %s has random >= wait\n", vtos(self.s.Origin[:]))
	}

	if (self.Spawnflags & 1) != 0 {
		self.nextthink = G.level.time + 1.0 + G.st.pausetime + self.Delay +
			self.Wait + shared.Crandk()*self.Random
		self.activator = self
	}

	self.svflags = shared.SVF_NOCLIENT
	return nil
}
