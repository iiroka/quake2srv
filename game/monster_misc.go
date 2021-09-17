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
 * Monster movement support functions.
 *
 * =======================================================================
 */
package game

import (
	"math"
	"quake2srv/shared"
)

const STEPSIZE = 18
const DI_NODIR = -1

/*
 * Called by monster program code.
 * The move will be adjusted for slopes
 * and stairs, but if the move isn't
 * possible, no move is done, false is
 * returned, and pr_global_struct->trace_normal
 * is set to the normal of the blocking wall
 */
func (G *qGame) svMovestep(ent *edict_t, move []float32, relink bool) bool {
	//  float dz;
	//  vec3_t oldorg, neworg, end;
	//  trace_t trace;
	//  int i;
	//  float stepsize;
	//  vec3_t test;
	//  int contents;

	if ent == nil {
		return false
	}

	/* try the move */
	oldorg := make([]float32, 3)
	copy(oldorg, ent.s.Origin[:])
	neworg := make([]float32, 3)
	shared.VectorAdd(ent.s.Origin[:], move, neworg)

	/* flying monsters don't step up */
	if (ent.flags & (FL_SWIM | FL_FLY)) != 0 {
		/* try one move with vertical motion, then one without */
		for i := 0; i < 2; i++ {
			shared.VectorAdd(ent.s.Origin[:], move, neworg)

			if (i == 0) && ent.enemy != nil {
				if ent.goalentity == nil {
					ent.goalentity = ent.enemy
				}

				// 		 dz = ent->s.origin[2] - ent->goalentity->s.origin[2];

				// 		 if (ent->goalentity->client) {
				// 			 if (dz > 40) {
				// 				 neworg[2] -= 8;
				// 			 }

				// 			 if (!((ent->flags & FL_SWIM) && (ent->waterlevel < 2))) {
				// 				 if (dz < 30) {
				// 					 neworg[2] += 8;
				// 				 }
				// 			 }
				// 		 } else {
				// 			 if (dz > 8) {
				// 				 neworg[2] -= 8;
				// 			 } else if (dz > 0) {
				// 				 neworg[2] -= dz;
				// 			 } else if (dz < -8) {
				// 				 neworg[2] += 8;
				// 			 } else {
				// 				 neworg[2] += dz;
				// 			 }
				// 		 }
			}

			// 	 trace = gi.trace(ent->s.origin, ent->mins, ent->maxs,
			// 			 neworg, ent, MASK_MONSTERSOLID);

			// 	 /* fly monsters don't enter water voluntarily */
			// 	 if (ent->flags & FL_FLY) != 0 {
			// 		 if (!ent->waterlevel) {
			// 			 test[0] = trace.endpos[0];
			// 			 test[1] = trace.endpos[1];
			// 			 test[2] = trace.endpos[2] + ent->mins[2] + 1;
			// 			 contents = gi.pointcontents(test);

			// 			 if (contents & MASK_WATER) != 0 {
			// 				 return false;
			// 			 }
			// 		 }
			// 	 }

			// 	 /* swim monsters don't exit water voluntarily */
			// 	 if (ent->flags & FL_SWIM) != 0 {
			// 		 if (ent->waterlevel < 2) {
			// 			 test[0] = trace.endpos[0];
			// 			 test[1] = trace.endpos[1];
			// 			 test[2] = trace.endpos[2] + ent->mins[2] + 1;
			// 			 contents = gi.pointcontents(test);

			// 			 if (!(contents & MASK_WATER)) {
			// 				 return false;
			// 			 }
			// 		 }
			// 	 }

			// 	 if (trace.fraction == 1) {
			// 		 VectorCopy(trace.endpos, ent->s.origin);

			// 		 if (relink) {
			// 			 gi.linkentity(ent);
			// 			 G_TouchTriggers(ent);
			// 		 }

			// 		 return true;
			// 	 }

			if ent.enemy == nil {
				break
			}
		}

		return false
	}

	/* push down from a step height above the wished position */
	var stepsize float32
	if (ent.monsterinfo.aiflags & AI_NOSTEP) == 0 {
		stepsize = STEPSIZE
	} else {
		stepsize = 1
	}

	neworg[2] += stepsize
	end := make([]float32, 3)
	copy(end, neworg)
	end[2] -= stepsize * 2

	trace := G.gi.Trace(neworg, ent.mins[:], ent.maxs[:], end, ent, shared.MASK_MONSTERSOLID)

	if trace.Allsolid {
		return false
	}

	if trace.Startsolid {
		neworg[2] -= stepsize
		trace = G.gi.Trace(neworg, ent.mins[:], ent.maxs[:], end, ent, shared.MASK_MONSTERSOLID)

		if trace.Allsolid || trace.Startsolid {
			return false
		}
	}

	/* don't go in to water */
	if ent.waterlevel == 0 {
		test := []float32{trace.Endpos[0], trace.Endpos[1], trace.Endpos[2] + ent.mins[2] + 1}
		contents := G.gi.Pointcontents(test)

		if (contents & shared.MASK_WATER) != 0 {
			return false
		}
	}

	if trace.Fraction == 1 {
		/* if monster had the ground pulled out, go ahead and fall */
		if (ent.flags & FL_PARTIALGROUND) != 0 {
			shared.VectorAdd(ent.s.Origin[:], move, ent.s.Origin[:])

			if relink {
				G.gi.Linkentity(ent)
				G.gTouchTriggers(ent)
			}

			ent.groundentity = nil
			return true
		}

		return false /* walked off an edge */
	}

	/* check point traces down for dangling corners */
	copy(ent.s.Origin[:], trace.Endpos[:])

	// if !M_CheckBottom(ent) {
	// 	if (ent.flags & FL_PARTIALGROUND) != 0 {
	// 		/* entity had floor mostly pulled out
	// 		from underneath it and is trying to
	// 		correct */
	// 		if relink {
	// 			gi.linkentity(ent)
	// 			G_TouchTriggers(ent)
	// 		}

	// 		return true
	// 	}

	// 	VectorCopy(oldorg, ent.s.origin)
	// 	return false
	// }

	if (ent.flags & FL_PARTIALGROUND) != 0 {
		ent.flags &^= FL_PARTIALGROUND
	}

	ent.groundentity = trace.Ent.(*edict_t)
	ent.groundentity_linkcount = trace.Ent.(*edict_t).linkcount

	/* the move is ok */
	if relink {
		G.gi.Linkentity(ent)
		G.gTouchTriggers(ent)
	}

	return true
}

