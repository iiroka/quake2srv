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
 * Combat code like damage, death and so on.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

func (G *qGame) killed(targ, inflictor, attacker *edict_t, damage int, point []float32) {
	if targ == nil || inflictor == nil || attacker == nil {
		return
	}

	if targ.Health < -999 {
		targ.Health = -999
	}

	targ.enemy = attacker

	if (targ.svflags&shared.SVF_MONSTER) != 0 && (targ.deadflag != DEAD_DEAD) {
		if (targ.monsterinfo.aiflags & AI_GOOD_GUY) == 0 {
			G.level.killed_monsters++

			if G.coop.Bool() && attacker.client != nil {
				attacker.client.resp.score++
			}

			/* medics won't heal monsters that they kill themselves */
			if len(attacker.Classname) > 0 && attacker.Classname == "monster_medic" {
				targ.owner = attacker
			}
		}
	}

	if (targ.movetype == MOVETYPE_PUSH) ||
		(targ.movetype == MOVETYPE_STOP) ||
		(targ.movetype == MOVETYPE_NONE) {
		/* doors, triggers, etc */
		targ.die(targ, inflictor, attacker, damage, point, G)
		return
	}

	if (targ.svflags&shared.SVF_MONSTER) != 0 && (targ.deadflag != DEAD_DEAD) {
		targ.touch = nil
		G.monster_death_use(targ)
	}

	targ.die(targ, inflictor, attacker, damage, point, G)
}

func (G *qGame) spawnDamage(dtype int, origin, normal []float32) {
	G.gi.WriteByte(shared.SvcTempEntity)
	G.gi.WriteByte(dtype)
	G.gi.WritePosition(origin)
	G.gi.WriteDir(normal)
	G.gi.Multicast(origin, shared.MULTICAST_PVS)
}

func (G *qGame) mReactToDamage(targ, attacker *edict_t) {
	if targ == nil || attacker == nil {
		return
	}

	if targ.Health <= 0 {
		return
	}

	if attacker.client == nil && (attacker.svflags&shared.SVF_MONSTER) == 0 {
		return
	}

	if (attacker == targ) || (attacker == targ.enemy) {
		return
	}

	/* if we are a good guy monster and our attacker is a player
	   or another good guy, do not get mad at them */
	if (targ.monsterinfo.aiflags & AI_GOOD_GUY) != 0 {
		if attacker.client != nil || (attacker.monsterinfo.aiflags&AI_GOOD_GUY) != 0 {
			return
		}
	}

	/* if attacker is a client, get mad at
	   them because he's good and we're not */
	if attacker.client != nil {
		targ.monsterinfo.aiflags &^= AI_SOUND_TARGET

		/* this can only happen in coop (both new and old
		   enemies are clients)  only switch if can't see
		   the current enemy */
		if targ.enemy != nil && targ.enemy.client != nil {
			if G.visible(targ, targ.enemy) {
				targ.oldenemy = attacker
				return
			}

			targ.oldenemy = targ.enemy
		}

		targ.enemy = attacker

		if (targ.monsterinfo.aiflags & AI_DUCKED) == 0 {
			G.foundTarget(targ)
		}

		return
	}

	/* it's the same base (walk/swim/fly) type and a
	   different classname and it's not a tank
	   (they spray too much), get mad at them */
	if ((targ.flags & (FL_FLY | FL_SWIM)) == (attacker.flags & (FL_FLY | FL_SWIM))) &&
		(targ.Classname != attacker.Classname) &&
		(attacker.Classname != "monster_tank") &&
		(attacker.Classname != "monster_supertank") &&
		(attacker.Classname != "monster_makron") &&
		(attacker.Classname != "monster_jorg") {
		if targ.enemy != nil && targ.enemy.client != nil {
			targ.oldenemy = targ.enemy
		}

		targ.enemy = attacker

		if (targ.monsterinfo.aiflags & AI_DUCKED) == 0 {
			G.foundTarget(targ)
		}
		/* if they *meant* to shoot us, then shoot back */
	} else if attacker.enemy == targ {
		if targ.enemy != nil && targ.enemy.client != nil {
			targ.oldenemy = targ.enemy
		}

		targ.enemy = attacker

		if (targ.monsterinfo.aiflags & AI_DUCKED) == 0 {
			G.foundTarget(targ)
		}
		/* otherwise get mad at whoever they are mad
		   at (help our buddy) unless it is us! */
	} else if attacker.enemy != nil {
		if targ.enemy != nil && targ.enemy.client != nil {
			targ.oldenemy = targ.enemy
		}

		targ.enemy = attacker.enemy

		if (targ.monsterinfo.aiflags & AI_DUCKED) == 0 {
			G.foundTarget(targ)
		}
	}
}

