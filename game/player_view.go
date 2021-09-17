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
 * The "camera" through which the player looks into the game.
 *
 * =======================================================================
 */
package game

import (
	"math"
	"quake2srv/shared"
)

func (G *qGame) svCalcRoll(angles, velocity []float32) float32 {

	side := shared.DotProduct(velocity, G.player_view_right[:])
	var sign float32 = 1.0
	if side < 0 {
		sign = -1.0
	}
	side = float32(math.Abs(float64(side)))

	value := G.sv_rollangle.Float()

	if side < G.sv_rollspeed.Float() {
		side = side * value / G.sv_rollspeed.Float()
	} else {
		side = value
	}

	return side * sign
}

/*
 * fall from 128: 400 = 160000
 * fall from 256: 580 = 336400
 * fall from 384: 720 = 518400
 * fall from 512: 800 = 640000
 * fall from 640: 960 =
 *
 * damage = deltavelocity*deltavelocity  * 0.0001
 */
func (G *qGame) svCalcViewOffset(ent *edict_t) {

	/* base angles */
	angles := ent.client.ps.Kick_angles[:]

	/* if dead, fix the angle and don't add any kick */
	if ent.deadflag != 0 {
		copy(angles, []float32{0, 0, 0})

		ent.client.ps.Viewangles[shared.ROLL] = 40
		ent.client.ps.Viewangles[shared.PITCH] = -15
		ent.client.ps.Viewangles[shared.YAW] = ent.client.killer_yaw
	} else {
		/* add angles based on weapon kick */
		copy(angles, ent.client.kick_angles[:])

		/* add angles based on damage kick */
		ratio := (ent.client.v_dmg_time - G.level.time) / DAMAGE_TIME

		if ratio < 0 {
			ratio = 0
			ent.client.v_dmg_pitch = 0
			ent.client.v_dmg_roll = 0
		}

		angles[shared.PITCH] += ratio * ent.client.v_dmg_pitch
		angles[shared.ROLL] += ratio * ent.client.v_dmg_roll

		/* add pitch based on fall kick */
		ratio = (ent.client.fall_time - G.level.time) / FALL_TIME

		if ratio < 0 {
			ratio = 0
		}

		angles[shared.PITCH] += ratio * ent.client.fall_value

		/* add angles based on velocity */
		delta := shared.DotProduct(ent.velocity[:], G.player_view_forward[:])
		angles[shared.PITCH] += delta * G.run_pitch.Float()

		delta = shared.DotProduct(ent.velocity[:], G.player_view_right[:])
		angles[shared.ROLL] += delta * G.run_roll.Float()

		/* add angles based on bob */
		delta = G.player_view_bobfracsin * G.bob_pitch.Float() * G.player_view_xyspeed

		if (ent.client.ps.Pmove.Pm_flags & shared.PMF_DUCKED) != 0 {
			delta *= 6 /* crouching */
		}

		angles[shared.PITCH] += delta
		delta = G.player_view_bobfracsin * G.bob_roll.Float() * G.player_view_xyspeed

		if (ent.client.ps.Pmove.Pm_flags & shared.PMF_DUCKED) != 0 {
			delta *= 6 /* crouching */
		}

		if (G.player_view_bobcycle & 1) != 0 {
			delta = -delta
		}

		angles[shared.ROLL] += delta
	}

	/* base origin */
	v := []float32{0, 0, 0}

	/* add view height */
	v[2] += float32(ent.viewheight)

	/* add fall height */
	ratio := (ent.client.fall_time - G.level.time) / FALL_TIME

	if ratio < 0 {
		ratio = 0
	}

	v[2] -= ratio * ent.client.fall_value * 0.4

	/* add bob height */
	bob := G.player_view_bobfracsin * G.player_view_xyspeed * G.bob_up.Float()

	if bob > 6 {
		bob = 6
	}

	v[2] += bob

	/* add kick offset */
	shared.VectorAdd(v, ent.client.kick_origin[:], v)

	/* absolutely bound offsets
	so the view can never be
	outside the player box */
	if v[0] < -14 {
		v[0] = -14
	} else if v[0] > 14 {
		v[0] = 14
	}

	if v[1] < -14 {
		v[1] = -14
	} else if v[1] > 14 {
		v[1] = 14
	}

	if v[2] < -22 {
		v[2] = -22
	} else if v[2] > 30 {
		v[2] = 30
	}

	copy(ent.client.ps.Viewoffset[:], v)
}

/*
 * Called for each player at the end of
 * the server frame and right after spawning
 */
