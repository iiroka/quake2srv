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
 * HUD, deathmatch scoreboard, help computer and intermission stuff.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

/* ======================================================================= */

func (G *qGame) gSetStats(ent *edict_t) {
	// gitem_t *item;
	// int index, cells = 0;
	// int power_armor_type;

	if ent == nil {
		return
	}

	/* health */
	ent.client.ps.Stats[shared.STAT_HEALTH_ICON] = int16(G.level.pic_health)
	ent.client.ps.Stats[shared.STAT_HEALTH] = int16(ent.Health)

	/* ammo */
	if ent.client.ammo_index == 0 {
		ent.client.ps.Stats[shared.STAT_AMMO_ICON] = 0
		ent.client.ps.Stats[shared.STAT_AMMO] = 0
	} else {
		item := &gameitemlist[ent.client.ammo_index]
		ent.client.ps.Stats[shared.STAT_AMMO_ICON] = int16(G.gi.Imageindex(item.icon))
		ent.client.ps.Stats[shared.STAT_AMMO] =
			int16(ent.client.pers.inventory[ent.client.ammo_index])
	}

	/* armor */
	// power_armor_type = PowerArmorType(ent);

	// if (power_armor_type)
	// {
	// 	cells = ent->client->pers.inventory[ITEM_INDEX(FindItem("cells"))];

	// 	if (cells == 0)
	// 	{
	// 		/* ran out of cells for power armor */
	// 		ent->flags &= ~FL_POWER_ARMOR;
	// 		gi.sound(ent, CHAN_ITEM, gi.soundindex(
	// 						"misc/power2.wav"), 1, ATTN_NORM, 0);
	// 		power_armor_type = 0;
	// 	}
	// }

	index := G.armorIndex(ent)

	// if (power_armor_type && (!index || (level.framenum & 8)))
	// {
	// 	/* flash between power armor and other armor icon */
	// 	ent->client->ps.stats[STAT_ARMOR_ICON] = gi.imageindex("i_powershield");
	// 	ent->client->ps.stats[STAT_ARMOR] = cells;
	// }
	// else
	if index != 0 {
		item := getItemByIndex(index)
		ent.client.ps.Stats[shared.STAT_ARMOR_ICON] = int16(G.gi.Imageindex(item.icon))
		ent.client.ps.Stats[shared.STAT_ARMOR] = int16(ent.client.pers.inventory[index])
	} else {
		ent.client.ps.Stats[shared.STAT_ARMOR_ICON] = 0
		ent.client.ps.Stats[shared.STAT_ARMOR] = 0
	}

	// /* pickup message */
	// if (level.time > ent->client->pickup_msg_time)
	// {
	// ent->client->ps.stats[STAT_PICKUP_ICON] = 0;
	// ent->client->ps.stats[STAT_PICKUP_STRING] = 0;
	// }

	// /* timers */
	// if (ent->client->quad_framenum > level.framenum)
	// {
	// 	ent->client->ps.stats[STAT_TIMER_ICON] = gi.imageindex("p_quad");
	// 	ent->client->ps.stats[STAT_TIMER] =
	// 		(ent->client->quad_framenum - level.framenum) / 10;
	// }
	// else if (ent->client->invincible_framenum > level.framenum)
	// {
	// 	ent->client->ps.stats[STAT_TIMER_ICON] = gi.imageindex(
	// 			"p_invulnerability");
	// 	ent->client->ps.stats[STAT_TIMER] =
	// 		(ent->client->invincible_framenum - level.framenum) / 10;
	// }
	// else if (ent->client->enviro_framenum > level.framenum)
	// {
	// 	ent->client->ps.stats[STAT_TIMER_ICON] = gi.imageindex("p_envirosuit");
	// 	ent->client->ps.stats[STAT_TIMER] =
	// 		(ent->client->enviro_framenum - level.framenum) / 10;
	// }
	// else if (ent->client->breather_framenum > level.framenum)
	// {
	// 	ent->client->ps.stats[STAT_TIMER_ICON] = gi.imageindex("p_rebreather");
	// 	ent->client->ps.stats[STAT_TIMER] =
	// 		(ent->client->breather_framenum - level.framenum) / 10;
	// }
	// else
	// {
	// 	ent->client->ps.stats[STAT_TIMER_ICON] = 0;
	// 	ent->client->ps.stats[STAT_TIMER] = 0;
	// }

	// /* selected item */
	// if (ent->client->pers.selected_item == -1)
	// {
	// 	ent->client->ps.stats[STAT_SELECTED_ICON] = 0;
	// }
	// else
	// {
	// 	ent->client->ps.stats[STAT_SELECTED_ICON] =
	// 		gi.imageindex(itemlist[ent->client->pers.selected_item].icon);
	// }

	ent.client.ps.Stats[shared.STAT_SELECTED_ITEM] = int16(ent.client.pers.selected_item)

	/* layouts */
	ent.client.ps.Stats[shared.STAT_LAYOUTS] = 0

	if G.deathmatch.Bool() {
		// 	if ((ent->client->pers.health <= 0) || level.intermissiontime ||
		// 		ent->client->showscores)
		// 	{
		// 		ent->client->ps.stats[STAT_LAYOUTS] |= 1;
		// 	}

		// 	if (ent->client->showinventory && (ent->client->pers.health > 0))
		// 	{
		// 		ent->client->ps.stats[STAT_LAYOUTS] |= 2;
		// 	}
	} else {
		if ent.client.showscores || ent.client.showhelp {
			ent.client.ps.Stats[shared.STAT_LAYOUTS] |= 1
		}

		if ent.client.showinventory && (ent.client.pers.health > 0) {
			ent.client.ps.Stats[shared.STAT_LAYOUTS] |= 2
		}
	}

	/* frags */
	ent.client.ps.Stats[shared.STAT_FRAGS] = int16(ent.client.resp.score)

	/* help icon / current weapon if not shown */
	// if ent.client.pers.helpchanged && (G.level.framenum&8) != 0 {
	// 	ent->client->ps.stats[STAT_HELPICON] = gi.imageindex("i_help");
	// } else if ((ent.client.pers.hand == CENTER_HANDED) ||
	// 	(ent.client.ps.Fov > 91)) &&
	// 	ent.client.pers.weapon {
	// 	cvar_t *gun;
	// 	gun = gi.cvar("cl_gun", "2", 0);

	// 	if (gun->value != 2)
	// 	{
	// 		ent->client->ps.stats[STAT_HELPICON] = gi.imageindex(
	// 				ent->client->pers.weapon->icon);
	// 	}
	// 	else
	// 	{
	// 		ent->client->ps.stats[STAT_HELPICON] = 0;
	// 	}
	// } else {
	ent.client.ps.Stats[shared.STAT_HELPICON] = 0
	// }

	ent.client.ps.Stats[shared.STAT_SPECTATOR] = 0
}
