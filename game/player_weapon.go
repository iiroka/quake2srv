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
 * Player weapons.
 *
 * =======================================================================
 */
package game

import (
	"quake2srv/game/misc"
	"quake2srv/shared"
)

func (G *qGame) projectSource(ent *edict_t, distance,
	forward, right, result []float32) {
	client := ent.client
	point := ent.s.Origin[:]
	// vec3_t     _distance;

	if client == nil {
		return
	}

	_distance := make([]float32, 3)
	copy(_distance, distance)

	// if (client->pers.hand == LEFT_HANDED) {
	// 	_distance[1] *= -1;
	// } else if (client->pers.hand == CENTER_HANDED) {
	// 	_distance[1] = 0;
	// }

	gProjectSource(point, _distance, forward, right, result)

	// Berserker: fix - now the projectile hits exactly where the scope is pointing.
	// if (aimfix->value) {
	// 	vec3_t start, end;
	// 	VectorSet(start, ent->s.origin[0], ent->s.origin[1], ent->s.origin[2] + ent->viewheight);
	// 	VectorMA(start, 8192, forward, end);

	// 	trace_t	tr = gi.trace(start, NULL, NULL, end, ent, MASK_SHOT);
	// 	if (tr.fraction < 1)
	// 	{
	// 		VectorSubtract(tr.endpos, result, forward);
	// 		VectorNormalize(forward);
	// 	}
	// }
}

/*
 * Each player can have two noise objects associated with it:
 * a personal noise (jumping, pain, weapon firing), and a weapon
 * target noise (bullet wall impacts)
 *
 * Monsters that don't directly see the player can move
 * to a noise in hopes of seeing the player from there.
 */
func (G *qGame) playerNoise(who *edict_t, where []float32, ntype int) {
	//  edict_t *noise;

	if who == nil {
		return
	}

	if ntype == PNOISE_WEAPON {
		// 	 if (who->client->silencer_shots) {
		// 		 who->client->silencer_shots--;
		// 		 return;
		// 	 }
	}

	//  if (deathmatch->value) {
	// 	 return;
	//  }

	if (who.flags & FL_NOTARGET) != 0 {
		return
	}

	var noise *edict_t
	if who.mynoise == nil {
		noise, _ = G.gSpawn()
		noise.Classname = "player_noise"
		copy(noise.mins[:], []float32{-8, -8, -8})
		copy(noise.maxs[:], []float32{8, 8, 8})
		noise.owner = who
		noise.svflags = shared.SVF_NOCLIENT
		who.mynoise = noise

		noise, _ = G.gSpawn()
		noise.Classname = "player_noise"
		copy(noise.mins[:], []float32{-8, -8, -8})
		copy(noise.maxs[:], []float32{8, 8, 8})
		noise.owner = who
		noise.svflags = shared.SVF_NOCLIENT
		who.mynoise2 = noise
	}

	if (ntype == PNOISE_SELF) || (ntype == PNOISE_WEAPON) {
		if G.level.framenum <= (G.level.sound_entity_framenum + 3) {
			return
		}

		noise = who.mynoise
		G.level.sound_entity = noise
		G.level.sound_entity_framenum = G.level.framenum
	} else {
		if G.level.framenum <= (G.level.sound2_entity_framenum + 3) {
			return
		}

		noise = who.mynoise2
		G.level.sound2_entity = noise
		G.level.sound2_entity_framenum = G.level.framenum
	}

	copy(noise.s.Origin[:], where)
	shared.VectorSubtract(where, noise.maxs[:], noise.absmin[:])
	shared.VectorAdd(where, noise.maxs[:], noise.absmax[:])
	// noise.last_sound_time = G.level.time
	G.gi.Linkentity(noise)
}

func Pickup_Weapon(ent, other *edict_t, G *qGame) bool {
	return G.do_pickup_Weapon(ent, other, G)
}

