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
 * Quake IIs legendary physic engine.
 *
 * =======================================================================
 */
package game

import (
	"quake2srv/shared"
)

const STOP_EPSILON = 0.1
const MAX_CLIP_PLANES = 5
const STOPSPEED = 100
const FRICTION = 6
const WATERFRICTION = 1

/*
 * pushmove objects do not obey gravity, and do not interact
 * with each other or trigger fields, but block normal movement
 * and push normal objects when they move.
 *
 * onground is set for toss objects when they come to a complete
 * rest. It is set for steping or walking objects.
 *
 * doors, plats, etc are SOLID_BSP, and MOVETYPE_PUSH
 * bonus items are SOLID_TRIGGER touch, and MOVETYPE_TOSS
 * corpses are SOLID_NOT and MOVETYPE_TOSS
 * crates are SOLID_BBOX and MOVETYPE_TOSS
 * walking monsters are SOLID_SLIDEBOX and MOVETYPE_STEP
 * flying/floating monsters are SOLID_SLIDEBOX and MOVETYPE_FLY
 *
 * solid_edge items only clip against bsp models.
 */

func (G *qGame) svTestEntityPosition(ent *edict_t) *edict_t {
	if ent == nil {
		return nil
	}

	mask := shared.MASK_SOLID
	if ent.clipmask != 0 {
		mask = ent.clipmask
	}

	trace := G.gi.Trace(ent.s.Origin[:], ent.mins[:], ent.maxs[:], ent.s.Origin[:], ent, mask)

	if trace.Startsolid {
		if (ent.svflags&shared.SVF_DEADMONSTER) != 0 && (trace.Ent.(*edict_t).client != nil || (trace.Ent.(*edict_t).svflags&shared.SVF_MONSTER) != 0) {
			return nil
		}

		return &G.g_edicts[0]
	}

	return nil
}

func (G *qGame) svCheckVelocity(ent *edict_t) {
	if ent == nil {
		return
	}

	if shared.VectorLength(ent.velocity[:]) > G.sv_maxvelocity.Float() {
		shared.VectorNormalize(ent.velocity[:])
		shared.VectorScale(ent.velocity[:], G.sv_maxvelocity.Float(), ent.velocity[:])
	}
}

/*
 * Runs thinking code for
 * this frame if necessary
 */
func (G *qGame) svRunThink(ent *edict_t) bool {

	if ent == nil {
		return false
	}

	thinktime := ent.nextthink

	if thinktime <= 0 {
		return true
	}

	if thinktime > G.level.time+0.001 {
		return true
	}

	ent.nextthink = 0

	if ent.think == nil {
		G.gi.Error("NULL ent->think %v", ent.Classname)
	}

	ent.think(ent, G)

	return false
}

/*
 * Two entities have touched, so
 * run their touch functions
 */
func (G *qGame) svImpact(e1 *edict_t, trace *shared.Trace_t) {

	if e1 == nil || trace == nil {
		return
	}

	e2 := trace.Ent.(*edict_t)

	if e1.touch != nil && (e1.solid != shared.SOLID_NOT) {
		e1.touch(e1, e2, &trace.Plane, trace.Surface, G)
	}

	if e2.touch != nil && (e2.solid != shared.SOLID_NOT) {
		e2.touch(e2, e1, nil, nil, G)
	}
}

/*
 * Slide off of the impacting object
 * returns the blocked flags (1 = floor,
 * 2 = step / wall)
 */
func (G *qGame) clipVelocity(in, normal, out []float32, overbounce float32) int {

	blocked := 0

	if normal[2] > 0 {
		blocked |= 1 /* floor */
	}

	if normal[2] == 0 {
		blocked |= 2 /* step */
	}

	backoff := shared.DotProduct(in, normal) * overbounce

	for i := 0; i < 3; i++ {
		change := normal[i] * backoff
		out[i] = in[i] - change

		if (out[i] > -STOP_EPSILON) && (out[i] < STOP_EPSILON) {
			out[i] = 0
		}
	}

	return blocked
}

/*
 * The basic solid body movement clip
 * that slides along multiple planes
 * Returns the clipflags if the velocity
 * was modified (hit something solid)
 *
 * 1 = floor
 * 2 = wall / step
 * 4 = dead stop
 */
