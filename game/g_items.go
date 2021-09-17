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
 * Item handling and item definitions.
 *
 * =======================================================================
 */
package game

import "quake2srv/shared"

const HEALTH_IGNORE_MAX = 1
const HEALTH_TIMED = 2

var jacketarmor_info = gitem_armor_t{25, 50, .30, .00, ARMOR_JACKET}
var combatarmor_info = gitem_armor_t{50, 100, .60, .30, ARMOR_COMBAT}
var bodyarmor_info = gitem_armor_t{100, 200, .80, .60, ARMOR_BODY}

/* ====================================================================== */

func getItemByIndex(index int) *gitem_t {
	if (index == 0) || (index >= len(gameitemlist)) {
		return nil
	}

	return &gameitemlist[index]
}
func itemIndex(item *gitem_t) int {
	for i, it := range gameitemlist {
		if it.classname == item.classname {
			return i
		}
	}
	return 0
}

func (G *qGame) findItem(pickup_name string) *gitem_t {

	if len(pickup_name) == 0 {
		return nil
	}

	for i, it := range gameitemlist {
		if len(it.pickup_name) == 0 {
			continue
		}

		if it.pickup_name == pickup_name {
			return &gameitemlist[i]
		}
	}

	return nil
}

func (G *qGame) findItemIndex(pickup_name string) int {

	if len(pickup_name) == 0 {
		return -1
	}

	for i, it := range gameitemlist {
		if len(it.pickup_name) == 0 {
			continue
		}

		if it.pickup_name == pickup_name {
			return i
		}
	}

	return -1
}

/*
 * Called by worldspawn
 */
func (G *qGame) setItemNames() {

	for i, it := range gameitemlist {
		G.gi.Configstring(shared.CS_ITEMS+i, it.pickup_name)
	}

	G.jacket_armor_index = G.findItemIndex("Jacket Armor")
	G.combat_armor_index = G.findItemIndex("Combat Armor")
	G.body_armor_index = G.findItemIndex("Body Armor")
	G.power_screen_index = G.findItemIndex("Power Screen")
	G.power_shield_index = G.findItemIndex("Power Shield")
}

/* ====================================================================== */

func touch_Item(ent, other *edict_t, plane /* unused */ *shared.Cplane_t, surf /* unused */ *shared.Csurface_t, G *qGame) {

	if ent == nil || other == nil || G == nil {
		return
	}

	if other.client == nil {
		return
	}

	if other.Health < 1 {
		return /* dead people can't pickup */
	}

	if ent.item.pickup == nil {
		return /* not a grabbable item? */
	}

	taken := ent.item.pickup(ent, other, G)

	if taken {
		/* flash the screen */
		other.client.bonus_alpha = 0.25

		/* show icon and name on status bar */
		other.client.ps.Stats[shared.STAT_PICKUP_ICON] = int16(G.gi.Imageindex(ent.item.icon))
		other.client.ps.Stats[shared.STAT_PICKUP_STRING] = int16(shared.CS_ITEMS + itemIndex(ent.item))
		other.client.pickup_msg_time = G.level.time + 3.0

		/* change selected item */
		if ent.item.use != nil {
			other.client.ps.Stats[shared.STAT_SELECTED_ITEM] = int16(itemIndex(ent.item))
			other.client.pers.selected_item = int(other.client.ps.Stats[shared.STAT_SELECTED_ITEM])
		}

		// if (ent->item->pickup == Pickup_Health) {
		// 	if (ent->count == 2) {
		// 		gi.sound(other, CHAN_ITEM, gi.soundindex(
		// 						"items/s_health.wav"), 1, ATTN_NORM, 0);
		// 	}
		// 	else if (ent->count == 10)
		// 	{
		// 		gi.sound(other, CHAN_ITEM, gi.soundindex(
		// 						"items/n_health.wav"), 1, ATTN_NORM, 0);
		// 	}
		// 	else if (ent->count == 25)
		// 	{
		// 		gi.sound(other, CHAN_ITEM, gi.soundindex(
		// 						"items/l_health.wav"), 1, ATTN_NORM, 0);
		// 	}
		// 	else /* (ent->count == 100) */
		// 	{
		// 		gi.sound(other, CHAN_ITEM, gi.soundindex(
		// 						"items/m_health.wav"), 1, ATTN_NORM, 0);
		// 	}
		// }
		// else if (ent->item->pickup_sound)
		// {
		// 	gi.sound(other, CHAN_ITEM, gi.soundindex(
		// 					ent->item->pickup_sound), 1, ATTN_NORM, 0);
		// }

		/* activate item instantly if appropriate */
		/* moved down here so activation sounds override the pickup sound */
		// if (deathmatch->value)
		// {
		// 	if ((((int)dmflags->value & DF_INSTANT_ITEMS) &&
		// 		 (ent->item->flags & IT_INSTANT_USE)) ||
		// 		((ent->item->use == Use_Quad) &&
		// 		 (ent->spawnflags & DROPPED_PLAYER_ITEM)))
		// 	{
		// 		if ((ent->item->use == Use_Quad) &&
		// 			(ent->spawnflags & DROPPED_PLAYER_ITEM))
		// 		{
		// 			quad_drop_timeout_hack =
		// 				(ent->nextthink - level.time) / FRAMETIME;
		// 		}

		// 		if (ent->item->use)
		// 		{
		// 			ent->item->use(other, ent->item);
		// 		}
		// 	}
		// }
	}

	if (ent.Spawnflags & ITEM_TARGETS_USED) == 0 {
		G.gUseTargets(ent, other)
		ent.Spawnflags |= ITEM_TARGETS_USED
	}

	if !taken {
		return
	}

	if !((G.coop.Bool()) &&
		(ent.item.flags&IT_STAY_COOP) != 0) ||
		(ent.Spawnflags&(DROPPED_ITEM|DROPPED_PLAYER_ITEM)) != 0 {
		if (ent.flags & FL_RESPAWN) != 0 {
			ent.flags &^= FL_RESPAWN
		} else {
			G.gFreeEdict(ent)
		}
	}
}

