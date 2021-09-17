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
 * Trigger.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

/*
 * QUAKED trigger_relay (.5 .5 .5) (-8 -8 -8) (8 8 8)
 * This fixed size trigger cannot be touched,
 * it can only be fired by other events.
 */
func trigger_relay_use(self, other, activator *edict_t, G *qGame) {
	if self == nil || activator == nil || G == nil {
		return
	}

	G.gUseTargets(self, activator)
}

/*
 * The wait time has passed, so
 * set back up for another activation
 */
func multi_wait(ent *edict_t, G *qGame) {
	if ent == nil {
		return
	}

	ent.nextthink = 0
}

/*
 * The trigger was just activated
 * ent->activator should be set to
 * the activator so it can be held
 * through a delay so wait for the
 * delay time before firing
 */
func (G *qGame) multi_trigger(ent *edict_t) {
	if ent == nil {
		return
	}

	if ent.nextthink != 0 {
		return /* already been triggered */
	}

	G.gUseTargets(ent, ent.activator)

	if ent.Wait > 0 {
		ent.think = multi_wait
		ent.nextthink = G.level.time + ent.Wait
	} else {
		/* we can't just remove (self) here,
		because this is a touch function
		called while looping through area
		links... */
		ent.touch = nil
		ent.nextthink = G.level.time + FRAMETIME
		ent.think = gFreeEdictFunc
	}
}

func use_Multi(self, other, activator *edict_t, G *qGame) {
	if self == nil || activator == nil || G == nil {
		return
	}

	self.activator = activator
	G.multi_trigger(self)
}

func touch_Multi(self, other *edict_t, plane *shared.Cplane_t, surf *shared.Csurface_t, G *qGame) {
	if self == nil || other == nil || G == nil {
		return
	}

	if other.client != nil {
		if (self.Spawnflags & 2) != 0 {
			return
		}
	} else if (other.svflags & shared.SVF_MONSTER) != 0 {
		if (self.Spawnflags & 1) == 0 {
			return
		}
	} else {
		return
	}

	if shared.VectorCompare(self.movedir[:], []float32{0, 0, 0}) == 0 {

		forward := make([]float32, 3)
		shared.AngleVectors(other.s.Angles[:], forward, nil, nil)

		if shared.DotProduct(forward, self.movedir[:]) < 0 {
			return
		}
	}

	self.activator = other
	G.multi_trigger(self)
}

/*
 * QUAKED trigger_multiple (.5 .5 .5) ? MONSTER NOT_PLAYER TRIGGERED
 * Variable sized repeatable trigger.  Must be targeted at one or more
 * entities. If "delay" is set, the trigger waits some time after
 * activating before firing.
 *
 * "wait" : Seconds between triggerings. (.2 default)
 *
 * sounds
 * 1)	secret
 * 2)	beep beep
 * 3)	large switch
 * 4)
 *
 * set "message" to text string
 */
func trigger_enable(self, other, activator *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	self.solid = shared.SOLID_TRIGGER
	self.use = use_Multi
	G.gi.Linkentity(self)
}

func spTriggerMultiple(ent *edict_t, G *qGame) error {
	if ent == nil || G == nil {
		return nil
	}

	if ent.Sounds == 1 {
		ent.noise_index = G.gi.Soundindex("misc/secret.wav")
	} else if ent.Sounds == 2 {
		ent.noise_index = G.gi.Soundindex("misc/talk.wav")
	} else if ent.Sounds == 3 {
		ent.noise_index = G.gi.Soundindex("misc/trigger1.wav")
	}

	if ent.Wait == 0 {
		ent.Wait = 0.2
	}

	ent.touch = touch_Multi
	ent.movetype = MOVETYPE_NONE
	ent.svflags |= shared.SVF_NOCLIENT

	if (ent.Spawnflags & 4) != 0 {
		ent.solid = shared.SOLID_NOT
		ent.use = trigger_enable
	} else {
		ent.solid = shared.SOLID_TRIGGER
		ent.use = use_Multi
	}

	if shared.VectorCompare(ent.s.Angles[:], []float32{0, 0, 0}) == 0 {
		gSetMovedir(ent.s.Angles[:], ent.movedir[:])
	}

	G.gi.Setmodel(ent, ent.Model)
	G.gi.Linkentity(ent)
	return nil
}

/*
 * QUAKED trigger_once (.5 .5 .5) ? x x TRIGGERED
 * Triggers once, then removes itself.
 *
 * You must set the key "target" to the name of another
 * object in the level that has a matching "targetname".
 *
 * If TRIGGERED, this trigger must be triggered before it is live.
 *
 * sounds
 *  1) secret
 *  2) beep beep
 *  3) large switch
 *
 * "message" string to be displayed when triggered
 */

func spTriggerOnce(ent *edict_t, G *qGame) error {
	if ent == nil || G == nil {
		return nil
	}

	/* make old maps work because I
	messed up on flag assignments here
	triggered was on bit 1 when it
	should have been on bit 4 */
	if (ent.Spawnflags & 1) != 0 {
		v := make([]float32, 3)
		shared.VectorMA(ent.mins[:], 0.5, ent.size[:], v)
		ent.Spawnflags &^= 1
		ent.Spawnflags |= 4
		G.gi.Dprintf("fixed TRIGGERED flag on %s at %s\n", ent.Classname, vtos(v))
	}

	ent.Wait = -1
	return spTriggerMultiple(ent, G)
}

func spTriggerRelay(self *edict_t, G *qGame) error {
	if self == nil || G == nil {
		return nil
	}

	self.use = trigger_relay_use
	return nil
}

/*
 * QUAKED trigger_always (.5 .5 .5) (-8 -8 -8) (8 8 8)
 * This trigger will always fire. It is activated by the world.
 */
func spTriggerAlways(ent *edict_t, G *qGame) error {
	if ent == nil || G == nil {
		return nil
	}

	/* we must have some delay to make
	sure our use targets are present */
	if ent.Delay < 0.2 {
		ent.Delay = 0.2
	}

	G.gUseTargets(ent, ent)
	return nil
}