func Do_Pickup_Weapon(ent, other *edict_t, G *qGame) bool {
	// int index;
	// gitem_t *ammo;

	println("Do_Pickup_Weapon")
	if ent == nil || other == nil || G == nil {
		return false
	}

	index := ent.item.index

	if ((G.dmflags.Int()&shared.DF_WEAPONS_STAY) != 0 || G.coop.Bool()) &&
		other.client.pers.inventory[index] != 0 {
		if (ent.Spawnflags&(DROPPED_ITEM|DROPPED_PLAYER_ITEM)) == 0 &&
			(!G.coop_pickup_weapons.Bool() || (ent.flags&FL_COOP_TAKEN) != 0) {
			return false /* leave the weapon for others to pickup */
		}
	}

	other.client.pers.inventory[index]++

	if (ent.Spawnflags & DROPPED_ITEM) == 0 {
		/* give them some ammo with it */
		ammo := G.findItem(ent.item.ammo)

		if (G.dmflags.Int() & shared.DF_INFINITE_AMMO) != 0 {
			G.add_Ammo(other, ammo, 1000)
		} else {
			G.add_Ammo(other, ammo, ammo.quantity)
		}

		if (ent.Spawnflags & DROPPED_PLAYER_ITEM) == 0 {
			if G.deathmatch.Bool() {
				if (G.dmflags.Int() & shared.DF_WEAPONS_STAY) != 0 {
					ent.flags |= FL_RESPAWN
				} else {
					// SetRespawn(ent, 30)
				}
			}

			if G.coop.Bool() {
				ent.flags |= FL_RESPAWN
				ent.flags |= FL_COOP_TAKEN
			}
		}
	}

	if (other.client.pers.weapon != ent.item) &&
		(other.client.pers.inventory[index] == 1) &&
		(!G.deathmatch.Bool() || (other.client.pers.weapon == G.findItem("blaster"))) {
		other.client.newweapon = ent.item
	}

	return true
}

/*
 * The old weapon has been dropped all
 * the way, so make the new one current
 */
func (G *qGame) changeWeapon(ent *edict_t) {

	if ent == nil {
		return
	}

	//  if (ent.client.grenade_time) {
	// 	 ent->client->grenade_time = level.time;
	// 	 ent->client->weapon_sound = 0;
	// 	 weapon_grenade_fire(ent, false);
	// 	 ent->client->grenade_time = 0;
	//  }

	ent.client.pers.lastweapon = ent.client.pers.weapon
	ent.client.pers.weapon = ent.client.newweapon
	ent.client.newweapon = nil
	ent.client.machinegun_shots = 0

	/* set visible model */
	if ent.s.Modelindex == 255 {
		var i int
		if ent.client.pers.weapon != nil {
			i = ((ent.client.pers.weapon.weapmodel & 0xff) << 8)
		} else {
			i = 0
		}

		ent.s.Skinnum = (ent.index - 1) | i
	}

	if ent.client.pers.weapon != nil && len(ent.client.pers.weapon.ammo) > 0 {
		ent.client.ammo_index = G.findItemIndex(ent.client.pers.weapon.ammo)
	} else {
		ent.client.ammo_index = 0
	}

	if ent.client.pers.weapon == nil {
		/* dead */
		ent.client.ps.Gunindex = 0
		return
	}

	ent.client.weaponstate = WEAPON_ACTIVATING
	ent.client.ps.Gunframe = 0
	ent.client.ps.Gunindex = G.gi.Modelindex(ent.client.pers.weapon.view_model)

	ent.client.anim_priority = ANIM_PAIN

	if (ent.client.ps.Pmove.Pm_flags & shared.PMF_DUCKED) != 0 {
		ent.s.Frame = misc.FRAME_crpain1
		ent.client.anim_end = misc.FRAME_crpain4
	} else {
		ent.s.Frame = misc.FRAME_pain301
		ent.client.anim_end = misc.FRAME_pain304
	}
}

/*
 * Called by ClientBeginServerFrame and ClientThink
 */
func (G *qGame) thinkWeapon(ent *edict_t) {
	if ent == nil {
		return
	}

	/* if just died, put the weapon away */
	if ent.Health < 1 {
		ent.client.newweapon = nil
		G.changeWeapon(ent)
	}

	/* call active weapon think routine */
	if ent.client.pers.weapon != nil && ent.client.pers.weapon.weaponthink != nil {
		// 	 is_quad = (ent->client->quad_framenum > level.framenum);

		//  if (ent.client.silencer_shots) {
		// 	 is_silenced = MZ_SILENCED;
		//  } else {
		G.player_weapon_is_silenced = 0
		//  }

		ent.client.pers.weapon.weaponthink(ent, G)
	}
}

/*
 * Make the weapon ready if there is ammo
 */