/* ============================================================================ */

func M_ChangeYaw(ent *edict_t) {
	// float ideal;
	// float current;
	// float move;
	// float speed;

	if ent == nil {
		return
	}

	current := shared.Anglemod(ent.s.Angles[shared.YAW])
	ideal := ent.ideal_yaw

	if current == ideal {
		return
	}

	move := ideal - current
	speed := ent.yaw_speed

	if ideal > current {
		if move >= 180 {
			move = move - 360
		}
	} else {
		if move <= -180 {
			move = move + 360
		}
	}

	if move > 0 {
		if move > speed {
			move = speed
		}
	} else {
		if move < -speed {
			move = -speed
		}
	}

	ent.s.Angles[shared.YAW] = shared.Anglemod(current + move)
}

/*
 * Turns to the movement direction, and
 * walks the current distance if facing it.
 */
func (G *qGame) svStepDirection(ent *edict_t, yaw, dist float32) bool {
	//  vec3_t move, oldorigin;
	//  float delta;

	if ent == nil {
		return false
	}

	ent.ideal_yaw = yaw
	M_ChangeYaw(ent)

	yaw = yaw * math.Pi * 2 / 360
	move := []float32{float32(math.Cos(float64(yaw))) * dist, float32(math.Sin(float64(yaw))) * dist, 0}

	oldorigin := make([]float32, 3)
	copy(oldorigin, ent.s.Origin[:])

	if G.svMovestep(ent, move, false) {
		delta := ent.s.Angles[shared.YAW] - ent.ideal_yaw

		if (delta > 45) && (delta < 315) {
			/* not turned far enough, so don't take the step */
			copy(ent.s.Origin[:], oldorigin)
		}

		G.gi.Linkentity(ent)
		G.gTouchTriggers(ent)
		return true
	}

	G.gi.Linkentity(ent)
	G.gTouchTriggers(ent)
	return false
}