func (G *qGame) svFlyMove(ent *edict_t, time float32, mask int) int {

	if ent == nil {
		return 0
	}

	numbumps := 4

	blocked := 0
	original_velocity := make([]float32, 3)
	primal_velocity := make([]float32, 3)
	copy(original_velocity, ent.velocity[:])
	copy(primal_velocity, ent.velocity[:])
	numplanes := 0

	time_left := time

	ent.groundentity = nil

	planes := make([][]float32, MAX_CLIP_PLANES)

	for bumpcount := 0; bumpcount < numbumps; bumpcount++ {
		end := make([]float32, 3)
		for i := 0; i < 3; i++ {
			end[i] = ent.s.Origin[i] + time_left*ent.velocity[i]
		}

		trace := G.gi.Trace(ent.s.Origin[:], ent.mins[:], ent.maxs[:], end, ent, mask)

		if trace.Allsolid {
			/* entity is trapped in another solid */
			copy(ent.velocity[:], []float32{0, 0, 0})
			return 3
		}

		if trace.Fraction > 0 {
			/* actually covered some distance */
			copy(ent.s.Origin[:], trace.Endpos[:])
			copy(original_velocity, ent.velocity[:])
			numplanes = 0
		}

		if trace.Fraction == 1 {
			break /* moved the entire distance */
		}

		hit := trace.Ent.(*edict_t)

		if trace.Plane.Normal[2] > 0.7 {
			blocked |= 1 /* floor */

			if hit.solid == shared.SOLID_BSP {
				ent.groundentity = hit
				ent.groundentity_linkcount = hit.linkcount
			}
		}

		if trace.Plane.Normal[2] == 0 {
			blocked |= 2 /* step */
		}

		/* run the impact function */
		G.svImpact(ent, &trace)

		if !ent.inuse {
			break /* removed by the impact function */
		}

		time_left -= time_left * trace.Fraction

		/* cliped to another plane */
		if numplanes >= MAX_CLIP_PLANES {
			/* this shouldn't really happen */
			copy(ent.velocity[:], []float32{0, 0, 0})
			return 3
		}

		planes[numplanes] = make([]float32, 3)
		copy(planes[numplanes], trace.Plane.Normal[:])
		numplanes++

		/* modify original_velocity so it
		parallels all of the clip planes */
		i := 0
		new_velocity := make([]float32, 3)
		for i = 0; i < numplanes; i++ {
			G.clipVelocity(original_velocity, planes[i], new_velocity, 1)

			j := 0
			for j = 0; j < numplanes; j++ {
				if (j != i) && shared.VectorCompare(planes[i], planes[j]) == 0 {
					if shared.DotProduct(new_velocity, planes[j]) < 0 {
						break /* not ok */
					}
				}
			}

			if j == numplanes {
				break
			}
		}

		if i != numplanes {
			/* go along this plane */
			copy(ent.velocity[:], new_velocity)
		} else {
			/* go along the crease */
			if numplanes != 2 {
				copy(ent.velocity[:], []float32{0, 0, 0})
				return 7
			}

			dir := make([]float32, 3)
			shared.CrossProduct(planes[0], planes[1], dir)
			d := shared.DotProduct(dir, ent.velocity[:])
			shared.VectorScale(dir, d, ent.velocity[:])
		}

		/* if original velocity is against the original
		velocity, stop dead to avoid tiny occilations
		in sloping corners */
		if shared.DotProduct(ent.velocity[:], primal_velocity) <= 0 {
			copy(ent.velocity[:], []float32{0, 0, 0})
			return blocked
		}
	}

	return blocked
}

func (G *qGame) svAddGravity(ent *edict_t) {
	if ent == nil {
		return
	}

	ent.velocity[2] -= ent.gravity * G.sv_gravity.Float() * FRAMETIME
}

/*
 * Returns the actual bounding box of a bmodel.
 * This is a big improvement over what q2 normally
 * does with rotating bmodels - q2 sets absmin,
 * absmax to a cube that will completely contain
 * the bmodel at *any* rotation on *any* axis, whether
 * the bmodel can actually rotate to that angle or not.
 * This leads to a lot of false block tests in SV_Push
 * if another bmodel is in the vicinity.
 */
