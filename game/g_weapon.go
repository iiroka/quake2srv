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
 * Weapon support functions.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

/*
 * This is an internal support routine
 * used for bullet/pellet based weapons.
 */
func (G *qGame) fire_lead(self *edict_t, start, aimdir []float32, damage, kick,
	te_impact, hspread, vspread, mod int) {
	//  trace_t tr;
	//  vec3_t dir;
	//  vec3_t forward, right, up;
	//  vec3_t end;
	//  float r;
	//  float u;
	//  vec3_t water_start;
	water_start := make([]float32, 3)
	water := false
	content_mask := shared.MASK_SHOT | shared.MASK_WATER

	if self == nil {
		return
	}

	tr := G.gi.Trace(self.s.Origin[:], nil, nil, start, self, shared.MASK_SHOT)

	if !(tr.Fraction < 1.0) {
		dir := make([]float32, 3)
		vectoangles(aimdir, dir)
		forward := make([]float32, 3)
		right := make([]float32, 3)
		up := make([]float32, 3)
		shared.AngleVectors(dir, forward, right, up)

		r := shared.Crandk() * float32(hspread)
		u := shared.Crandk() * float32(vspread)
		end := make([]float32, 3)
		shared.VectorMA(start, 8192, forward, end)
		shared.VectorMA(end, r, right, end)
		shared.VectorMA(end, u, up, end)

		if (G.gi.Pointcontents(start) & shared.MASK_WATER) != 0 {
			water = true
			copy(water_start, start)
			content_mask &^= shared.MASK_WATER
		}

		tr = G.gi.Trace(start, nil, nil, end, self, content_mask)

		/* see if we hit water */
		if (tr.Contents & shared.MASK_WATER) != 0 {

			water = true
			copy(water_start, tr.Endpos[:])

			if shared.VectorCompare(start, tr.Endpos[:]) == 0 {
				color := shared.SPLASH_UNKNOWN
				if (tr.Contents & shared.CONTENTS_WATER) != 0 {
					if tr.Surface.Name == "*brwater" {
						color = shared.SPLASH_BROWN_WATER
					} else {
						color = shared.SPLASH_BLUE_WATER
					}
				} else if (tr.Contents & shared.CONTENTS_SLIME) != 0 {
					color = shared.SPLASH_SLIME
				} else if (tr.Contents & shared.CONTENTS_LAVA) != 0 {
					color = shared.SPLASH_LAVA
				}

				if color != shared.SPLASH_UNKNOWN {
					G.gi.WriteByte(shared.SvcTempEntity)
					G.gi.WriteByte(shared.TE_SPLASH)
					G.gi.WriteByte(8)
					G.gi.WritePosition(tr.Endpos[:])
					G.gi.WriteDir(tr.Plane.Normal[:])
					G.gi.WriteByte(color)
					G.gi.Multicast(tr.Endpos[:], shared.MULTICAST_PVS)
				}

				/* change bullet's course when it enters water */
				shared.VectorSubtract(end, start, dir)
				vectoangles(dir, dir)
				shared.AngleVectors(dir, forward, right, up)
				r = shared.Crandk() * float32(hspread*2)
				u = shared.Crandk() * float32(vspread*2)
				shared.VectorMA(water_start, 8192, forward, end)
				shared.VectorMA(end, r, right, end)
				shared.VectorMA(end, u, up, end)
			}

			/* re-trace ignoring water this time */
			tr = G.gi.Trace(water_start, nil, nil, end, self, shared.MASK_SHOT)
		}
	}

	/* send gun puff / flash */
	if !((tr.Surface != nil) && (tr.Surface.Flags&shared.SURF_SKY) != 0) {
		if tr.Fraction < 1.0 {
			if tr.Ent.(*edict_t).takedamage != 0 {
				G.tDamage(tr.Ent.(*edict_t), self, self, aimdir, tr.Endpos[:], tr.Plane.Normal[:],
					damage, kick, DAMAGE_BULLET, mod)
			} else {
				if tr.Surface.Name[0:3] != "sky" {
					G.gi.WriteByte(shared.SvcTempEntity)
					G.gi.WriteByte(te_impact)
					G.gi.WritePosition(tr.Endpos[:])
					G.gi.WriteDir(tr.Plane.Normal[:])
					G.gi.Multicast(tr.Endpos[:], shared.MULTICAST_PVS)

					// 				 if (self->client)
					// 				 {
					// 					 PlayerNoise(self, tr.endpos, PNOISE_IMPACT);
					// 				 }
				}
			}
		}
	}

	/* if went through water, determine
	where the end and make a bubble trail */
	if water {
		// 	 vec3_t pos;

		// 	 VectorSubtract(tr.endpos, water_start, dir);
		// 	 VectorNormalize(dir);
		// 	 VectorMA(tr.endpos, -2, dir, pos);

		// 	 if (gi.pointcontents(pos) & MASK_WATER)
		// 	 {
		// 		 VectorCopy(pos, tr.endpos);
		// 	 }
		// 	 else
		// 	 {
		// 		 tr = gi.trace(pos, NULL, NULL, water_start, tr.ent, MASK_WATER);
		// 	 }

		// 	 VectorAdd(water_start, tr.endpos, pos);
		// 	 VectorScale(pos, 0.5, pos);

		// 	 gi.WriteByte(svc_temp_entity);
		// 	 gi.WriteByte(TE_BUBBLETRAIL);
		// 	 gi.WritePosition(water_start);
		// 	 gi.WritePosition(tr.endpos);
		// 	 gi.multicast(pos, MULTICAST_PVS);
	}
}

