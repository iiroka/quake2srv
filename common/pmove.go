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
 * Player movement code. This is the core of Quake IIs legendary physics
 * engine
 *
 * =======================================================================
 */
package common

import (
	"fmt"
	"math"
	"quake2srv/shared"
)

const STEPSIZE = 18

/* all of the locals will be zeroed before each
 * pmove, just to make damn sure we don't have
 * any differences when running on client or server */

type pml_t struct {
	origin   [3]float32 /* full float precision */
	velocity [3]float32 /* full float precision */

	forward, right, up [3]float32
	frametime          float32

	groundsurface  *shared.Csurface_t
	groundplane    shared.Cplane_t
	groundcontents int

	previous_origin [3]float32
	ladder          bool
}

const STOP_EPSILON = 0.1    /* Slide off of the impacting object returns the blocked flags (1 = floor, 2 = step / wall) */
const MIN_STEP_NORMAL = 0.7 /* can't step up onto very steep slopes */
const MAX_CLIP_PLANES = 5

func PM_ClipVelocity(in, normal, out []float32, overbounce float32) {
	// float backoff;
	// float change;
	// int i;

	backoff := shared.DotProduct(in, normal) * overbounce

	for i := 0; i < 3; i++ {
		change := normal[i] * backoff
		out[i] = in[i] - change

		if (out[i] > -STOP_EPSILON) && (out[i] < STOP_EPSILON) {
			out[i] = 0
		}
	}
}

/*
 * Each intersection will try to step over the obstruction instead of
 * sliding along it.
 *
 * Returns a new origin, velocity, and contact entity
 * Does not modify any world state?
 */
func PM_StepSlideMove_(pm *shared.Pmove_t, pml *pml_t) {

	numbumps := 4

	primal_velocity := make([]float32, 3)
	copy(primal_velocity, pml.velocity[:])
	numplanes := 0

	time_left := pml.frametime

	var planes [MAX_CLIP_PLANES][3]float32

	for bumpcount := 0; bumpcount < numbumps; bumpcount++ {
		end := make([]float32, 3)
		for i := 0; i < 3; i++ {
			end[i] = pml.origin[i] + time_left*pml.velocity[i]
		}

		trace := pm.Trace(pml.origin[:], pm.Mins[:], pm.Maxs[:], end, pm.TraceArg)

		if trace.Allsolid {
			/* entity is trapped in another solid */
			pml.velocity[2] = 0 /* don't build up falling damage */
			return
		}

		if trace.Fraction > 0 {
			/* actually covered some distance */
			copy(pml.origin[:], trace.Endpos[:])
			numplanes = 0
		}

		if trace.Fraction == 1 {
			break /* moved the entire distance */
		}

		/* save entity for contact */
		if (pm.Numtouch < shared.MAXTOUCH) && trace.Ent != nil {
			pm.Touchents[pm.Numtouch] = trace.Ent
			pm.Numtouch++
		}

		time_left -= time_left * trace.Fraction

		/* slide along this plane */
		if numplanes >= MAX_CLIP_PLANES {
			/* this shouldn't really happen */
			copy(pml.velocity[:], []float32{0, 0, 0})
			break
		}

		copy(planes[numplanes][:], trace.Plane.Normal[:])
		numplanes++

		/* modify original_velocity so it parallels all of the clip planes */
		found := false
		for i := 0; i < numplanes; i++ {
			PM_ClipVelocity(pml.velocity[:], planes[i][:], pml.velocity[:], 1.01)

			for j := 0; j < numplanes; j++ {
				if j != i {
					if shared.DotProduct(pml.velocity[:], planes[j][:]) < 0 {
						found = true
						break /* not ok */
					}
				}
			}

			if !found {
				break
			}
		}

		if found {
			/* go along this plane */
		} else {
			/* go along the crease */
			if numplanes != 2 {
				copy(pml.velocity[:], []float32{0, 0, 0})
				break
			}

			dir := make([]float32, 3)
			shared.CrossProduct(planes[0][:], planes[1][:], dir)
			d := shared.DotProduct(dir, pml.velocity[:])
			shared.VectorScale(dir, d, pml.velocity[:])
		}

		/* if velocity is against the original velocity, stop dead
		to avoid tiny occilations in sloping corners */
		if shared.DotProduct(pml.velocity[:], primal_velocity) <= 0 {
			copy(pml.velocity[:], []float32{0, 0, 0})
			break
		}
	}

	if pm.S.Pm_time != 0 {
		copy(pml.velocity[:], primal_velocity)
	}
}