func RealBoundingBox(ent *edict_t, mins, maxs []float32) {
	//  vec3_t forward, left, up, f1, l1, u1;
	//  vec3_t p[8];
	//  int i, j, k, j2, k4;
	p := make([][]float32, 8)
	for i := 0; i < 8; i++ {
		p[i] = make([]float32, 3)
	}

	for k := 0; k < 2; k++ {
		k4 := k * 4

		if k != 0 {
			p[k4][2] = ent.maxs[2]
		} else {
			p[k4][2] = ent.mins[2]
		}

		p[k4+1][2] = p[k4][2]
		p[k4+2][2] = p[k4][2]
		p[k4+3][2] = p[k4][2]

		for j := 0; j < 2; j++ {
			j2 := j * 2

			if j != 0 {
				p[j2+k4][1] = ent.maxs[1]
			} else {
				p[j2+k4][1] = ent.mins[1]
			}

			p[j2+k4+1][1] = p[j2+k4][1]

			for i := 0; i < 2; i++ {
				if i != 0 {
					p[i+j2+k4][0] = ent.maxs[0]
				} else {
					p[i+j2+k4][0] = ent.mins[0]
				}
			}
		}
	}

	forward := make([]float32, 3)
	left := make([]float32, 3)
	up := make([]float32, 3)
	shared.AngleVectors(ent.s.Angles[:], forward, left, up)

	f1 := make([]float32, 3)
	l1 := make([]float32, 3)
	u1 := make([]float32, 3)
	for i := 0; i < 8; i++ {
		shared.VectorScale(forward, p[i][0], f1)
		shared.VectorScale(left, -p[i][1], l1)
		shared.VectorScale(up, p[i][2], u1)
		shared.VectorAdd(ent.s.Origin[:], f1, p[i])
		shared.VectorAdd(p[i], l1, p[i])
		shared.VectorAdd(p[i], u1, p[i])
	}

	copy(mins, p[0])
	copy(maxs, p[0])

	for i := 1; i < 8; i++ {
		if mins[0] > p[i][0] {
			mins[0] = p[i][0]
		}

		if mins[1] > p[i][1] {
			mins[1] = p[i][1]
		}

		if mins[2] > p[i][2] {
			mins[2] = p[i][2]
		}

		if maxs[0] < p[i][0] {
			maxs[0] = p[i][0]
		}

		if maxs[1] < p[i][1] {
			maxs[1] = p[i][1]
		}

		if maxs[2] < p[i][2] {
			maxs[2] = p[i][2]
		}
	}
}

/* ================================================================== */

/* PUSHMOVE */

/*
 * Does not change the entities velocity at all
 */

func (G *qGame) svPushEntity(ent *edict_t, push []float32) shared.Trace_t {

	start := make([]float32, 3)
	end := make([]float32, 3)
	copy(start, ent.s.Origin[:])
	shared.VectorAdd(start, push, end)

	retry := true
	var trace shared.Trace_t
	for retry {
		retry = false

		mask := shared.MASK_SOLID
		if ent.clipmask != 0 {
			mask = ent.clipmask
		}

		trace = G.gi.Trace(start, ent.mins[:], ent.maxs[:], end, ent, mask)

		if trace.Startsolid || trace.Allsolid {
			mask = mask ^ shared.CONTENTS_DEADMONSTER
			trace = G.gi.Trace(start, ent.mins[:], ent.maxs[:], end, ent, mask)
		}

		copy(ent.s.Origin[:], trace.Endpos[:])
		G.gi.Linkentity(ent)

		/* Push slightly away from non-horizontal surfaces,
		prevent origin stuck in the plane which causes
		the entity to be rendered in full black. */
		if trace.Plane.Type != 2 {
			/* Limit the fix to gibs, debris and dead monsters.
			Everything else may break existing maps. Items
			may slide to unreachable locations, monsters may
			get stuck, etc. */
			// 		 if (((strncmp(ent->classname, "monster_", 8) == 0) && ent->health < 1) ||
			// 				 (strcmp(ent->classname, "debris") == 0) || (ent->s.effects & EF_GIB))
			// 		 {
			// 			 VectorAdd(ent->s.origin, trace.plane.normal, ent->s.origin);
			// 		 }
		}

		if trace.Fraction != 1.0 {
			G.svImpact(ent, &trace)

			/* if the pushed entity went away
			and the pusher is still there */
			if !trace.Ent.(*edict_t).inuse && ent.inuse {
				/* move the pusher back and try again */
				copy(ent.s.Origin[:], start)
				G.gi.Linkentity(ent)
				retry = true
			}
		}
	}

	if ent.inuse {
		G.gTouchTriggers(ent)
	}

	return trace
}