func (G *qGame) clientEndServerFrame(ent *edict_t) {
	//  float bobtime;
	//  int i;

	if ent == nil {
		return
	}

	G.current_player = ent
	G.current_client = ent.client

	/* If the origin or velocity have changed since ClientThink(),
	update the pmove values. This will happen when the client
	is pushed by a bmodel or kicked by an explosion.
	If it wasn't updated here, the view position would lag a frame
	behind the body position when pushed -- "sinking into plats" */
	for i := 0; i < 3; i++ {
		G.current_client.ps.Pmove.Origin[i] = int16(ent.s.Origin[i] * 8.0)
		G.current_client.ps.Pmove.Velocity[i] = int16(ent.velocity[i] * 8.0)
	}

	/* If the end of unit layout is displayed, don't give
	the player any normal movement attributes */
	//  if (level.intermissiontime) {
	// 	 current_client->ps.blend[3] = 0;
	// 	 current_client->ps.fov = 90;
	// 	 G_SetStats(ent);
	// 	 return;
	//  }

	shared.AngleVectors(ent.client.v_angle[:], G.player_view_forward[:], G.player_view_right[:], G.player_view_up[:])

	/* burn from lava, etc */
	//  P_WorldEffects();

	/* set model angles from view angles so other things in
	the world can tell which direction you are looking */
	if ent.client.v_angle[shared.PITCH] > 180 {
		ent.s.Angles[shared.PITCH] = (-360 + ent.client.v_angle[shared.PITCH]) / 3
	} else {
		ent.s.Angles[shared.PITCH] = ent.client.v_angle[shared.PITCH] / 3
	}

	ent.s.Angles[shared.YAW] = ent.client.v_angle[shared.YAW]
	ent.s.Angles[shared.ROLL] = 0
	ent.s.Angles[shared.ROLL] = G.svCalcRoll(ent.s.Angles[:], ent.velocity[:]) * 4

	/* calculate speed and cycle to be used for
	all cyclic walking effects */
	G.player_view_xyspeed = float32(math.Sqrt(
		float64(ent.velocity[0])*float64(ent.velocity[0]) + float64(ent.velocity[1])*
			float64(ent.velocity[1])))

	if G.player_view_xyspeed < 5 {
		G.player_view_bobmove = 0
		G.current_client.bobtime = 0 /* start at beginning of cycle again */
	} else if ent.groundentity != nil {
		/* so bobbing only cycles when on ground */
		if G.player_view_xyspeed > 210 {
			G.player_view_bobmove = 0.25
		} else if G.player_view_xyspeed > 100 {
			G.player_view_bobmove = 0.125
		} else {
			G.player_view_bobmove = 0.0625
		}
	}

	G.current_client.bobtime += G.player_view_bobmove
	bobtime := G.current_client.bobtime

	if (G.current_client.ps.Pmove.Pm_flags & shared.PMF_DUCKED) != 0 {
		bobtime *= 4
	}

	G.player_view_bobcycle = int(bobtime)
	G.player_view_bobfracsin = float32(math.Abs(math.Sin(float64(bobtime) * math.Pi)))

	/* detect hitting the floor */
	//  P_FallingDamage(ent);

	/* apply all the damage taken this frame */
	//  P_DamageFeedback(ent);

	/* determine the view offsets */
	G.svCalcViewOffset(ent)

	/* determine the gun offsets */
	//  SV_CalcGunOffset(ent);

	/* determine the full screen color blend
	must be after viewoffset, so eye contents
	can be accurately determined */
	//  SV_CalcBlend(ent);

	/* chase cam stuff */
	//  if (ent->client->resp.spectator)
	//  {
	// 	 G_SetSpectatorStats(ent);
	//  }
	//  else
	//  {
	G.gSetStats(ent)
	//  }

	//  G_CheckChaseStats(ent);

	//  G_SetClientEvent(ent);

	//  G_SetClientEffects(ent);

	//  G_SetClientSound(ent);

	//  G_SetClientFrame(ent);

	copy(ent.client.oldvelocity[:], ent.velocity[:])
	copy(ent.client.oldviewangles[:], ent.client.ps.Viewangles[:])

	/* clear weapon kicks */
	copy(ent.client.kick_origin[:], []float32{0, 0, 0})
	copy(ent.client.kick_angles[:], []float32{0, 0, 0})

	if (G.level.framenum & 31) == 0 {
		// 	 /* if the scoreboard is up, update it */
		// 	 if (ent->client->showscores)
		// 	 {
		// 		 DeathmatchScoreboardMessage(ent, ent->enemy);
		// 		 gi.unicast(ent, false);
		// 	 }

		// 	 /* if the help computer is up, update it */
		// 	 if (ent->client->showhelp)
		// 	 {
		// 		 ent->client->pers.helpchanged = 0;
		// 		 HelpComputerMessage(ent);
		// 		 gi.unicast(ent, false);
		// 	 }
	}

	/* if the inventory is up, update it */
	// if ent.client.showinventory {
	// 	 InventoryMessage(ent);
	// 	 gi.unicast(ent, false);
	// }
}