func use_Item(ent, other /* unused */, activator /* unused */ *edict_t, G *qGame) {
	if ent == nil {
		return
	}

	ent.svflags &^= shared.SVF_NOCLIENT
	ent.use = nil

	if (ent.Spawnflags & ITEM_NO_TOUCH) != 0 {
		ent.solid = shared.SOLID_BBOX
		ent.touch = nil
	} else {
		ent.solid = shared.SOLID_TRIGGER
		ent.touch = touch_Item
	}

	G.gi.Linkentity(ent)
}

/* ====================================================================== */

func droptofloor(ent *edict_t, G *qGame) {

	if ent == nil || G == nil {
		return
	}

	copy(ent.mins[:], []float32{-15, -15, -15})
	copy(ent.maxs[:], []float32{15, 15, 15})

	if len(ent.Model) > 0 {
		G.gi.Setmodel(ent, ent.Model)
	} else {
		G.gi.Setmodel(ent, ent.item.world_model)
	}

	ent.solid = shared.SOLID_TRIGGER
	ent.movetype = MOVETYPE_TOSS
	// ent->touch = Touch_Item;

	dest := make([]float32, 3)
	shared.VectorAdd(ent.s.Origin[:], []float32{0, 0, -129}, dest)

	tr := G.gi.Trace(ent.s.Origin[:], ent.mins[:], ent.maxs[:], dest, ent, shared.MASK_SOLID)

	if tr.Startsolid {
		G.gi.Dprintf("droptofloor: %s startsolid at %s\n", ent.Classname,
			vtos(ent.s.Origin[:]))
		G.gFreeEdict(ent)
		return
	}

	copy(ent.s.Origin[:], tr.Endpos[:])

	// if (ent->team) {
	// 	ent->flags &= ~FL_TEAMSLAVE;
	// 	ent->chain = ent->teamchain;
	// 	ent->teamchain = NULL;

	// 	ent->svflags |= SVF_NOCLIENT;
	// 	ent->solid = SOLID_NOT;

	// 	if (ent == ent->teammaster) {
	// 		ent->nextthink = level.time + FRAMETIME;
	// 		ent->think = DoRespawn;
	// 	}
	// }

	if (ent.Spawnflags & ITEM_NO_TOUCH) != 0 {
		ent.solid = shared.SOLID_BBOX
		ent.touch = nil
		ent.s.Effects &^= shared.EF_ROTATE
		ent.s.Renderfx &^= shared.RF_GLOW
	}

	if (ent.Spawnflags & ITEM_TRIGGER_SPAWN) != 0 {
		ent.svflags |= shared.SVF_NOCLIENT
		ent.solid = shared.SOLID_NOT
		ent.use = use_Item
	}

	G.gi.Linkentity(ent)
}

/*
 * ============
 * Sets the clipping size and
 * plants the object on the floor.
 *
 * Items can't be immediately dropped
 * to floor, because they might be on
 * an entity that hasn't spawned yet.
 * ============
 */