func PM_StepSlideMove(pm *shared.Pmove_t, pml *pml_t) {

	start_o := make([]float32, 3)
	copy(start_o, pml.origin[:])
	start_v := make([]float32, 3)
	copy(start_v, pml.velocity[:])

	PM_StepSlideMove_(pm, pml)

	down_o := make([]float32, 3)
	copy(down_o, pml.origin[:])
	down_v := make([]float32, 3)
	copy(down_v, pml.velocity[:])

	up := make([]float32, 3)
	copy(up, start_o)
	up[2] += STEPSIZE

	trace := pm.Trace(up, pm.Mins[:], pm.Maxs[:], up, pm.TraceArg)

	if trace.Allsolid {
		return /* can't step up */
	}

	/* try sliding above */
	copy(pml.origin[:], up)
	copy(pml.velocity[:], start_v)

	PM_StepSlideMove_(pm, pml)

	/* push down the final amount */
	down := make([]float32, 3)
	copy(down, pml.origin[:])
	down[2] -= STEPSIZE
	trace = pm.Trace(pml.origin[:], pm.Mins[:], pm.Maxs[:], down, pm.TraceArg)

	if !trace.Allsolid {
		copy(pml.origin[:], trace.Endpos[:])
	}

	copy(up, pml.origin[:])

	/* decide which one went farther */
	down_dist := (down_o[0]-start_o[0])*(down_o[0]-start_o[0]) +
		(down_o[1]-start_o[1])*(down_o[1]-start_o[1])
	up_dist := (up[0]-start_o[0])*(up[0]-start_o[0]) +
		(up[1]-start_o[1])*(up[1]-start_o[1])

	if (down_dist > up_dist) || (trace.Plane.Normal[2] < MIN_STEP_NORMAL) {
		copy(pml.origin[:], down_o)
		copy(pml.velocity[:], down_v)
		return
	}

	pml.velocity[2] = down_v[2]
}

/*
 * Handles both ground friction and water friction
 */
func (T *qCommon) pmFriction(pm *shared.Pmove_t, pml *pml_t) {

	vel := pml.velocity[:]

	speed := float32(math.Sqrt(float64(vel[0]*vel[0]) + float64(vel[1]*vel[1]) + float64(vel[2]*vel[2])))

	if speed < 1 {
		vel[0] = 0
		vel[1] = 0
		return
	}

	var drop float32 = 0

	/* apply ground friction */
	if (pm.Groundentity != nil && pml.groundsurface != nil &&
		(pml.groundsurface.Flags&shared.SURF_SLICK) == 0) || (pml.ladder) {
		friction := T.pm_friction
		var control float32
		if speed < T.pm_stopspeed {
			control = T.pm_stopspeed
		} else {
			control = speed
		}
		drop += control * friction * pml.frametime
	}

	/* apply water friction */
	if pm.Waterlevel != 0 && !pml.ladder {
		drop += speed * T.pm_waterfriction * float32(pm.Waterlevel) * pml.frametime
	}

	/* scale the velocity */
	newspeed := speed - drop

	if newspeed < 0 {
		newspeed = 0
	}

	newspeed /= speed

	vel[0] = vel[0] * newspeed
	vel[1] = vel[1] * newspeed
	vel[2] = vel[2] * newspeed
}

/*
 * Handles user intended acceleration
 */
func PM_Accelerate(wishdir []float32, wishspeed, accel float32, pml *pml_t) {

	currentspeed := shared.DotProduct(pml.velocity[:], wishdir)
	addspeed := wishspeed - currentspeed
	if addspeed <= 0 {
		return
	}

	accelspeed := accel * pml.frametime * wishspeed

	if accelspeed > addspeed {
		accelspeed = addspeed
	}

	for i := 0; i < 3; i++ {
		pml.velocity[i] += accelspeed * wishdir[i]
	}
}