type pushed_t struct {
	ent      *edict_t
	origin   [3]float32
	angles   [3]float32
	deltayaw float32
}

/*
 * Objects need to be moved back on a failed push,
 * otherwise riders would continue to slide.
 */
func (G *qGame) svPush(pusher *edict_t, move, amove []float32) bool {
	//  int i, e;
	//  edict_t *check, *block;
	//  pushed_t *p;
	//  vec3_t org, org2, move2, forward, right, up;
	//  vec3_t realmins, realmaxs;

	if pusher == nil {
		return false
	}

	/* clamp the move to 1/8 units, so the position will
	be accurate for client side prediction */
	for i := 0; i < 3; i++ {
		temp := move[i] * 8.0

		if temp > 0.0 {
			temp += 0.5
		} else {
			temp -= 0.5
		}

		move[i] = 0.125 * float32(int(temp))
	}

	/* we need this for pushing things later */
	org := make([]float32, 3)
	shared.VectorSubtract([]float32{0, 0, 0}, amove, org)
	forward := make([]float32, 3)
	right := make([]float32, 3)
	up := make([]float32, 3)
	shared.AngleVectors(org, forward, right, up)

	/* save the pusher's original position */
	G.pushed[G.pushed_i].ent = pusher
	copy(G.pushed[G.pushed_i].origin[:], pusher.s.Origin[:])
	copy(G.pushed[G.pushed_i].angles[:], pusher.s.Angles[:])

	if pusher.client != nil {
		G.pushed[G.pushed_i].deltayaw = float32(pusher.client.ps.Pmove.Delta_angles[shared.YAW])
	}

	G.pushed_i++

	/* move the pusher to it's final position */
	shared.VectorAdd(pusher.s.Origin[:], move, pusher.s.Origin[:])
	shared.VectorAdd(pusher.s.Angles[:], amove, pusher.s.Angles[:])
	G.gi.Linkentity(pusher)

	/* Create a real bounding box for
	rotating brush models. */
	realmins := make([]float32, 3)
	realmaxs := make([]float32, 3)
	RealBoundingBox(pusher, realmins, realmaxs)

	/* see if any solid entities
	are inside the final position */
	//  check = g_edicts + 1;

	for e := 1; e < G.num_edicts; e++ {
		check := &G.g_edicts[e]
		if !check.inuse {
			continue
		}

		if (check.movetype == MOVETYPE_PUSH) ||
			(check.movetype == MOVETYPE_STOP) ||
			(check.movetype == MOVETYPE_NONE) ||
			(check.movetype == MOVETYPE_NOCLIP) {
			continue
		}

		if check.area.Prev == nil {
			continue /* not linked in anywhere */
		}

		/* if the entity is standing on the pusher,
		it will definitely be moved */
		if check.groundentity != pusher {
			/* see if the ent needs to be tested */
			if (check.absmin[0] >= realmaxs[0]) ||
				(check.absmin[1] >= realmaxs[1]) ||
				(check.absmin[2] >= realmaxs[2]) ||
				(check.absmax[0] <= realmins[0]) ||
				(check.absmax[1] <= realmins[1]) ||
				(check.absmax[2] <= realmins[2]) {
				continue
			}

			// 		 /* see if the ent's bbox is inside
			// 			the pusher's final position */
			// 		 if (!SV_TestEntityPosition(check)) {
			// 			 continue;
			// 		 }
		}

		if (pusher.movetype == MOVETYPE_PUSH) || (check.groundentity == pusher) {
			/* move this entity */
			G.pushed[G.pushed_i].ent = check
			copy(G.pushed[G.pushed_i].origin[:], check.s.Origin[:])
			copy(G.pushed[G.pushed_i].angles[:], check.s.Angles[:])
			G.pushed_i++

			/* try moving the contacted entity */
			shared.VectorAdd(check.s.Origin[:], move, check.s.Origin[:])

			if check.client != nil {
				check.client.ps.Pmove.Delta_angles[shared.YAW] += int16(amove[shared.YAW])
			}

			/* figure movement due to the pusher's amove */
			shared.VectorSubtract(check.s.Origin[:], pusher.s.Origin[:], org)
			org2 := make([]float32, 3)
			org2[0] = shared.DotProduct(org, forward)
			org2[1] = -shared.DotProduct(org, right)
			org2[2] = shared.DotProduct(org, up)
			move2 := make([]float32, 3)
			shared.VectorSubtract(org2, org, move2)
			shared.VectorAdd(check.s.Origin[:], move2, check.s.Origin[:])

			/* may have pushed them off an edge */
			if check.groundentity != pusher {
				check.groundentity = nil
			}

			block := G.svTestEntityPosition(check)

			if block == nil {
				/* pushed ok */
				G.gi.Linkentity(check)
				continue
			}

			/* if it is ok to leave in the old position, do it
			this is only relevent for riding entities, not
			pushed */
			shared.VectorSubtract(check.s.Origin[:], move, check.s.Origin[:])
			block = G.svTestEntityPosition(check)

			if block == nil {
				G.pushed_i--
				continue
			}
		}

		/* save off the obstacle so we can
		call the block function */
		G.obstacle = check

		/* move back any entities we already moved
		go backwards, so if the same entity was pushed
		twice, it goes back to the original position */
		for p_i := G.pushed_i - 1; p_i >= 0; p_i-- {
			p := &G.pushed[p_i]
			copy(p.ent.s.Origin[:], p.origin[:])
			copy(p.ent.s.Angles[:], p.angles[:])

			if p.ent.client != nil {
				p.ent.client.ps.Pmove.Delta_angles[shared.YAW] = int16(p.deltayaw)
			}

			G.gi.Linkentity(p.ent)
		}

		return false
	}

	/* see if anything we moved has touched a trigger */
	for p_i := G.pushed_i - 1; p_i >= 0; p_i-- {
		p := &G.pushed[p_i]
		G.gTouchTriggers(p.ent)
	}

	return true
}