func (G *qGame) svNewChaseDir(actor, enemy *edict_t, dist float32) {

	if actor == nil || enemy == nil {
		return
	}

	olddir := shared.Anglemod(float32(int(actor.ideal_yaw/45) * 45))
	turnaround := shared.Anglemod(olddir - 180)

	deltax := enemy.s.Origin[0] - actor.s.Origin[0]
	deltay := enemy.s.Origin[1] - actor.s.Origin[1]

	d := make([]float32, 3)
	if deltax > 10 {
		d[1] = 0
	} else if deltax < -10 {
		d[1] = 180
	} else {
		d[1] = DI_NODIR
	}

	if deltay < -10 {
		d[2] = 270
	} else if deltay > 10 {
		d[2] = 90
	} else {
		d[2] = DI_NODIR
	}

	/* try direct route */
	if (d[1] != DI_NODIR) && (d[2] != DI_NODIR) {
		var tdir float32
		if d[1] == 0 {
			if d[2] == 90 {
				tdir = 45
			} else {
				tdir = 315
			}
		} else {
			if d[2] == 90 {
				tdir = 135
			} else {
				tdir = 215
			}
		}

		if (tdir != turnaround) && G.svStepDirection(actor, tdir, dist) {
			return
		}
	}

	/* try other directions */
	if ((shared.Randk()&3)&1) != 0 || (math.Abs(float64(deltay)) > math.Abs(float64(deltax))) {
		tdir := d[1]
		d[1] = d[2]
		d[2] = tdir
	}

	if (d[1] != DI_NODIR) && (d[1] != turnaround) &&
		G.svStepDirection(actor, d[1], dist) {
		return
	}

	if (d[2] != DI_NODIR) && (d[2] != turnaround) &&
		G.svStepDirection(actor, d[2], dist) {
		return
	}

	/* there is no direct path to the player, so pick another direction */
	if (olddir != DI_NODIR) && G.svStepDirection(actor, olddir, dist) {
		return
	}

	if (shared.Randk() & 1) != 0 { /* randomly determine direction of search */
		for tdir := 0; tdir <= 315; tdir += 45 {
			if (float32(tdir) != turnaround) && G.svStepDirection(actor, float32(tdir), dist) {
				return
			}
		}
	} else {
		for tdir := 315; tdir >= 0; tdir -= 45 {
			if (float32(tdir) != turnaround) && G.svStepDirection(actor, float32(tdir), dist) {
				return
			}
		}
	}

	if (turnaround != DI_NODIR) && G.svStepDirection(actor, turnaround, dist) {
		return
	}

	actor.ideal_yaw = olddir /* can't move */

	/* if a bridge was pulled out from underneath
	   a monster, it may not have a valid standing
	   position at all */
	// if (!M_CheckBottom(actor)) {
	// 	SV_FixCheckBottom(actor);
	// }
}

func SV_CloseEnough(ent, goal *edict_t, dist float32) bool {

	if ent == nil || goal == nil {
		return false
	}

	for i := 0; i < 3; i++ {
		if goal.absmin[i] > ent.absmax[i]+dist {
			return false
		}

		if goal.absmax[i] < ent.absmin[i]-dist {
			return false
		}
	}

	return true
}

func (G *qGame) mMoveToGoal(ent *edict_t, dist float32) {

	if ent == nil {
		return
	}

	goal := ent.goalentity

	if ent.groundentity == nil && (ent.flags&(FL_FLY|FL_SWIM)) == 0 {
		return
	}

	/* if the next step hits the enemy, return immediately */
	if ent.enemy != nil && SV_CloseEnough(ent, ent.enemy, dist) {
		return
	}

	/* bump around... */
	if ((shared.Randk() & 3) == 1) || !G.svStepDirection(ent, ent.ideal_yaw, dist) {
		if ent.inuse {
			G.svNewChaseDir(ent, goal, dist)
		}
	}
}

func (G *qGame) mWalkmove(ent *edict_t, yaw, dist float32) bool {

	if ent == nil {
		return false
	}

	if ent.groundentity == nil && (ent.flags&(FL_FLY|FL_SWIM) == 0) {
		return false
	}

	dyaw := float64(yaw) * math.Pi * 2 / 360

	move := []float32{
		float32(math.Cos(dyaw)) * dist,
		float32(math.Sin(dyaw)) * dist,
		0}

	return G.svMovestep(ent, move, true)
}