func PM_AddCurrents(pm *shared.Pmove_t, pml *pml_t, wishvel []float32) {
	// vec3_t v;
	// float s;

	/* account for ladders */
	if pml.ladder && (math.Abs(float64(pml.velocity[2])) <= 200) {
		if (pm.Viewangles[shared.PITCH] <= -15) && (pm.Cmd.Forwardmove > 0) {
			wishvel[2] = 200
		} else if (pm.Viewangles[shared.PITCH] >= 15) && (pm.Cmd.Forwardmove > 0) {
			wishvel[2] = -200
		} else if pm.Cmd.Upmove > 0 {
			wishvel[2] = 200
		} else if pm.Cmd.Upmove < 0 {
			wishvel[2] = -200
		} else {
			wishvel[2] = 0
		}

		/* limit horizontal speed when on a ladder */
		if wishvel[0] < -25 {
			wishvel[0] = -25
		} else if wishvel[0] > 25 {
			wishvel[0] = 25
		}

		if wishvel[1] < -25 {
			wishvel[1] = -25
		} else if wishvel[1] > 25 {
			wishvel[1] = 25
		}
	}

	// /* add water currents  */
	// if (pm.watertype & MASK_CURRENT) != 0 {
	// 	VectorClear(v);

	// 	if (pm.watertype & CONTENTS_CURRENT_0) != 0 {
	// 		v[0] += 1;
	// 	}

	// 	if (pm.watertype & CONTENTS_CURRENT_90) != 0 {
	// 		v[1] += 1;
	// 	}

	// 	if (pm.watertype & CONTENTS_CURRENT_180) != 0 {
	// 		v[0] -= 1;
	// 	}

	// 	if (pm.watertype & CONTENTS_CURRENT_270) != 0 {
	// 		v[1] -= 1;
	// 	}

	// 	if (pm.watertype & CONTENTS_CURRENT_UP) != 0 {
	// 		v[2] += 1;
	// 	}

	// 	if (pm.watertype & CONTENTS_CURRENT_DOWN) != 0 {
	// 		v[2] -= 1;
	// 	}

	// 	s = pm_waterspeed;

	// 	if ((pm.waterlevel == 1) && (pm.groundentity)) {
	// 		s /= 2;
	// 	}

	// 	VectorMA(wishvel, s, v, wishvel);
	// }

	// /* add conveyor belt velocities */
	// if (pm.groundentity != nil) {
	// 	VectorClear(v);

	// 	if (pml.groundcontents & CONTENTS_CURRENT_0) != 0 {
	// 		v[0] += 1;
	// 	}

	// 	if (pml.groundcontents & CONTENTS_CURRENT_90) != 0 {
	// 		v[1] += 1;
	// 	}

	// 	if (pml.groundcontents & CONTENTS_CURRENT_180) != 0 {
	// 		v[0] -= 1;
	// 	}

	// 	if (pml.groundcontents & CONTENTS_CURRENT_270) != 0 {
	// 		v[1] -= 1;
	// 	}

	// 	if (pml.groundcontents & CONTENTS_CURRENT_UP) != 0 {
	// 		v[2] += 1;
	// 	}

	// 	if (pml.groundcontents & CONTENTS_CURRENT_DOWN) != 0 {
	// 		v[2] -= 1;
	// 	}

	// 	VectorMA(wishvel, 100, v, wishvel);
	// }
}