/*
 * Bmodel objects don't interact with each
 * other, but push all box objects
 */
func (G *qGame) svPhysics_Pusher(ent *edict_t) {

	if ent == nil {
		return
	}

	/* if not a team captain, so movement
	will be handled elsewhere */
	if (ent.flags & FL_TEAMSLAVE) != 0 {
		return
	}

	/* make sure all team slaves can move before commiting
	any moves or calling any think functions if the move
	is blocked, all moved objects will be backed out */
	G.pushed_i = 0

	var part *edict_t
	for part = ent; part != nil; part = part.teamchain {
		if part.velocity[0] != 0 || part.velocity[1] != 0 || part.velocity[2] != 0 ||
			part.avelocity[0] != 0 || part.avelocity[1] != 0 || part.avelocity[2] != 0 {
			/* object is moving */
			move := make([]float32, 3)
			amove := make([]float32, 3)
			shared.VectorScale(part.velocity[:], FRAMETIME, move)
			shared.VectorScale(part.avelocity[:], FRAMETIME, amove)

			if !G.svPush(part, move, amove) {
				break /* move was blocked */
			}
		}
	}

	//  if (pushed_p > &pushed[MAX_EDICTS -1 ])
	//  {
	// 	 gi.error("pushed_p > &pushed[MAX_EDICTS - 1], memory corrupted");
	//  }

	if part != nil {
		/* the move failed, bump all nextthink
		times and back out moves */
		for mv := ent; mv != nil; mv = mv.teamchain {
			if mv.nextthink > 0 {
				mv.nextthink += FRAMETIME
			}
		}

		/* if the pusher has a "blocked" function, call it
		otherwise, just stay in place until the obstacle
		is gone */
		// 	 if (part.blocked) {
		// 		 part->blocked(part, obstacle);
		// 	 }
	} else {
		/* the move succeeded, so call all think functions */
		for part := ent; part != nil; part = part.teamchain {
			G.svRunThink(part)
		}
	}
}

/* ================================================================== */

/*
 * Non moving objects can only think
 */
func (G *qGame) svPhysics_None(ent *edict_t) {
	if ent == nil {
		return
	}

	/* regular thinking */
	G.svRunThink(ent)
}

/* ================================================================== */

/* TOSS / BOUNCE */

/*
 * Toss, bounce, and fly movement.
 * When onground, do nothing.
 */