func use_Weapon(ent *edict_t, item *gitem_t, G *qGame) {
	//  int ammo_index;
	//  gitem_t *ammo_item;

	if ent == nil || item == nil || G == nil {
		return
	}

	/* see if we're already using it */
	if item == ent.client.pers.weapon {
		return
	}

	if len(item.ammo) > 0 && !G.g_select_empty.Bool() && (item.flags&IT_AMMO) == 0 {
		// 	 ammo_item = FindItem(item->ammo);
		// 	 ammo_index = ITEM_INDEX(ammo_item);

		// 	 if (!ent->client->pers.inventory[ammo_index]) {
		// 		 gi.cprintf(ent, PRINT_HIGH, "No %s for %s.\n",
		// 				 ammo_item->pickup_name, item->pickup_name);
		// 		 return;
		// 	 }

		// 	 if (ent->client->pers.inventory[ammo_index] < item->quantity) {
		// 		 gi.cprintf(ent, PRINT_HIGH, "Not enough %s for %s.\n",
		// 				 ammo_item->pickup_name, item->pickup_name);
		// 		 return;
		// 	 }
	}

	/* change to this weapon when down */
	ent.client.newweapon = item
}

/*
 * A generic function to handle
 * the basics of weapon thinking
 */
func (G *qGame) weapon_Generic(ent *edict_t, FRAME_ACTIVATE_LAST, FRAME_FIRE_LAST,
	FRAME_IDLE_LAST, FRAME_DEACTIVATE_LAST int, pause_frames,
	fire_frames []int, fire func(*edict_t, *qGame)) {
	//  int n;

	FRAME_FIRE_FIRST := (FRAME_ACTIVATE_LAST + 1)
	FRAME_IDLE_FIRST := (FRAME_FIRE_LAST + 1)
	FRAME_DEACTIVATE_FIRST := (FRAME_IDLE_LAST + 1)

	if ent == nil || fire_frames == nil || fire == nil {
		return
	}

	if ent.deadflag != 0 || (ent.s.Modelindex != 255) { /* VWep animations screw up corpses */
		return
	}

	if ent.client.weaponstate == WEAPON_DROPPING {
		// 	 if (ent->client->ps.gunframe == FRAME_DEACTIVATE_LAST) {
		// 		 ChangeWeapon(ent);
		// 		 return;
		// 	 } else if ((FRAME_DEACTIVATE_LAST - ent->client->ps.gunframe) == 4) {
		// 		 ent->client->anim_priority = ANIM_REVERSE;

		// 		 if (ent->client->ps.pmove.pm_flags & PMF_DUCKED) {
		// 			 ent->s.frame = FRAME_crpain4 + 1;
		// 			 ent->client->anim_end = FRAME_crpain1;
		// 		 } else {
		// 			 ent->s.frame = FRAME_pain304 + 1;
		// 			 ent->client->anim_end = FRAME_pain301;
		// 		 }
		// 	 }

		ent.client.ps.Gunframe++
		return
	}

	if ent.client.weaponstate == WEAPON_ACTIVATING {
		if ent.client.ps.Gunframe == FRAME_ACTIVATE_LAST {
			ent.client.weaponstate = WEAPON_READY
			ent.client.ps.Gunframe = FRAME_IDLE_FIRST
			return
		}

		ent.client.ps.Gunframe++
		return
	}

	if (ent.client.newweapon != nil) && (ent.client.weaponstate != WEAPON_FIRING) {
		ent.client.weaponstate = WEAPON_DROPPING
		ent.client.ps.Gunframe = FRAME_DEACTIVATE_FIRST

		if (FRAME_DEACTIVATE_LAST - FRAME_DEACTIVATE_FIRST) < 4 {
			ent.client.anim_priority = ANIM_REVERSE

			if (ent.client.ps.Pmove.Pm_flags & shared.PMF_DUCKED) != 0 {
				ent.s.Frame = misc.FRAME_crpain4 + 1
				ent.client.anim_end = misc.FRAME_crpain1
			} else {
				ent.s.Frame = misc.FRAME_pain304 + 1
				ent.client.anim_end = misc.FRAME_pain301
			}
		}

		return
	}

	if ent.client.weaponstate == WEAPON_READY {
		if ((ent.client.latched_buttons |
			ent.client.buttons) & int(shared.BUTTON_ATTACK)) != 0 {
			ent.client.latched_buttons &^= int(shared.BUTTON_ATTACK)

			if (ent.client.ammo_index == 0) ||
				(ent.client.pers.inventory[ent.client.ammo_index] >= ent.client.pers.weapon.quantity) {
				ent.client.ps.Gunframe = FRAME_FIRE_FIRST
				ent.client.weaponstate = WEAPON_FIRING

				/* start the animation */
				ent.client.anim_priority = ANIM_ATTACK

				if (ent.client.ps.Pmove.Pm_flags & shared.PMF_DUCKED) != 0 {
					ent.s.Frame = misc.FRAME_crattak1 - 1
					ent.client.anim_end = misc.FRAME_crattak9
				} else {
					ent.s.Frame = misc.FRAME_attack1 - 1
					ent.client.anim_end = misc.FRAME_attack8
				}
			} else {
				// 			 if (level.time >= ent->pain_debounce_time)
				// 			 {
				// 				 gi.sound(ent, CHAN_VOICE, gi.soundindex(
				// 							 "weapons/noammo.wav"), 1, ATTN_NORM, 0);
				// 				 ent->pain_debounce_time = level.time + 1;
				// 			 }

				// 			 NoAmmoWeaponChange(ent);
			}
		} else {
			if ent.client.ps.Gunframe == FRAME_IDLE_LAST {
				ent.client.ps.Gunframe = FRAME_IDLE_FIRST
				return
			}

			// 		 if (pause_frames)
			// 		 {
			// 			 for (n = 0; pause_frames[n]; n++)
			// 			 {
			// 				 if (ent->client->ps.gunframe == pause_frames[n])
			// 				 {
			// 					 if (randk() & 15)
			// 					 {
			// 						 return;
			// 					 }
			// 				 }
			// 			 }
			// 		 }

			ent.client.ps.Gunframe++
			return
		}
	}

	if ent.client.weaponstate == WEAPON_FIRING {
		n := 0
		for n = 0; fire_frames[n] != 0; n++ {
			if ent.client.ps.Gunframe == fire_frames[n] {
				// 			 if (ent->client->quad_framenum > level.framenum)
				// 			 {
				// 				 gi.sound(ent, CHAN_ITEM, gi.soundindex(
				// 							 "items/damage3.wav"), 1, ATTN_NORM, 0);
				// 			 }

				fire(ent, G)
				break
			}
		}

		if fire_frames[n] == 0 {
			ent.client.ps.Gunframe++
		}

		if ent.client.ps.Gunframe == FRAME_IDLE_FIRST+1 {
			ent.client.weaponstate = WEAPON_READY
		}
	}
}