func (G *qGame) spawnItem(ent *edict_t, item *gitem_t) {
	if ent == nil || item == nil {
		return
	}

	// PrecacheItem(item)

	if ent.Spawnflags != 0 {
		if ent.Classname != "key_power_cube" {
			ent.Spawnflags = 0
			G.gi.Dprintf("%s at %s has invalid spawnflags set\n",
				ent.Classname, vtos(ent.s.Origin[:]))
		}
	}

	/* some items will be prevented in deathmatch */
	//  if (deathmatch.value) {
	// 	 if ((int)dmflags->value & DF_NO_ARMOR) != 0 {
	// 		 if ((item->pickup == Pickup_Armor) ||
	// 			 (item->pickup == Pickup_PowerArmor)) {
	// 			 G_FreeEdict(ent);
	// 			 return;
	// 		 }
	// 	 }

	// 	 if ((int)dmflags->value & DF_NO_ITEMS) != 0 {
	// 		 if (item->pickup == Pickup_Powerup) {
	// 			 G_FreeEdict(ent);
	// 			 return;
	// 		 }
	// 	 }

	// 	 if ((int)dmflags->value & DF_NO_HEALTH) != 0 {
	// 		 if ((item->pickup == Pickup_Health) ||
	// 			 (item->pickup == Pickup_Adrenaline) ||
	// 			 (item->pickup == Pickup_AncientHead)) {
	// 			 G_FreeEdict(ent);
	// 			 return;
	// 		 }
	// 	 }

	// 	 if ((int)dmflags->value & DF_INFINITE_AMMO) != 0 {
	// 		 if ((item->flags == IT_AMMO) ||
	// 			 (strcmp(ent->classname, "weapon_bfg") == 0)) {
	// 			 G_FreeEdict(ent);
	// 			 return;
	// 		 }
	// 	 }
	//  }

	//  if (coop->value && (strcmp(ent->classname, "key_power_cube") == 0)) {
	// 	 ent->spawnflags |= (1 << (8 + level.power_cubes));
	// 	 level.power_cubes++;
	//  }

	/* don't let them drop items that stay in a coop game */
	//  if ((coop->value) && (item->flags & IT_STAY_COOP) != 0) {
	// 	 item->drop = NULL;
	//  }

	ent.item = item
	ent.nextthink = G.level.time + 2*FRAMETIME /* items start after other solids */
	ent.think = droptofloor
	//  ent->s.effects = item->world_model_flags;
	ent.s.Renderfx = shared.RF_GLOW

	if len(ent.Model) > 0 {
		G.gi.Modelindex(ent.Model)
	}
}

func Pickup_Health(ent, other *edict_t, G *qGame) bool {
	if ent == nil || other == nil || G == nil {
		return false
	}

	if (ent.Style & HEALTH_IGNORE_MAX) == 0 {
		if other.Health >= other.max_health {
			return false
		}
	}

	other.Health += ent.count

	if (ent.Style & HEALTH_IGNORE_MAX) == 0 {
		if other.Health > other.max_health {
			other.Health = other.max_health
		}
	}

	if (ent.Style & HEALTH_TIMED) != 0 {
		// 	ent.think = MegaHealth_think
		ent.nextthink = G.level.time + 5
		ent.owner = other
		ent.flags |= FL_RESPAWN
		ent.svflags |= shared.SVF_NOCLIENT
		ent.solid = shared.SOLID_NOT
	} else {
		// 	if (ent.Spawnflags&DROPPED_ITEM) == 0 && G.deathmatch.Bool() {
		// 		SetRespawn(ent, 30)
		// 	}
	}

	return true
}

/* ====================================================================== */

func (G *qGame) armorIndex(ent *edict_t) int {
	if ent == nil {
		return 0
	}

	if ent.client == nil {
		return 0
	}

	if ent.client.pers.inventory[G.jacket_armor_index] > 0 {
		return G.jacket_armor_index
	}

	if ent.client.pers.inventory[G.combat_armor_index] > 0 {
		return G.combat_armor_index
	}

	if ent.client.pers.inventory[G.body_armor_index] > 0 {
		return G.body_armor_index
	}

	return 0
}

/* ====================================================================== */