func (T *qCommon) pmWaterMove(pm *shared.Pmove_t, pml *pml_t) {
	// int i;
	// vec3_t wishvel;
	// float wishspeed;
	// vec3_t wishdir;

	/* user intentions */
	wishvel := make([]float32, 3)
	for i := 0; i < 3; i++ {
		wishvel[i] = pml.forward[i]*float32(pm.Cmd.Forwardmove) +
			pml.right[i]*float32(pm.Cmd.Sidemove)
	}

	if pm.Cmd.Forwardmove == 0 && pm.Cmd.Sidemove == 0 && pm.Cmd.Upmove == 0 {
		wishvel[2] -= 60 /* drift towards bottom */
	} else {
		wishvel[2] += float32(pm.Cmd.Upmove)
	}

	PM_AddCurrents(pm, pml, wishvel)

	wishdir := make([]float32, 3)
	copy(wishdir, wishvel)
	wishspeed := shared.VectorNormalize(wishdir)

	if wishspeed > T.pm_maxspeed {
		shared.VectorScale(wishvel, T.pm_maxspeed/wishspeed, wishvel)
		wishspeed = T.pm_maxspeed
	}

	wishspeed *= 0.5

	PM_Accelerate(wishdir, wishspeed, T.pm_wateraccelerate, pml)

	PM_StepSlideMove(pm, pml)
}

func (T *qCommon) pmAirMove(pm *shared.Pmove_t, pml *pml_t) {

	fmove := float32(pm.Cmd.Forwardmove)
	smove := float32(pm.Cmd.Sidemove)

	var wishvel [3]float32
	for i := 0; i < 2; i++ {
		wishvel[i] = pml.forward[i]*fmove + pml.right[i]*smove
	}

	wishvel[2] = 0

	PM_AddCurrents(pm, pml, wishvel[:])

	var wishdir [3]float32
	copy(wishdir[:], wishvel[:])
	wishspeed := shared.VectorNormalize(wishdir[:])

	/* clamp to server defined max speed */
	var maxspeed float32
	if (pm.S.Pm_flags & shared.PMF_DUCKED) != 0 {
		maxspeed = T.pm_duckspeed
	} else {
		maxspeed = T.pm_maxspeed
	}

	if wishspeed > maxspeed {
		shared.VectorScale(wishvel[:], maxspeed/wishspeed, wishvel[:])
		wishspeed = maxspeed
	}

	if pml.ladder {
		PM_Accelerate(wishdir[:], wishspeed, T.pm_accelerate, pml)

		if wishvel[2] == 0 {
			if pml.velocity[2] > 0 {
				pml.velocity[2] -= float32(pm.S.Gravity) * pml.frametime

				if pml.velocity[2] < 0 {
					pml.velocity[2] = 0
				}
			} else {
				pml.velocity[2] += float32(pm.S.Gravity) * pml.frametime

				if pml.velocity[2] > 0 {
					pml.velocity[2] = 0
				}
			}
		}

		PM_StepSlideMove(pm, pml)
	} else if pm.Groundentity != nil {
		/* walking on ground */
		pml.velocity[2] = 0
		PM_Accelerate(wishdir[:], wishspeed, T.pm_accelerate, pml)

		if pm.S.Gravity > 0 {
			pml.velocity[2] = 0
		} else {
			pml.velocity[2] -= float32(pm.S.Gravity) * pml.frametime
		}

		if pml.velocity[0] == 0 && pml.velocity[1] == 0 {
			return
		}

		PM_StepSlideMove(pm, pml)
	} else {
		// 	/* not on ground, so little effect on velocity */
		// 	if (pm_airaccelerate)
		// 	{
		// 		PM_AirAccelerate(wishdir, wishspeed, pm_accelerate);
		// 	}
		// 	else
		// 	{
		// 		PM_Accelerate(wishdir, wishspeed, 1);
		// 	}

		/* add gravity */
		pml.velocity[2] -= float32(pm.S.Gravity) * pml.frametime
		PM_StepSlideMove(pm, pml)
	}
}