func (G *qGame) tDamage(targ, inflictor, attacker *edict_t,
	dir, point, normal []float32, damage, knockback, dflags, mod int) {

	if targ == nil || inflictor == nil || attacker == nil {
		return
	}

	if targ.takedamage == 0 {
		return
	}

	/* friendly fire avoidance if enabled you
	   can't hurt teammates (but you can hurt
	   yourself) knockback still occurs */
	// if ((targ != attacker) && ((deathmatch->value &&
	// 	  ((int)(dmflags->value) & (DF_MODELTEAMS | DF_SKINTEAMS))) ||
	// 	 coop->value))
	// {
	// 	if (OnSameTeam(targ, attacker))
	// 	{
	// 		if ((int)(dmflags->value) & DF_NO_FRIENDLY_FIRE)
	// 		{
	// 			damage = 0;
	// 		}
	// 		else
	// 		{
	// 			mod |= MOD_FRIENDLY_FIRE;
	// 		}
	// 	}
	// }

	G.meansOfDeath = mod

	// /* easy mode takes half damage */
	// if ((skill->value == SKILL_EASY) && (deathmatch->value == 0) && targ->client)
	// {
	// 	damage *= 0.5;

	// 	if (!damage)
	// 	{
	// 		damage = 1;
	// 	}
	// }

	client := targ.client

	te_sparks := shared.TE_SPARKS
	if (dflags & DAMAGE_BULLET) != 0 {
		te_sparks = shared.TE_BULLET_SPARKS
	}

	shared.VectorNormalize(dir)

	/* bonus damage for suprising a monster */
	// if (!(dflags & DAMAGE_RADIUS) && (targ->svflags & SVF_MONSTER) &&
	// 	(attacker->client) && (!targ->enemy) && (targ->health > 0))
	// {
	// 	damage *= 2;
	// }

	if (targ.flags & FL_NO_KNOCKBACK) != 0 {
		knockback = 0
	}

	/* figure momentum add */
	if (dflags & DAMAGE_NO_KNOCKBACK) == 0 {
		// 	if ((knockback) && (targ->movetype != MOVETYPE_NONE) &&
		// 		(targ->movetype != MOVETYPE_BOUNCE) &&
		// 		(targ->movetype != MOVETYPE_PUSH) &&
		// 		(targ->movetype != MOVETYPE_STOP))
		// 	{
		// 		vec3_t kvel;
		// 		float mass;

		// 		if (targ->mass < 50)
		// 		{
		// 			mass = 50;
		// 		}
		// 		else
		// 		{
		// 			mass = targ->mass;
		// 		}

		// 		if (targ->client && (attacker == targ))
		// 		{
		// 			/* This allows rocket jumps */
		// 			VectorScale(dir, 1600.0 * (float)knockback / mass, kvel);
		// 		}
		// 		else
		// 		{
		// 			VectorScale(dir, 500.0 * (float)knockback / mass, kvel);
		// 		}

		// 		VectorAdd(targ->velocity, kvel, targ->velocity);
		// 	}
	}

	take := damage
	// save := 0

	/* check for godmode */
	// if ((targ->flags & FL_GODMODE) && !(dflags & DAMAGE_NO_PROTECTION))
	// {
	// 	take = 0;
	// 	save = damage;
	// 	SpawnDamage(te_sparks, point, normal);
	// }

	/* check for invincibility */
	// if ((client && (client->invincible_framenum > level.framenum)) &&
	// 	!(dflags & DAMAGE_NO_PROTECTION))
	// {
	// 	if (targ->pain_debounce_time < level.time)
	// 	{
	// 		gi.sound(targ, CHAN_ITEM, gi.soundindex(
	// 					"items/protect4.wav"), 1, ATTN_NORM, 0);
	// 		targ->pain_debounce_time = level.time + 2;
	// 	}

	// 	take = 0;
	// 	save = damage;
	// }

	// psave = CheckPowerArmor(targ, point, normal, take, dflags);
	// take -= psave;

	// asave = CheckArmor(targ, point, normal, take, te_sparks, dflags);
	// take -= asave;

	/* treat cheat/powerup savings the same as armor */
	// asave += save

	/* do the damage */
	if take != 0 {
		if (targ.svflags&shared.SVF_MONSTER) != 0 || (client != nil) {
			G.spawnDamage(shared.TE_BLOOD, point, normal)
		} else {
			G.spawnDamage(te_sparks, point, normal)
		}

		targ.Health = targ.Health - take

		if targ.Health <= 0 {
			if (targ.svflags&shared.SVF_MONSTER) != 0 || (client != nil) {
				targ.flags |= FL_NO_KNOCKBACK
			}

			G.killed(targ, inflictor, attacker, take, point)
			return
		}
	}

	if (targ.svflags & shared.SVF_MONSTER) != 0 {
		G.mReactToDamage(targ, attacker)

		if (targ.monsterinfo.aiflags&AI_DUCKED) == 0 && take != 0 {
			targ.pain(targ, attacker, float32(knockback), take, G)

			// 		/* nightmare mode monsters don't go into pain frames often */
			// 		if (skill.value == SKILL_HARDPLUS) {
			// 			targ->pain_debounce_time = level.time + 5;
			// 		}
		}
	} else if client != nil {
		if (targ.flags&FL_GODMODE) == 0 && take != 0 {
			targ.pain(targ, attacker, float32(knockback), take, G)
		}
	} else if take != 0 {
		if targ.pain != nil {
			targ.pain(targ, attacker, float32(knockback), take, G)
		}
	}

	/* add to the damage inflicted on a player this frame
	   the total will be turned into screen blends and view
	   angle kicks at the end of the frame */
	if client != nil {
		// 	client->damage_parmor += psave;
		// 	client->damage_armor += asave;
		client.damage_blood += take
		client.damage_knockback += knockback
		copy(client.damage_from[:], point)
	}
}