var gameitemlist = []gitem_t{
	{}, /* leave index 0 alone */

	/* QUAKED item_armor_body (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_armor_body",
		nil, // Pickup_Armor,
		nil,
		nil,
		nil,
		"misc/ar1_pkup.wav",
		"models/items/armor/body/tris.md2", shared.EF_ROTATE,
		"",
		"i_bodyarmor",
		"Body Armor",
		3,
		0,
		"",
		IT_ARMOR,
		0,
		&bodyarmor_info,
		ARMOR_BODY,
		"",
	},

	/* QUAKED item_armor_combat (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_armor_combat",
		nil, // Pickup_Armor,
		nil,
		nil,
		nil,
		"misc/ar1_pkup.wav",
		"models/items/armor/combat/tris.md2", shared.EF_ROTATE,
		"",
		"i_combatarmor",
		"Combat Armor",
		3,
		0,
		"",
		IT_ARMOR,
		0,
		&combatarmor_info,
		ARMOR_COMBAT,
		"",
	},

	/* QUAKED item_armor_jacket (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_armor_jacket",
		nil, // Pickup_Armor,
		nil,
		nil,
		nil,
		"misc/ar1_pkup.wav",
		"models/items/armor/jacket/tris.md2", shared.EF_ROTATE,
		"",
		"i_jacketarmor",
		"Jacket Armor",
		3,
		0,
		"",
		IT_ARMOR,
		0,
		&jacketarmor_info,
		ARMOR_JACKET,
		"",
	},

	/* QUAKED item_armor_shard (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_armor_shard",
		nil, // Pickup_Armor,
		nil,
		nil,
		nil,
		"misc/ar2_pkup.wav",
		"models/items/armor/shard/tris.md2", shared.EF_ROTATE,
		"",
		"i_jacketarmor",
		"Armor Shard",
		3,
		0,
		"",
		IT_ARMOR,
		0,
		nil,
		ARMOR_SHARD,
		"",
	},

	/* QUAKED item_power_screen (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_power_screen",
		nil, // Pickup_PowerArmor,
		nil, // Use_PowerArmor,
		nil, // Drop_PowerArmor,
		nil,
		"misc/ar3_pkup.wav",
		"models/items/armor/screen/tris.md2", shared.EF_ROTATE,
		"",
		"i_powerscreen",
		"Power Screen",
		0,
		60,
		"",
		IT_ARMOR,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED item_power_shield (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_power_shield",
		nil, // Pickup_PowerArmor,
		nil, // Use_PowerArmor,
		nil, // Drop_PowerArmor,
		nil,
		"misc/ar3_pkup.wav",
		"models/items/armor/shield/tris.md2", shared.EF_ROTATE,
		"",
		"i_powershield",
		"Power Shield",
		0,
		60,
		"",
		IT_ARMOR,
		0,
		nil,
		0,
		"misc/power2.wav misc/power1.wav",
	},

	/* weapon_blaster (.3 .3 1) (-16 -16 -16) (16 16 16)
	   always owned, never in the world */
	{
		"weapon_blaster",
		nil,
		use_Weapon,
		nil,
		weapon_Blaster,
		"misc/w_pkup.wav",
		"", 0,
		"models/weapons/v_blast/tris.md2",
		"w_blaster",
		"Blaster",
		0,
		0,
		"",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_BLASTER,
		nil,
		0,
		"weapons/blastf1a.wav misc/lasfly.wav",
	},

	/* QUAKED weapon_shotgun (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"weapon_shotgun",
		nil, // Pickup_Weapon,
		use_Weapon,
		nil, // Drop_Weapon,
		nil, // Weapon_Shotgun,
		"misc/w_pkup.wav",
		"models/weapons/g_shotg/tris.md2", shared.EF_ROTATE,
		"models/weapons/v_shotg/tris.md2",
		"w_shotgun",
		"Shotgun",
		0,
		1,
		"Shells",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_SHOTGUN,
		nil,
		0,
		"weapons/shotgf1b.wav weapons/shotgr1b.wav",
	},

	/* QUAKED weapon_supershotgun (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"weapon_supershotgun",
		nil, // Pickup_Weapon,
		use_Weapon,
		nil, // Drop_Weapon,
		nil, // Weapon_SuperShotgun,
		"misc/w_pkup.wav",
		"models/weapons/g_shotg2/tris.md2", shared.EF_ROTATE,
		"models/weapons/v_shotg2/tris.md2",
		"w_sshotgun",
		"Super Shotgun",
		0,
		2,
		"Shells",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_SUPERSHOTGUN,
		nil,
		0,
		"weapons/sshotf1b.wav",
	},

	/* QUAKED weapon_machinegun (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"weapon_machinegun",
		nil, // Pickup_Weapon,
		use_Weapon,
		nil, // Drop_Weapon,
		nil, // Weapon_Machinegun,
		"misc/w_pkup.wav",
		"models/weapons/g_machn/tris.md2", shared.EF_ROTATE,
		"models/weapons/v_machn/tris.md2",
		"w_machinegun",
		"Machinegun",
		0,
		1,
		"Bullets",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_MACHINEGUN,
		nil,
		0,
		"weapons/machgf1b.wav weapons/machgf2b.wav weapons/machgf3b.wav weapons/machgf4b.wav weapons/machgf5b.wav",
	},

	/* QUAKED weapon_chaingun (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"weapon_chaingun",
		nil, // Pickup_Weapon,
		use_Weapon,
		nil, // Drop_Weapon,
		nil, // Weapon_Chaingun,
		"misc/w_pkup.wav",
		"models/weapons/g_chain/tris.md2", shared.EF_ROTATE,
		"models/weapons/v_chain/tris.md2",
		"w_chaingun",
		"Chaingun",
		0,
		1,
		"Bullets",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_CHAINGUN,
		nil,
		0,
		"weapons/chngnu1a.wav weapons/chngnl1a.wav weapons/machgf3b.wav` weapons/chngnd1a.wav",
	},

	/* QUAKED ammo_grenades (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"ammo_grenades",
		nil, // Pickup_Ammo,
		use_Weapon,
		nil, // Drop_Ammo,
		nil, // Weapon_Grenade,
		"misc/am_pkup.wav",
		"models/items/ammo/grenades/medium/tris.md2", 0,
		"models/weapons/v_handgr/tris.md2",
		"a_grenades",
		"Grenades",
		3,
		5,
		"grenades",
		IT_AMMO | IT_WEAPON,
		WEAP_GRENADES,
		nil,
		AMMO_GRENADES,
		"weapons/hgrent1a.wav weapons/hgrena1b.wav weapons/hgrenc1b.wav weapons/hgrenb1a.wav weapons/hgrenb2a.wav ",
	},

	/* QUAKED weapon_grenadelauncher (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"weapon_grenadelauncher",
		nil, // Pickup_Weapon,
		use_Weapon,
		nil, // Drop_Weapon,
		nil, // Weapon_GrenadeLauncher,
		"misc/w_pkup.wav",
		"models/weapons/g_launch/tris.md2", shared.EF_ROTATE,
		"models/weapons/v_launch/tris.md2",
		"w_glauncher",
		"Grenade Launcher",
		0,
		1,
		"Grenades",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_GRENADELAUNCHER,
		nil,
		0,
		"models/objects/grenade/tris.md2 weapons/grenlf1a.wav weapons/grenlr1b.wav weapons/grenlb1b.wav",
	},

	/* QUAKED weapon_rocketlauncher (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"weapon_rocketlauncher",
		nil, // Pickup_Weapon,
		use_Weapon,
		nil, // Drop_Weapon,
		nil, // Weapon_RocketLauncher,
		"misc/w_pkup.wav",
		"models/weapons/g_rocket/tris.md2", shared.EF_ROTATE,
		"models/weapons/v_rocket/tris.md2",
		"w_rlauncher",
		"Rocket Launcher",
		0,
		1,
		"Rockets",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_ROCKETLAUNCHER,
		nil,
		0,
		"models/objects/rocket/tris.md2 weapons/rockfly.wav weapons/rocklf1a.wav weapons/rocklr1b.wav models/objects/debris2/tris.md2",
	},

	/* QUAKED weapon_hyperblaster (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"weapon_hyperblaster",
		nil, // Pickup_Weapon,
		use_Weapon,
		nil, // Drop_Weapon,
		nil, // Weapon_HyperBlaster,
		"misc/w_pkup.wav",
		"models/weapons/g_hyperb/tris.md2", shared.EF_ROTATE,
		"models/weapons/v_hyperb/tris.md2",
		"w_hyperblaster",
		"HyperBlaster",
		0,
		1,
		"Cells",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_HYPERBLASTER,
		nil,
		0,
		"weapons/hyprbu1a.wav weapons/hyprbl1a.wav weapons/hyprbf1a.wav weapons/hyprbd1a.wav misc/lasfly.wav",
	},

	/* QUAKED weapon_railgun (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"weapon_railgun",
		nil, // Pickup_Weapon,
		use_Weapon,
		nil, // Drop_Weapon,
		nil, // Weapon_Railgun,
		"misc/w_pkup.wav",
		"models/weapons/g_rail/tris.md2", shared.EF_ROTATE,
		"models/weapons/v_rail/tris.md2",
		"w_railgun",
		"Railgun",
		0,
		1,
		"Slugs",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_RAILGUN,
		nil,
		0,
		"weapons/rg_hum.wav",
	},

	/* QUAKED weapon_bfg (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"weapon_bfg",
		nil, // Pickup_Weapon,
		use_Weapon,
		nil, // Drop_Weapon,
		nil, // Weapon_BFG,
		"misc/w_pkup.wav",
		"models/weapons/g_bfg/tris.md2", shared.EF_ROTATE,
		"models/weapons/v_bfg/tris.md2",
		"w_bfg",
		"BFG10K",
		0,
		50,
		"Cells",
		IT_WEAPON | IT_STAY_COOP,
		WEAP_BFG,
		nil,
		0,
		"sprites/s_bfg1.sp2 sprites/s_bfg2.sp2 sprites/s_bfg3.sp2 weapons/bfg__f1y.wav weapons/bfg__l1a.wav weapons/bfg__x1b.wav weapons/bfg_hum.wav",
	},

	/* QUAKED ammo_shells (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"ammo_shells",
		nil, // Pickup_Ammo,
		nil,
		nil, // Drop_Ammo,
		nil,
		"misc/am_pkup.wav",
		"models/items/ammo/shells/medium/tris.md2", 0,
		"",
		"a_shells",
		"Shells",
		3,
		10,
		"",
		IT_AMMO,
		0,
		nil,
		AMMO_SHELLS,
		"",
	},

	/* QUAKED ammo_bullets (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"ammo_bullets",
		nil, // Pickup_Ammo,
		nil,
		nil, // Drop_Ammo,
		nil,
		"misc/am_pkup.wav",
		"models/items/ammo/bullets/medium/tris.md2", 0,
		"",
		"a_bullets",
		"Bullets",
		3,
		50,
		"",
		IT_AMMO,
		0,
		nil,
		AMMO_BULLETS,
		"",
	},

	/* QUAKED ammo_cells (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"ammo_cells",
		nil, // Pickup_Ammo,
		nil,
		nil, // Drop_Ammo,
		nil,
		"misc/am_pkup.wav",
		"models/items/ammo/cells/medium/tris.md2", 0,
		"",
		"a_cells",
		"Cells",
		3,
		50,
		"",
		IT_AMMO,
		0,
		nil,
		AMMO_CELLS,
		"",
	},

	/* QUAKED ammo_rockets (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"ammo_rockets",
		nil, // Pickup_Ammo,
		nil,
		nil, // Drop_Ammo,
		nil,
		"misc/am_pkup.wav",
		"models/items/ammo/rockets/medium/tris.md2", 0,
		"",
		"a_rockets",
		"Rockets",
		3,
		5,
		"",
		IT_AMMO,
		0,
		nil,
		AMMO_ROCKETS,
		"",
	},

	/* QUAKED ammo_slugs (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"ammo_slugs",
		nil, // Pickup_Ammo,
		nil,
		nil, // Drop_Ammo,
		nil,
		"misc/am_pkup.wav",
		"models/items/ammo/slugs/medium/tris.md2", 0,
		"",
		"a_slugs",
		"Slugs",
		3,
		10,
		"",
		IT_AMMO,
		0,
		nil,
		AMMO_SLUGS,
		"",
	},

	/* QUAKED item_quad (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_quad",
		nil, // Pickup_Powerup,
		nil, // Use_Quad,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/quaddama/tris.md2", shared.EF_ROTATE,
		"",
		"p_quad",
		"Quad Damage",
		2,
		60,
		"",
		IT_POWERUP | IT_INSTANT_USE,
		0,
		nil,
		0,
		"items/damage.wav items/damage2.wav items/damage3.wav",
	},

	/* QUAKED item_invulnerability (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_invulnerability",
		nil, // Pickup_Powerup,
		nil, // Use_Invulnerability,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/invulner/tris.md2", shared.EF_ROTATE,
		"",
		"p_invulnerability",
		"Invulnerability",
		2,
		300,
		"",
		IT_POWERUP | IT_INSTANT_USE,
		0,
		nil,
		0,
		"items/protect.wav items/protect2.wav items/protect4.wav",
	},

	/* QUAKED item_silencer (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_silencer",
		nil, // Pickup_Powerup,
		nil, // Use_Silencer,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/silencer/tris.md2", shared.EF_ROTATE,
		"",
		"p_silencer",
		"Silencer",
		2,
		60,
		"",
		IT_POWERUP | IT_INSTANT_USE,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED item_breather (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_breather",
		nil, // Pickup_Powerup,
		nil, // Use_Breather,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/breather/tris.md2", shared.EF_ROTATE,
		"",
		"p_rebreather",
		"Rebreather",
		2,
		60,
		"",
		IT_STAY_COOP | IT_POWERUP | IT_INSTANT_USE,
		0,
		nil,
		0,
		"items/airout.wav",
	},

	/* QUAKED item_enviro (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_enviro",
		nil, // Pickup_Powerup,
		nil, // Use_Envirosuit,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/enviro/tris.md2", shared.EF_ROTATE,
		"",
		"p_envirosuit",
		"Environment Suit",
		2,
		60,
		"",
		IT_STAY_COOP | IT_POWERUP | IT_INSTANT_USE,
		0,
		nil,
		0,
		"items/airout.wav",
	},

	/* QUAKED item_ancient_head (.3 .3 1) (-16 -16 -16) (16 16 16)
	   Special item that gives +2 to maximum health */
	{
		"item_ancient_head",
		nil, // Pickup_AncientHead,
		nil,
		nil,
		nil,
		"items/pkup.wav",
		"models/items/c_head/tris.md2", shared.EF_ROTATE,
		"",
		"i_fixme",
		"Ancient Head",
		2,
		60,
		"",
		0,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED item_adrenaline (.3 .3 1) (-16 -16 -16) (16 16 16)
	   gives +1 to maximum health */
	{
		"item_adrenaline",
		nil, // Pickup_Adrenaline,
		nil,
		nil,
		nil,
		"items/pkup.wav",
		"models/items/adrenal/tris.md2", shared.EF_ROTATE,
		"",
		"p_adrenaline",
		"Adrenaline",
		2,
		60,
		"",
		0,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED item_bandolier (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_bandolier",
		nil, // Pickup_Bandolier,
		nil,
		nil,
		nil,
		"items/pkup.wav",
		"models/items/band/tris.md2", shared.EF_ROTATE,
		"",
		"p_bandolier",
		"Bandolier",
		2,
		60,
		"",
		0,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED item_pack (.3 .3 1) (-16 -16 -16) (16 16 16) */
	{
		"item_pack",
		nil, // Pickup_Pack,
		nil,
		nil,
		nil,
		"items/pkup.wav",
		"models/items/pack/tris.md2", shared.EF_ROTATE,
		"",
		"i_pack",
		"Ammo Pack",
		2,
		180,
		"",
		0,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED key_data_cd (0 .5 .8) (-16 -16 -16) (16 16 16)
	   key for computer centers */
	{
		"key_data_cd",
		nil, // Pickup_Key,
		nil,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/keys/data_cd/tris.md2", shared.EF_ROTATE,
		"",
		"k_datacd",
		"Data CD",
		2,
		0,
		"",
		IT_STAY_COOP | IT_KEY,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED key_power_cube (0 .5 .8) (-16 -16 -16) (16 16 16) TRIGGER_SPAWN NO_TOUCH
	   warehouse circuits */
	{
		"key_power_cube",
		nil, // Pickup_Key,
		nil,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/keys/power/tris.md2", shared.EF_ROTATE,
		"",
		"k_powercube",
		"Power Cube",
		2,
		0,
		"",
		IT_STAY_COOP | IT_KEY,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED key_pyramid (0 .5 .8) (-16 -16 -16) (16 16 16)
	   key for the entrance of jail3 */
	{
		"key_pyramid",
		nil, // Pickup_Key,
		nil,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/keys/pyramid/tris.md2", shared.EF_ROTATE,
		"",
		"k_pyramid",
		"Pyramid Key",
		2,
		0,
		"",
		IT_STAY_COOP | IT_KEY,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED key_data_spinner (0 .5 .8) (-16 -16 -16) (16 16 16)
	   key for the city computer */
	{
		"key_data_spinner",
		nil, // Pickup_Key,
		nil,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/keys/spinner/tris.md2", shared.EF_ROTATE,
		"",
		"k_dataspin",
		"Data Spinner",
		2,
		0,
		"",
		IT_STAY_COOP | IT_KEY,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED key_pass (0 .5 .8) (-16 -16 -16) (16 16 16)
	   security pass for the security level */
	{
		"key_pass",
		nil, // Pickup_Key,
		nil,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/keys/pass/tris.md2", shared.EF_ROTATE,
		"",
		"k_security",
		"Security Pass",
		2,
		0,
		"",
		IT_STAY_COOP | IT_KEY,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED key_blue_key (0 .5 .8) (-16 -16 -16) (16 16 16)
	   normal door key - blue */
	{
		"key_blue_key",
		nil, // Pickup_Key,
		nil,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/keys/key/tris.md2", shared.EF_ROTATE,
		"",
		"k_bluekey",
		"Blue Key",
		2,
		0,
		"",
		IT_STAY_COOP | IT_KEY,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED key_red_key (0 .5 .8) (-16 -16 -16) (16 16 16)
	   normal door key - red */
	{
		"key_red_key",
		nil, // Pickup_Key,
		nil,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/keys/red_key/tris.md2", shared.EF_ROTATE,
		"",
		"k_redkey",
		"Red Key",
		2,
		0,
		"",
		IT_STAY_COOP | IT_KEY,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED key_commander_head (0 .5 .8) (-16 -16 -16) (16 16 16)
	   tank commander's head */
	{
		"key_commander_head",
		nil, // Pickup_Key,
		nil,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/monsters/commandr/head/tris.md2", shared.EF_GIB,
		"",
		"k_comhead",
		"Commander's Head",
		2,
		0,
		"",
		IT_STAY_COOP | IT_KEY,
		0,
		nil,
		0,
		"",
	},

	/* QUAKED key_airstrike_target (0 .5 .8) (-16 -16 -16) (16 16 16) */
	{
		"key_airstrike_target",
		nil, // Pickup_Key,
		nil,
		nil, // Drop_General,
		nil,
		"items/pkup.wav",
		"models/items/keys/target/tris.md2", shared.EF_ROTATE,
		"",
		"i_airstrike",
		"Airstrike Marker",
		2,
		0,
		"",
		IT_STAY_COOP | IT_KEY,
		0,
		nil,
		0,
		"",
	},

	{
		"",
		Pickup_Health,
		nil,
		nil,
		nil,
		"items/pkup.wav",
		"", 0,
		"",
		"i_health",
		"Health",
		3,
		0,
		"",
		0,
		0,
		nil,
		0,
		"items/s_health.wav items/n_health.wav items/l_health.wav items/m_health.wav",
	},
}

/*
 * QUAKED item_health (.3 .3 1) (-16 -16 -16) (16 16 16)
 */
func spItemHealth(self *edict_t, G *qGame) error {
	if self == nil || G == nil {
		return nil
	}

	if G.deathmatch.Bool() && (G.dmflags.Int()&shared.DF_NO_HEALTH) != 0 {
		G.gFreeEdict(self)
		return nil
	}

	self.Model = "models/items/healing/medium/tris.md2"
	self.count = 10
	G.spawnItem(self, G.findItem("Health"))
	G.gi.Soundindex("items/n_health.wav")
	return nil
}

/*
 * QUAKED item_health_small (.3 .3 1) (-16 -16 -16) (16 16 16)
 */
func spItemHealthSmall(self *edict_t, G *qGame) error {
	if self == nil || G == nil {
		return nil
	}

	if G.deathmatch.Bool() && (G.dmflags.Int()&shared.DF_NO_HEALTH) != 0 {
		G.gFreeEdict(self)
		return nil
	}

	self.Model = "models/items/healing/stimpack/tris.md2"
	self.count = 2
	G.spawnItem(self, G.findItem("Health"))
	// self.Style = HEALTH_IGNORE_MAX
	G.gi.Soundindex("items/s_health.wav")
	return nil
}

/*
 * QUAKED item_health_large (.3 .3 1) (-16 -16 -16) (16 16 16)
 */
func spItemHealthLarge(self *edict_t, G *qGame) error {
	if self == nil || G == nil {
		return nil
	}

	if G.deathmatch.Bool() && (G.dmflags.Int()&shared.DF_NO_HEALTH) != 0 {
		G.gFreeEdict(self)
		return nil
	}

	self.Model = "models/items/healing/large/tris.md2"
	self.count = 25
	G.spawnItem(self, G.findItem("Health"))
	G.gi.Soundindex("items/l_health.wav")
	return nil
}

/*
 * QUAKED item_health_mega (.3 .3 1) (-16 -16 -16) (16 16 16)
 */
func spItemHealthMega(self *edict_t, G *qGame) error {
	if self == nil || G == nil {
		return nil
	}

	if G.deathmatch.Bool() && (G.dmflags.Int()&shared.DF_NO_HEALTH) != 0 {
		G.gFreeEdict(self)
		return nil
	}

	self.Model = "models/items/mega_h/tris.md2"
	self.count = 100
	G.spawnItem(self, G.findItem("Health"))
	G.gi.Soundindex("items/m_health.wav")
	// self.Style = HEALTH_IGNORE_MAX | HEALTH_TIMED
	return nil
}
