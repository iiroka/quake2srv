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
		// 	 gi.WriteByte(svc_temp_entity);
		// 	 gi.WriteByte(TE_BLASTER);
		// 	 gi.WritePosition(self->s.origin);

		// 	 if (!plane)
		// 	 {
		// 		 gi.WriteDir(vec3_origin);
		// 	 }
		// 	 else
		// 	 {
		// 		 gi.WriteDir(plane->normal);
		// 	 }

		// 	 gi.multicast(self->s.origin, MULTICAST_PVS);
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
		// 		VectorMA(bolt->s.origin, -10, dir, bolt->s.origin);
		// 		bolt->touch(bolt, tr.ent, NULL, NULL);
	}
}