func PM_CheckJump(pm *shared.Pmove_t, pml *pml_t) {
	if (pm.S.Pm_flags & shared.PMF_TIME_LAND) != 0 {
		/* hasn't been long enough since landing to jump again */
		return
	}

	if pm.Cmd.Upmove < 10 {
		/* not holding jump */
		pm.S.Pm_flags &^= shared.PMF_JUMP_HELD
		return
	}

	/* must wait for jump to be released */
	if (pm.S.Pm_flags & shared.PMF_JUMP_HELD) != 0 {
		return
	}

	if pm.S.Pm_type == shared.PM_DEAD {
		return
	}

	if pm.Waterlevel >= 2 {
		/* swimming, not jumping */
		pm.Groundentity = nil

		if pml.velocity[2] <= -300 {
			return
		}

		if pm.Watertype == shared.CONTENTS_WATER {
			pml.velocity[2] = 100
		} else if pm.Watertype == shared.CONTENTS_SLIME {
			pml.velocity[2] = 80
		} else {
			pml.velocity[2] = 50
		}

		return
	}

	if pm.Groundentity == nil {
		return /* in air, so no effect */
	}

	pm.S.Pm_flags |= shared.PMF_JUMP_HELD

	pm.Groundentity = nil
	pml.velocity[2] += 270

	if pml.velocity[2] < 270 {
		pml.velocity[2] = 270
	}
}

func PM_CheckSpecialMovement(pm *shared.Pmove_t, pml *pml_t) {
	// vec3_t spot;
	// int cont;
	// vec3_t flatforward;
	// trace_t trace;

	if pm.S.Pm_time != 0 {
		return
	}

	pml.ladder = false

	/* check for ladder */
	flatforward := []float32{pml.forward[0], pml.forward[1], 0}
	shared.VectorNormalize(flatforward)

	spot := make([]float32, 3)
	shared.VectorMA(pml.origin[:], 1, flatforward, spot)
	trace := pm.Trace(pml.origin[:], pm.Mins[:], pm.Maxs[:], spot, pm.TraceArg)

	if (trace.Fraction < 1) && (trace.Contents&shared.CONTENTS_LADDER) != 0 {
		pml.ladder = true
	}

	/* check for water jump */
	if pm.Waterlevel != 2 {
		return
	}

	shared.VectorMA(pml.origin[:], 30, flatforward, spot)
	spot[2] += 4
	cont := pm.Pointcontents(spot, pm.PCArg)

	if (cont & shared.CONTENTS_SOLID) == 0 {
		return
	}

	spot[2] += 16
	cont = pm.Pointcontents(spot, pm.PCArg)

	if cont != 0 {
		return
	}

	/* jump out of water */
	shared.VectorScale(flatforward, 50, pml.velocity[:])
	pml.velocity[2] = 350

	pm.S.Pm_flags |= shared.PMF_TIME_WATERJUMP
	pm.S.Pm_time = 255
}

func PM_CatagorizePosition(pm *shared.Pmove_t, pml *pml_t) {

	/* if the player hull point one unit down
	is solid, the player is on ground */

	/* see if standing on something solid */
	point := []float32{pml.origin[0], pml.origin[1], pml.origin[2] - 0.25}

	if pml.velocity[2] > 180 {
		pm.S.Pm_flags &^= shared.PMF_ON_GROUND
		pm.Groundentity = nil
	} else {
		trace := pm.Trace(pml.origin[:], pm.Mins[:], pm.Maxs[:], point, pm.TraceArg)
		pml.groundplane = trace.Plane
		pml.groundsurface = trace.Surface
		pml.groundcontents = trace.Contents

		if trace.Ent == nil || ((trace.Plane.Normal[2] < 0.7) && !trace.Startsolid) {
			pm.Groundentity = nil
			pm.S.Pm_flags &^= shared.PMF_ON_GROUND
		} else {
			pm.Groundentity = trace.Ent

			// 		/* hitting solid ground will end a waterjump */
			// 		if (pm.S.pm_flags & PMF_TIME_WATERJUMP) != 0 {
			// 			pm.S.pm_flags &=
			// 				~(PMF_TIME_WATERJUMP | PMF_TIME_LAND | PMF_TIME_TELEPORT);
			// 			pm.S.pm_time = 0;
			// 		}

			if (pm.S.Pm_flags & shared.PMF_ON_GROUND) == 0 {
				/* just hit the ground */
				pm.S.Pm_flags |= shared.PMF_ON_GROUND

				/* don't do landing time if we were just going down a slope */
				// 			if (pml.velocity[2] < -200) {
				// 				pm->s.pm_flags |= PMF_TIME_LAND;

				// 				/* don't allow another jump for a little while */
				// 				if (pml.velocity[2] < -400) {
				// 					pm->s.pm_time = 25;
				// 				} else {
				// 					pm->s.pm_time = 18;
				// 				}
				// 			}
			}
		}

		if (pm.Numtouch < shared.MAXTOUCH) && trace.Ent != nil {
			pm.Touchents[pm.Numtouch] = trace.Ent
			pm.Numtouch++
		}
	}

	/* get waterlevel, accounting for ducking */
	pm.Waterlevel = 0
	pm.Watertype = 0

	sample2 := pm.Viewheight - pm.Mins[2]
	sample1 := sample2 / 2

	point[2] = pml.origin[2] + pm.Mins[2] + 1
	cont := pm.Pointcontents(point, pm.PCArg)

	if (cont & shared.MASK_WATER) != 0 {
		pm.Watertype = cont
		pm.Waterlevel = 1
		point[2] = pml.origin[2] + pm.Mins[2] + sample1
		cont = pm.Pointcontents(point, pm.PCArg)

		if (cont & shared.MASK_WATER) != 0 {
			pm.Waterlevel = 2
			point[2] = pml.origin[2] + pm.Mins[2] + sample2
			cont = pm.Pointcontents(point, pm.PCArg)

			if (cont & shared.MASK_WATER) != 0 {
				pm.Waterlevel = 3
			}
		}
	}
}

