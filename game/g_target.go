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
 * Targets.
 *
 * =======================================================================
 */
package game

import (
	"fmt"
	"quake2srv/shared"
	"strings"
)

func spTargetSpeaker(ent *edict_t, G *qGame) error {

	if ent == nil || G == nil {
		return nil
	}

	if len(G.st.Noise) == 0 {
		G.gi.Dprintf("target_speaker with no noise set at %s\n", vtos(ent.s.Origin[:]))
		return nil
	}

	var buffer string
	if !strings.Contains(G.st.Noise, ".wav") {
		buffer = fmt.Sprintf("%s.wav", G.st.Noise)
	} else {
		buffer = G.st.Noise
	}

	ent.noise_index = G.gi.Soundindex(buffer)

	if ent.Volume == 0 {
		ent.Volume = 1.0
	}

	if ent.Attenuation == 0 {
		ent.Attenuation = 1.0
	} else if ent.Attenuation == -1 { /* use -1 so 0 defaults to 1 */
		ent.Attenuation = 0
	}

	/* check for prestarted looping sound */
	if (ent.Spawnflags & 1) != 0 {
		ent.s.Sound = ent.noise_index
	}

	// ent.use = Use_Target_Speaker

	/* must link the entity so we get areas and clusters so
	   the server can determine who to send updates to */
	G.gi.Linkentity(ent)
	return nil
}

/* ========================================================== */

func use_Target_Help(ent, other, activator *edict_t, G *qGame) {

	if ent == nil || G == nil {
		return
	}

	if (ent.Spawnflags & 1) != 0 {
		G.game.helpmessage1 = ent.Message
	} else {
		G.game.helpmessage2 = ent.Message
	}

	G.game.helpchanged++
}

/*
 * QUAKED target_help (1 0 1) (-16 -16 -24) (16 16 24) help1
 * When fired, the "message" key becomes the current personal computer string,
 * and the message light will be set on all clients status bars.
 */
func spTargetHelp(ent *edict_t, G *qGame) error {

	if ent == nil || G == nil {
		return nil
	}

	// if (deathmatch->value)
	// {
	// 	/* auto-remove for deathmatch */
	// 	G_FreeEdict(ent);
	// 	return;
	// }

	if len(ent.Message) == 0 {
		G.gi.Dprintf("%s with no message at %s\n", ent.Classname,
			vtos(ent.s.Origin[:]))
		G.gFreeEdict(ent)
		return nil
	}

	ent.use = use_Target_Help
	return nil
}

/* ========================================================== */

/*
 * QUAKED target_explosion (1 0 0) (-8 -8 -8) (8 8 8)
 * Spawns an explosion temporary entity when used.
 *
 * "delay"		wait this long before going off
 * "dmg"		how much radius damage should be done, defaults to 0
 */
func target_explosion_explode(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	println("target_explosion_explode")
	G.gi.WriteByte(shared.SvcTempEntity)
	G.gi.WriteByte(shared.TE_EXPLOSION1)
	G.gi.WritePosition(self.s.Origin[:])
	G.gi.Multicast(self.s.Origin[:], shared.MULTICAST_PHS)

	//  T_RadiusDamage(self, self->activator, self->dmg, NULL,
	// 		 self->dmg + 40, MOD_EXPLOSIVE);

	save := self.Delay
	self.Delay = 0
	G.gUseTargets(self, self.activator)
	self.Delay = save
}

func use_target_explosion(self, other /* unused */, activator *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}
	self.activator = activator

	if activator == nil {
		return
	}

	if self.Delay == 0 {
		target_explosion_explode(self, G)
		return
	}

	self.think = target_explosion_explode
	self.nextthink = G.level.time + self.Delay
}

func spTargetExplosion(ent *edict_t, G *qGame) error {
	if ent == nil || G == nil {
		return nil
	}

	ent.use = use_target_explosion
	ent.svflags = shared.SVF_NOCLIENT
	return nil
}