/* ====================================================================== */

/* BLASTER / HYPERBLASTER */

func (G *qGame) blaster_Fire(ent *edict_t, g_offset []float32, damage int,
	hyper bool, effect int) {

	if ent == nil {
		return
	}

	// if is_quad {
	// 	damage *= 4
	// }

	forward := make([]float32, 3)
	right := make([]float32, 3)
	shared.AngleVectors(ent.client.v_angle[:], forward, right, nil)
	offset := []float32{24, 8, float32(ent.viewheight) - 8}
	shared.VectorAdd(offset, g_offset, offset)
	start := make([]float32, 3)
	G.projectSource(ent, offset, forward, right, start)

	shared.VectorScale(forward, -2, ent.client.kick_origin[:])
	ent.client.kick_angles[0] = -1

	G.fire_blaster(ent, start, forward, damage, 1000, effect, hyper)

	/* send muzzle flash */
	G.gi.WriteByte(shared.SvcMuzzleflash)
	G.gi.WriteShort(ent.index)

	// if (hyper)
	// {
	// 	gi.WriteByte(MZ_HYPERBLASTER | is_silenced);
	// }
	// else
	// {
	G.gi.WriteByte(shared.MZ_BLASTER | G.player_weapon_is_silenced)
	// }

	G.gi.Multicast(ent.s.Origin[:], shared.MULTICAST_PVS)

	G.playerNoise(ent, start, PNOISE_WEAPON)
}

func weapon_Blaster_Fire(ent *edict_t, G *qGame) {

	if ent == nil || G == nil {
		return
	}

	damage := 10
	if G.deathmatch.Bool() {
		damage = 15
	}

	G.blaster_Fire(ent, []float32{0, 0, 0}, damage, false, shared.EF_BLASTER)
	ent.client.ps.Gunframe++
}

func weapon_Blaster(ent *edict_t, G *qGame) {
	// static int pause_frames[] = {19, 32, 0};
	// static int fire_frames[] = {5, 0};

	if ent == nil || G == nil {
		return
	}

	G.weapon_Generic(ent, 4, 8, 52, 55, []int{19, 32, 0},
		[]int{5, 0}, weapon_Blaster_Fire)
}