/*
 * Sets mins, maxs, and pm->viewheight
 */
func PM_CheckDuck(pm *shared.Pmove_t, pml *pml_t) {

	pm.Mins[0] = -16
	pm.Mins[1] = -16

	pm.Maxs[0] = 16
	pm.Maxs[1] = 16

	if pm.S.Pm_type == shared.PM_GIB {
		pm.Mins[2] = 0
		pm.Maxs[2] = 16
		pm.Viewheight = 8
		return
	}

	pm.Mins[2] = -24

	if pm.S.Pm_type == shared.PM_DEAD {
		pm.S.Pm_flags |= shared.PMF_DUCKED
	} else if (pm.Cmd.Upmove < 0) && (pm.S.Pm_flags&shared.PMF_ON_GROUND) != 0 {
		/* duck */
		pm.S.Pm_flags |= shared.PMF_DUCKED
	} else {
		/* stand up if possible */
		if (pm.S.Pm_flags & shared.PMF_DUCKED) != 0 {
			/* try to stand up */
			pm.Maxs[2] = 32
			trace := pm.Trace(pml.origin[:], pm.Mins[:], pm.Maxs[:], pml.origin[:], pm.TraceArg)

			if !trace.Allsolid {
				pm.S.Pm_flags &^= shared.PMF_DUCKED
			}
		}
	}

	if (pm.S.Pm_flags & shared.PMF_DUCKED) != 0 {
		pm.Maxs[2] = 4
		pm.Viewheight = -2
	} else {
		pm.Maxs[2] = 32
		pm.Viewheight = 22
	}
}

func PM_GoodPosition(pm *shared.Pmove_t) bool {

	if pm.S.Pm_type == shared.PM_SPECTATOR {
		return true
	}

	var origin [3]float32
	var end [3]float32
	for i := 0; i < 3; i++ {
		end[i] = float32(pm.S.Origin[i]) * 0.125
		origin[i] = end[i]
	}

	trace := pm.Trace(origin[:], pm.Mins[:], pm.Maxs[:], end[:], pm.TraceArg)

	return !trace.Allsolid
}

/*
 * On exit, the origin will have a value that is pre-quantized to the 0.125
 * precision of the network channel and in a valid position.
 */