func (G *qGame) svPhysics_Toss(ent *edict_t) {
	//  trace_t trace;
	//  vec3_t move;
	//  float backoff;
	//  edict_t *slave;
	//  qboolean wasinwater;
	//  qboolean isinwater;
	//  vec3_t old_origin;

	if ent == nil {
		return
	}

	/* regular thinking */
	G.svRunThink(ent)

	/* if not a team captain, so movement
	will be handled elsewhere */
	if (ent.flags & FL_TEAMSLAVE) != 0 {
		return
	}

	if ent.velocity[2] > 0 {
		ent.groundentity = nil
	}

	/* check for the groundentity going away */
	if ent.groundentity != nil {
		if !ent.groundentity.inuse {
			ent.groundentity = nil
		}
	}

	/* if onground, return without moving */
	if ent.groundentity != nil {
		return
	}

	//  VectorCopy(ent->s.origin, old_origin);

	G.svCheckVelocity(ent)

	/* add gravity */
	if (ent.movetype != MOVETYPE_FLY) &&
		(ent.movetype != MOVETYPE_FLYMISSILE) {
		G.svAddGravity(ent)
	}

	/* move angles */
	shared.VectorMA(ent.s.Angles[:], FRAMETIME, ent.avelocity[:], ent.s.Angles[:])

	/* move origin */
	move := make([]float32, 3)
	shared.VectorScale(ent.velocity[:], FRAMETIME, move)
	trace := G.svPushEntity(ent, move)

	if !ent.inuse {
		return
	}

	if trace.Fraction < 1 {
		var backoff float32 = 1.0
		if ent.movetype == MOVETYPE_BOUNCE {
			backoff = 1.5
		}

		G.clipVelocity(ent.velocity[:], trace.Plane.Normal[:], ent.velocity[:], backoff)

		/* stop if on ground */
		if trace.Plane.Normal[2] > 0.7 {
			if (ent.velocity[2] < 60) || (ent.movetype != MOVETYPE_BOUNCE) {
				ent.groundentity = trace.Ent.(*edict_t)
				ent.groundentity_linkcount = ent.groundentity.linkcount
				copy(ent.velocity[:], []float32{0, 0, 0})
				copy(ent.avelocity[:], []float32{0, 0, 0})
			}
		}
	}

	/* check for water transition */
	wasinwater := (ent.watertype & shared.MASK_WATER) != 0
	ent.watertype = G.gi.Pointcontents(ent.s.Origin[:])
	isinwater := (ent.watertype & shared.MASK_WATER) != 0

	if isinwater {
		ent.waterlevel = 1
	} else {
		ent.waterlevel = 0
	}

	if !wasinwater && isinwater {
		// 	 gi.positioned_sound(old_origin, g_edicts, CHAN_AUTO,
		// 			 gi.soundindex("misc/h2ohit1.wav"), 1, 1, 0);
	} else if wasinwater && !isinwater {
		// 	 gi.positioned_sound(ent->s.origin, g_edicts, CHAN_AUTO,
		// 			 gi.soundindex("misc/h2ohit1.wav"), 1, 1, 0);
	}

	/* move teamslaves */
	for slave := ent.teamchain; slave != nil; slave = slave.teamchain {
		copy(slave.s.Origin[:], ent.s.Origin[:])
		G.gi.Linkentity(slave)
	}
}

