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
 * Jump in into the game.so and support functions.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

/* ====================================================================== */

func (G *qGame) clientEndServerFrames() {

	/* calc the player views now that all
	   pushing  and damage has been added */
	for i := 0; i < G.maxclients.Int(); i++ {
		ent := &G.g_edicts[1+i]

		if !ent.inuse || ent.client == nil {
			continue
		}

		G.clientEndServerFrame(ent)
	}
}

/*
 * Advances the world by 0.1 seconds
 */
func (G *qGame) RunFrame() error {
	//  int i;
	//  edict_t *ent;

	G.level.framenum++
	G.level.time = float32(G.level.framenum) * FRAMETIME

	//  gibsthisframe = 0;
	//  debristhisframe = 0;

	/* choose a client for monsters to target this frame */
	//  AI_SetSightClient();

	//  /* exit intermissions */
	//  if (level.exitintermission) {
	// 	 ExitLevel();
	// 	 return;
	//  }

	/* treat each object in turn
	even the world gets a chance
	to think */

	for i := 0; i < G.num_edicts; i++ {
		ent := &G.g_edicts[i]
		if !ent.inuse {
			continue
		}

		G.level.current_entity = ent

		copy(ent.s.Old_origin[:], ent.s.Origin[:])

		/* if the ground entity moved, make sure we are still on it */
		// 	 if ((ent->groundentity) &&
		// 		 (ent->groundentity->linkcount != ent->groundentity_linkcount))
		// 	 {
		// 		 ent->groundentity = NULL;

		// 		 if (!(ent->flags & (FL_SWIM | FL_FLY)) &&
		// 			 (ent->svflags & SVF_MONSTER))
		// 		 {
		// 			 M_CheckGround(ent);
		// 		 }
		// 	 }

		if (i > 0) && (i <= G.maxclients.Int()) {
			G.clientBeginServerFrame(ent)
			continue
		}

		if err := G.runEntity(ent); err != nil {
			return err
		}
	}

	/* see if it is time to end a deathmatch */
	//  CheckDMRules();

	/* see if needpass needs updated */
	//  CheckNeedPass();

	/* build the playerstate_t structures for all players */
	G.clientEndServerFrames()
	return nil
}

func (G *qGame) Edict(index int) shared.Edict_s {
	return &G.g_edicts[index]
}

func (G *qGame) NumEdicts() int {
	return G.num_edicts
}

func (G *qGame) MaxEdicts() int {
	return G.game.maxentities
}

func (G *qGame) Shutdown() {
	G.gi.Dprintf("==== ShutdownGame ====\n")
	// gi.FreeTags(TAG_LEVEL);
	// gi.FreeTags(TAG_GAME);
}