func PM_SnapPosition(pm *shared.Pmove_t, pml *pml_t) {
	/* try all single bits first */
	jitterbits := []int{0, 4, 1, 2, 3, 5, 6, 7}

	/* snap velocity to eigths */
	for i := 0; i < 3; i++ {
		pm.S.Velocity[i] = int16(pml.velocity[i] * 8)
	}

	var sign [3]int16
	for i := 0; i < 3; i++ {
		if pml.origin[i] >= 0 {
			sign[i] = 1
		} else {
			sign[i] = -1
		}

		pm.S.Origin[i] = int16(pml.origin[i] * 8)

		if float32(pm.S.Origin[i])*0.125 == pml.origin[i] {
			sign[i] = 0
		}
	}

	var base [3]int16
	copy(base[:], pm.S.Origin[:])

	/* try all combinations */
	for j := 0; j < 8; j++ {
		bits := jitterbits[j]
		copy(pm.S.Origin[:], base[:])

		for i := 0; i < 3; i++ {
			if (bits & (1 << i)) != 0 {
				pm.S.Origin[i] += sign[i]
			}
		}

		if PM_GoodPosition(pm) {
			return
		}
	}

	/* go back to the last position */
	for i := 0; i < 3; i++ {
		pm.S.Origin[i] = int16(pml.previous_origin[i])
	}
}

func pmInitialSnapPosition(pm *shared.Pmove_t, pml *pml_t) {

	offset := []int16{0, -1, 1}

	base := make([]int16, 3)
	copy(base, pm.S.Origin[:])

	for z := 0; z < 3; z++ {
		pm.S.Origin[2] = base[2] + offset[z]

		for y := 0; y < 3; y++ {
			pm.S.Origin[1] = base[1] + offset[y]

			for x := 0; x < 3; x++ {
				pm.S.Origin[0] = base[0] + offset[x]

				if PM_GoodPosition(pm) {
					pml.origin[0] = float32(pm.S.Origin[0]) * 0.125
					pml.origin[1] = float32(pm.S.Origin[1]) * 0.125
					pml.origin[2] = float32(pm.S.Origin[2]) * 0.125
					for i := 0; i < 3; i++ {
						pml.previous_origin[i] = float32(pm.S.Origin[i])
					}
					return
				}
			}
		}
	}

	fmt.Printf("Bad InitialSnapPosition\n")
}

func PM_ClampAngles(pm *shared.Pmove_t, pml *pml_t) {

	if (pm.S.Pm_flags & shared.PMF_TIME_TELEPORT) != 0 {
		pm.Viewangles[shared.YAW] = shared.SHORT2ANGLE(
			int(pm.Cmd.Angles[shared.YAW]) + int(pm.S.Delta_angles[shared.YAW]))
		pm.Viewangles[shared.PITCH] = 0
		pm.Viewangles[shared.ROLL] = 0
	} else {
		/* circularly clamp the angles with deltas */
		for i := 0; i < 3; i++ {
			temp := int(pm.Cmd.Angles[i]) + int(pm.S.Delta_angles[i])
			pm.Viewangles[i] = shared.SHORT2ANGLE(temp)
		}

		/* don't let the player look up or down more than 90 degrees */
		if (pm.Viewangles[shared.PITCH] > 89) && (pm.Viewangles[shared.PITCH] < 180) {
			pm.Viewangles[shared.PITCH] = 89
		} else if (pm.Viewangles[shared.PITCH] < 271) && (pm.Viewangles[shared.PITCH] >= 180) {
			pm.Viewangles[shared.PITCH] = 271
		}
	}

	shared.AngleVectors(pm.Viewangles[:], pml.forward[:], pml.right[:], pml.up[:])
}

func PM_CalculateViewHeightForDemo(pm *shared.Pmove_t) {
	if pm.S.Pm_type == shared.PM_GIB {
		pm.Viewheight = 8
	} else {
		if (pm.S.Pm_flags & shared.PMF_DUCKED) != 0 {
			pm.Viewheight = -2
		} else {
			pm.Viewheight = 22
		}
	}
}

/*
 * Can be called by either the server or the client
 */
