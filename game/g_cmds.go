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
 * Game command processing.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

/*
 * Use an inventory item
 */
func (G *qGame) cmd_Use_f(ent *edict_t, args []string) {
	//  int index;
	//  gitem_t *it;
	//  char *s;

	if ent == nil {
		return
	}

	s := ""
	for i := 1; i < len(args); i++ {
		s += args[i]
		if i < len(args)-1 {
			s += " "
		}
	}
	it := G.findItem(s)
	println("USE:", len(args), args[1])

	if it == nil {
		// G.gi.Cprintf(ent, PRINT_HIGH, "unknown item: %s\n", s)
		return
	}

	if it.use == nil {
		// 	 G.gi.Cprintf(ent, PRINT_HIGH, "Item is not usable.\n");
		return
	}

	//  index = ITEM_INDEX(it);

	//  if (!ent->client->pers.inventory[index])
	//  {
	// 	 gi.cprintf(ent, PRINT_HIGH, "Out of item: %s\n", s);
	// 	 return;
	//  }

	it.use(ent, it, G)
}

func (G *qGame) ClientCommand(sent shared.Edict_s, args []string) {
	if sent == nil {
		return
	}
	ent := sent.(*edict_t)

	if ent.client == nil {
		return /* not fully in game yet */
	}

	// if (Q_stricmp(cmd, "players") == 0)
	// {
	// 	Cmd_Players_f(ent);
	// 	return;
	// }

	// if (Q_stricmp(cmd, "say") == 0)
	// {
	// 	Cmd_Say_f(ent, false, false);
	// 	return;
	// }

	// if (Q_stricmp(cmd, "say_team") == 0)
	// {
	// 	Cmd_Say_f(ent, true, false);
	// 	return;
	// }

	// if (Q_stricmp(cmd, "score") == 0)
	// {
	// 	Cmd_Score_f(ent);
	// 	return;
	// }

	// if (Q_stricmp(cmd, "help") == 0)
	// {
	// 	Cmd_Help_f(ent);
	// 	return;
	// }

	// if (level.intermissiontime)
	// {
	// 	return;
	// }

	if args[0] == "use" {
		G.cmd_Use_f(ent, args)
		// }
		// else if (Q_stricmp(cmd, "drop") == 0)
		// {
		// 	Cmd_Drop_f(ent);
		// }
		// else if (Q_stricmp(cmd, "give") == 0)
		// {
		// 	Cmd_Give_f(ent);
		// }
		// else if (Q_stricmp(cmd, "god") == 0)
		// {
		// 	Cmd_God_f(ent);
		// }
		// else if (Q_stricmp(cmd, "notarget") == 0)
		// {
		// 	Cmd_Notarget_f(ent);
		// }
		// else if (Q_stricmp(cmd, "noclip") == 0)
		// {
		// 	Cmd_Noclip_f(ent);
		// }
		// else if (Q_stricmp(cmd, "inven") == 0)
		// {
		// 	Cmd_Inven_f(ent);
		// }
		// else if (Q_stricmp(cmd, "invnext") == 0)
		// {
		// 	SelectNextItem(ent, -1);
		// }
		// else if (Q_stricmp(cmd, "invprev") == 0)
		// {
		// 	SelectPrevItem(ent, -1);
		// }
		// else if (Q_stricmp(cmd, "invnextw") == 0)
		// {
		// 	SelectNextItem(ent, IT_WEAPON);
		// }
		// else if (Q_stricmp(cmd, "invprevw") == 0)
		// {
		// 	SelectPrevItem(ent, IT_WEAPON);
		// }
		// else if (Q_stricmp(cmd, "invnextp") == 0)
		// {
		// 	SelectNextItem(ent, IT_POWERUP);
		// }
		// else if (Q_stricmp(cmd, "invprevp") == 0)
		// {
		// 	SelectPrevItem(ent, IT_POWERUP);
		// }
		// else if (Q_stricmp(cmd, "invuse") == 0)
		// {
		// 	Cmd_InvUse_f(ent);
		// }
		// else if (Q_stricmp(cmd, "invdrop") == 0)
		// {
		// 	Cmd_InvDrop_f(ent);
		// }
		// else if (Q_stricmp(cmd, "weapprev") == 0)
		// {
		// 	Cmd_WeapPrev_f(ent);
		// }
		// else if (Q_stricmp(cmd, "weapnext") == 0)
		// {
		// 	Cmd_WeapNext_f(ent);
		// }
		// else if (Q_stricmp(cmd, "weaplast") == 0)
		// {
		// 	Cmd_WeapLast_f(ent);
		// }
		// else if (Q_stricmp(cmd, "kill") == 0)
		// {
		// 	Cmd_Kill_f(ent);
		// }
		// else if (Q_stricmp(cmd, "putaway") == 0)
		// {
		// 	Cmd_PutAway_f(ent);
		// }
		// else if (Q_stricmp(cmd, "wave") == 0)
		// {
		// 	Cmd_Wave_f(ent);
		// }
		// else if (Q_stricmp(cmd, "playerlist") == 0)
		// {
		// 	Cmd_PlayerList_f(ent);
		// }
		// else if (Q_stricmp(cmd, "teleport") == 0)
		// {
		// 	Cmd_Teleport_f(ent);
		// }
		// else if (Q_stricmp(cmd, "listentities") == 0)
		// {
		// 	Cmd_ListEntities_f(ent);
		// }
		// else if (Q_stricmp(cmd, "cycleweap") == 0)
		// {
		// 	Cmd_CycleWeap_f(ent);
		// }
		// else /* anything that doesn't match a command will be a chat */
		// {
		// 	Cmd_Say_f(ent, false, true);
	}
}