/*
 * Fires a single round.  Used for machinegun and
 * chaingun.  Would be fine for pistols, rifles, etc....
 */
func (G *qGame) fire_bullet(self *edict_t, start, aimdir []float32, damage,
	kick, hspread, vspread, mod int) {

	if self == nil {
		return
	}

	G.fire_lead(self, start, aimdir, damage, kick, shared.TE_GUNSHOT, hspread, vspread, mod)
}

/*
 * Shoots shotgun pellets. Used
 * by shotgun and super shotgun.
 */
func (G *qGame) fire_shotgun(self *edict_t, start, aimdir []float32, damage,
	kick, hspread, vspread, count, mod int) {

	if self == nil {
		return
	}

	for i := 0; i < count; i++ {
		G.fire_lead(self, start, aimdir, damage, kick, shared.TE_SHOTGUN,
			hspread, vspread, mod)
	}
}

/*
 * Fires a single blaster bolt.
 * Used by the blaster and hyper blaster.
 */
func blaster_touch(self, other *edict_t, plane *shared.Cplane_t, surf *shared.Csurface_t, G *qGame) {
	//  int mod;

	if self == nil || G == nil {
		return
	}

	if other == nil { /* plane and surf can be NULL */
		G.gFreeEdict(self)
		return
	}

	if other == self.owner {
		return
	}

	if surf != nil && (surf.Flags&shared.SURF_SKY) != 0 {
		G.gFreeEdict(self)
		return
	}

	if self.owner != nil && self.owner.client != nil {
		G.playerNoise(self.owner, self.s.Origin[:], PNOISE_IMPACT)
	}

	if other.takedamage != 0 {
		mod := MOD_BLASTER
		if (self.Spawnflags & 1) != 0 {
			mod = MOD_HYPERBLASTER
		}

		if plane != nil {
			G.tDamage(other, self, self.owner, self.velocity[:], self.s.Origin[:],
				plane.Normal[:], self.Dmg, 1, DAMAGE_ENERGY, mod)
		} else {
			G.tDamage(other, self, self.owner, self.velocity[:], self.s.Origin[:],
				[]float32{0, 0, 0}, self.Dmg, 1, DAMAGE_ENERGY, mod)
		}
	} else {
		G.gi.WriteByte(shared.SvcTempEntity)
		G.gi.WriteByte(shared.TE_BLASTER)
		G.gi.WritePosition(self.s.Origin[:])

		if plane == nil {
			G.gi.WriteDir([]float32{0, 0, 0})
		} else {
			G.gi.WriteDir(plane.Normal[:])
		}

		G.gi.Multicast(self.s.Origin[:], shared.MULTICAST_PVS)
	}

	G.gFreeEdict(self)
}

func (G *qGame) fire_blaster(self *edict_t, start, dir []float32, damage,
	speed, effect int, hyper bool) {
	// 	edict_t *bolt;
	// 	trace_t tr;

	if self == nil {
		return
	}

	shared.VectorNormalize(dir)

	bolt, _ := G.gSpawn()
	bolt.svflags = shared.SVF_DEADMONSTER

	/* yes, I know it looks weird that projectiles are deadmonsters
	   what this means is that when prediction is used against the object
	   (blaster/hyperblaster shots), the player won't be solid clipped against
	   the object.  Right now trying to run into a firing hyperblaster
	   is very jerky since you are predicted 'against' the shots. */
	copy(bolt.s.Origin[:], start)
	copy(bolt.s.Old_origin[:], start)
	vectoangles(dir, bolt.s.Angles[:])
	shared.VectorScale(dir, float32(speed), bolt.velocity[:])
	bolt.movetype = MOVETYPE_FLYMISSILE
	bolt.clipmask = shared.MASK_SHOT
	bolt.solid = shared.SOLID_BBOX
	bolt.s.Effects |= uint(effect)
	bolt.s.Renderfx |= shared.RF_NOSHADOW
	for i := range bolt.mins {
		bolt.mins[i] = 0
		bolt.maxs[i] = 0
	}
	bolt.s.Modelindex = G.gi.Modelindex("models/objects/laser/tris.md2")
	// 	bolt->s.sound = gi.soundindex("misc/lasfly.wav");
	bolt.owner = self
	bolt.touch = blaster_touch
	bolt.nextthink = G.level.time + 2
	bolt.think = gFreeEdictFunc
	bolt.Dmg = damage
	bolt.Classname = "bolt"

	// 	if (hyper)
	// 	{
	// 		bolt->spawnflags = 1;
	// 	}

	G.gi.Linkentity(bolt)

	// 	if (self->client) {
	// 		check_dodge(self, bolt->s.origin, dir, speed);
	// 	}

	tr := G.gi.Trace(self.s.Origin[:], nil, nil, bolt.s.Origin[:], bolt, shared.MASK_SHOT)

	if tr.Fraction < 1.0 {
		shared.VectorMA(bolt.s.Origin[:], -10, dir, bolt.s.Origin[:])
		bolt.touch(bolt, tr.Ent.(*edict_t), nil, nil, G)
	}
}