func (T *qCommon) Pmove(pm *shared.Pmove_t) {
	/* clear results */
	pm.Numtouch = 0
	pm.Viewangles[0] = 0
	pm.Viewangles[1] = 0
	pm.Viewangles[2] = 0
	pm.Viewheight = 0
	pm.Groundentity = nil
	pm.Watertype = 0
	pm.Waterlevel = 0

	/* clear all pmove local vars */
	pml := pml_t{}

	/* convert origin and velocity to float values */
	pml.origin[0] = float32(pm.S.Origin[0]) * 0.125
	pml.origin[1] = float32(pm.S.Origin[1]) * 0.125
	pml.origin[2] = float32(pm.S.Origin[2]) * 0.125

	pml.velocity[0] = float32(pm.S.Velocity[0]) * 0.125
	pml.velocity[1] = float32(pm.S.Velocity[1]) * 0.125
	pml.velocity[2] = float32(pm.S.Velocity[2]) * 0.125

	/* save old org in case we get stuck */
	for i := range pm.S.Origin {
		pml.previous_origin[i] = float32(pm.S.Origin[i])
	}

	pml.frametime = float32(pm.Cmd.Msec) * 0.001

	PM_ClampAngles(pm, &pml)

	if pm.S.Pm_type == shared.PM_SPECTATOR {
		// 		PM_FlyMove(false);
		PM_SnapPosition(pm, &pml)
		return
	}

	if pm.S.Pm_type >= shared.PM_DEAD {
		pm.Cmd.Forwardmove = 0
		pm.Cmd.Sidemove = 0
		pm.Cmd.Upmove = 0
	}

	if pm.S.Pm_type == shared.PM_FREEZE {
		// if T.client.IsAttractloop() {
		// 	PM_CalculateViewHeightForDemo(pm)
		// 			PM_CalculateWaterLevelForDemo();
		// 			PM_UpdateUnderwaterSfx();
		// }
		return /* no movement at all */
	}

	/* set mins, maxs, and viewheight */
	PM_CheckDuck(pm, &pml)

	if pm.Snapinitial {
		pmInitialSnapPosition(pm, &pml)
	}

	/* set groundentity, watertype, and waterlevel */
	PM_CatagorizePosition(pm, &pml)

	if pm.S.Pm_type == shared.PM_DEAD {
		// 		PM_DeadMove();
	}

	PM_CheckSpecialMovement(pm, &pml)

	/* drop timing counter */
	if pm.S.Pm_time != 0 {
		msec := pm.Cmd.Msec >> 3

		if msec == 0 {
			msec = 1
		}

		if msec >= pm.S.Pm_time {
			pm.S.Pm_flags &^= (shared.PMF_TIME_WATERJUMP | shared.PMF_TIME_LAND | shared.PMF_TIME_TELEPORT)
			pm.S.Pm_time = 0
		} else {
			pm.S.Pm_time -= msec
		}
	}

	if (pm.S.Pm_flags & shared.PMF_TIME_TELEPORT) != 0 {
		/* teleport pause stays exactly in place */
	} else if (pm.S.Pm_flags & shared.PMF_TIME_WATERJUMP) != 0 {
		/* waterjump has no control, but falls */
		pml.velocity[2] -= float32(pm.S.Gravity) * pml.frametime

		if pml.velocity[2] < 0 {
			/* cancel as soon as we are falling down again */
			pm.S.Pm_flags &^= (shared.PMF_TIME_WATERJUMP | shared.PMF_TIME_LAND | shared.PMF_TIME_TELEPORT)
			pm.S.Pm_time = 0
		}

		PM_StepSlideMove(pm, &pml)
	} else {
		PM_CheckJump(pm, &pml)

		T.pmFriction(pm, &pml)

		if pm.Waterlevel >= 2 {
			T.pmWaterMove(pm, &pml)
		} else {
			angles := make([]float32, 3)
			copy(angles, pm.Viewangles[:])

			if angles[shared.PITCH] > 180 {
				angles[shared.PITCH] = angles[shared.PITCH] - 360
			}

			angles[shared.PITCH] /= 3

			shared.AngleVectors(angles, pml.forward[:], pml.right[:], pml.up[:])

			T.pmAirMove(pm, &pml)
		}
	}

	/* set groundentity, watertype, and waterlevel for final spot */
	PM_CatagorizePosition(pm, &pml)

	//     PM_UpdateUnderwaterSfx();

	PM_SnapPosition(pm, &pml)
}