func (G *qGame) svPhysics_Step(ent *edict_t) {
	// 	qboolean wasonground;
	// 	qboolean hitsound = false;
	// 	float *vel;
	// 	float speed, newspeed, control;
	// 	float friction;
	// 	edict_t *groundentity;
	// 	int mask;
	// 	vec3_t oldorig;
	// 	trace_t tr;

	if ent == nil {
		return
	}

	/* airborn monsters should always check for ground */
	if ent.groundentity == nil {
		G.mCheckGround(ent)
	}

	groundentity := ent.groundentity

	G.svCheckVelocity(ent)

	wasonground := groundentity != nil

	if ent.avelocity[0] != 0 || ent.avelocity[1] != 0 || ent.avelocity[2] != 0 {
		// 		SV_AddRotationalFriction(ent);
	}

	/* add gravity except:
	   flying monsters
	   swimming monsters who are in the water */
	if !wasonground {
		if (ent.flags & FL_FLY) == 0 {
			if !((ent.flags&FL_SWIM) != 0 && (ent.waterlevel > 2)) {
				// if ent.velocity[2] < G.sv_gravity.Float()*-0.1 {
				// 	hitsound = true
				// }

				if ent.waterlevel == 0 {
					G.svAddGravity(ent)
				}
			}
		}
	}

	/* friction for flying monsters that have been given vertical velocity */
	if (ent.flags&FL_FLY) != 0 && (ent.velocity[2] != 0) {
		println("FLYING")
		// 		speed = fabs(ent->velocity[2]);
		// 		control = speed < STOPSPEED ? STOPSPEED : speed;
		// 		friction = FRICTION / 3;
		// 		newspeed = speed - (FRAMETIME * control * friction);

		// 		if (newspeed < 0) {
		// 			newspeed = 0;
		// 		}

		// 		newspeed /= speed;
		// 		ent->velocity[2] *= newspeed;
	}

	/* friction for flying monsters that have been given vertical velocity */
	if (ent.flags&FL_SWIM) != 0 && (ent.velocity[2] != 0) {
		println("SWIMMING")
		// 		speed = fabs(ent->velocity[2]);
		// 		control = speed < STOPSPEED ? STOPSPEED : speed;
		// 		newspeed = speed - (FRAMETIME * control * WATERFRICTION * ent->waterlevel);

		// 		if (newspeed < 0)
		// 		{
		// 			newspeed = 0;
		// 		}

		// 		newspeed /= speed;
		// 		ent->velocity[2] *= newspeed;
	}

	if ent.velocity[2] != 0 || ent.velocity[1] != 0 || ent.velocity[0] != 0 {
		/* apply friction: let dead monsters who
		   aren't completely onground slide */
		if (wasonground) || (ent.flags&(FL_SWIM|FL_FLY)) != 0 {
			println("WASONGROUND")
			// 			if (!((ent->health <= 0.0) && !M_CheckBottom(ent)))
			// 			{
			// 				vel = ent->velocity;
			// 				speed = sqrt(vel[0] * vel[0] + vel[1] * vel[1]);

			// 				if (speed != 0) {
			// 					friction = FRICTION;

			// 					control = speed < STOPSPEED ? STOPSPEED : speed;
			// 					newspeed = speed - FRAMETIME * control * friction;

			// 					if (newspeed < 0) {
			// 						newspeed = 0;
			// 					}

			// 					newspeed /= speed;

			// 					vel[0] *= newspeed;
			// 					vel[1] *= newspeed;
			// 				}
			// 			}
		}

		mask := shared.MASK_SOLID
		if (ent.svflags & shared.SVF_MONSTER) != 0 {
			mask = shared.MASK_MONSTERSOLID
		}

		// 		VectorCopy(ent->s.origin, oldorig);
		G.svFlyMove(ent, FRAMETIME, mask)

		/* Evil hack to work around dead parasites (and maybe other monster)
		   falling through the worldmodel into the void. We copy the current
		   origin (see above) and after the SV_FlyMove() was performend we
		   checl if we're stuck in the world model. If yes we're undoing the
		   move. */
		// 		if (!VectorCompare(ent->s.origin, oldorig)) {
		// 			tr = gi.trace(ent->s.origin, ent->mins, ent->maxs, ent->s.origin, ent, mask);

		// 			if (tr.startsolid) {
		// 				VectorCopy(oldorig, ent->s.origin);
		// 			}
		// 		}

		G.gi.Linkentity(ent)
		G.gTouchTriggers(ent)

		if !ent.inuse {
			return
		}

		// 		if (ent.groundentity != nil) {
		// 			if (!wasonground) {
		// 				if (hitsound) {
		// 					gi.sound(ent, 0, gi.soundindex("world/land.wav"), 1, 1, 0);
		// 				}
		// 			}
		// 		}
	}

	/* regular thinking */
	G.svRunThink(ent)
}

/* ================================================================== */

func (G *qGame) runEntity(ent *edict_t) error {
	if ent == nil {
		return nil
	}

	if ent.prethink != nil {
		ent.prethink(ent, G)
	}

	switch ent.movetype {
	case MOVETYPE_PUSH,
		MOVETYPE_STOP:
		G.svPhysics_Pusher(ent)
	case MOVETYPE_NONE:
		G.svPhysics_None(ent)
		// 	case MOVETYPE_NOCLIP:
		// 		SV_Physics_Noclip(ent);
		// 		break;
	case MOVETYPE_STEP:
		G.svPhysics_Step(ent)
	case MOVETYPE_TOSS,
		MOVETYPE_BOUNCE,
		MOVETYPE_FLY,
		MOVETYPE_FLYMISSILE:
		G.svPhysics_Toss(ent)
	default:
		return G.gi.Error("SV_Physics: bad movetype %v", ent.movetype)
	}
	return nil
}
