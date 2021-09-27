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
 * Soldier aka "Guard". This is the most complex enemy in Quake 2, since
 * it uses all AI features (dodging, sight, crouching, etc) and comes
 * in a myriad of variants.
 *
 * =======================================================================
 */
package game

import (
	"math"
	"quake2srv/game/soldier"
	"quake2srv/shared"
)

func soldier_idle(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	// if (random() > 0.8) {
	// 	gi.sound(self, CHAN_VOICE, sound_idle, 1, ATTN_IDLE, 0);
	// }
}

func soldier_cock(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	// if (self->s.frame == FRAME_stand322)
	// {
	// 	gi.sound(self, CHAN_WEAPON, sound_cock, 1, ATTN_IDLE, 0);
	// }
	// else
	// {
	// 	gi.sound(self, CHAN_WEAPON, sound_cock, 1, ATTN_NORM, 0);
	// }
}

var soldier_frames_stand1 = []mframe_t{
	{ai_stand, 0, soldier_idle},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},

	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},

	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil}}

var soldier_move_stand1 = mmove_t{
	soldier.FRAME_stand101,
	soldier.FRAME_stand130,
	soldier_frames_stand1,
	nil,
}

var soldier_frames_stand3 = []mframe_t{
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},

	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},

	{ai_stand, 0, nil},
	{ai_stand, 0, soldier_cock},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},

	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil},
	{ai_stand, 0, nil}}

var soldier_move_stand3 = mmove_t{
	soldier.FRAME_stand301,
	soldier.FRAME_stand339,
	soldier_frames_stand3,
	nil,
}

func soldier_stand(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if (self.monsterinfo.currentmove == &soldier_move_stand3) ||
		(shared.Frandk() < 0.8) {
		self.monsterinfo.currentmove = &soldier_move_stand1
	} else {
		self.monsterinfo.currentmove = &soldier_move_stand3
	}
}

func soldier_walk1_random(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if shared.Frandk() > 0.1 {
		self.monsterinfo.nextframe = soldier.FRAME_walk101
	}
}

var soldier_frames_walk1 = []mframe_t{
	{ai_walk, 3, nil},
	{ai_walk, 6, nil},
	{ai_walk, 2, nil},
	{ai_walk, 2, nil},
	{ai_walk, 2, nil},
	{ai_walk, 1, nil},
	{ai_walk, 6, nil},
	{ai_walk, 5, nil},
	{ai_walk, 3, nil},
	{ai_walk, -1, soldier_walk1_random},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
	{ai_walk, 0, nil},
}

var soldier_move_walk1 = mmove_t{
	soldier.FRAME_walk101,
	soldier.FRAME_walk133,
	soldier_frames_walk1,
	nil,
}

var soldier_frames_walk2 = []mframe_t{
	{ai_walk, 4, nil},
	{ai_walk, 4, nil},
	{ai_walk, 9, nil},
	{ai_walk, 8, nil},
	{ai_walk, 5, nil},
	{ai_walk, 1, nil},
	{ai_walk, 3, nil},
	{ai_walk, 7, nil},
	{ai_walk, 6, nil},
	{ai_walk, 7, nil},
}

var soldier_move_walk2 = mmove_t{
	soldier.FRAME_walk209,
	soldier.FRAME_walk218,
	soldier_frames_walk2,
	nil,
}

func soldier_walk(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if shared.Frandk() < 0.5 {
		self.monsterinfo.currentmove = &soldier_move_walk1
	} else {
		self.monsterinfo.currentmove = &soldier_move_walk2
	}
}

var soldier_frames_start_run = []mframe_t{
	{ai_run, 7, nil},
	{ai_run, 5, nil},
}

var soldier_move_start_run = mmove_t{
	soldier.FRAME_run01,
	soldier.FRAME_run02,
	soldier_frames_start_run,
	soldier_run,
}

var soldier_frames_run = []mframe_t{
	{ai_run, 10, nil},
	{ai_run, 11, nil},
	{ai_run, 11, nil},
	{ai_run, 16, nil},
	{ai_run, 10, nil},
	{ai_run, 15, nil},
}

var soldier_move_run = mmove_t{
	soldier.FRAME_run03,
	soldier.FRAME_run08,
	soldier_frames_run,
	nil,
}

var soldier_move_start_run_var *mmove_t
var soldier_move_run_var *mmove_t

func soldier_run(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if (self.monsterinfo.aiflags & AI_STAND_GROUND) != 0 {
		self.monsterinfo.currentmove = &soldier_move_stand1
		return
	}

	if (self.monsterinfo.currentmove == &soldier_move_walk1) ||
		(self.monsterinfo.currentmove == &soldier_move_walk2) ||
		(self.monsterinfo.currentmove == soldier_move_start_run_var) {
		self.monsterinfo.currentmove = soldier_move_run_var
	} else {
		self.monsterinfo.currentmove = soldier_move_start_run_var
	}
}

var soldier_frames_pain1 = []mframe_t{
	{ai_move, -3, nil},
	{ai_move, 4, nil},
	{ai_move, 1, nil},
	{ai_move, 1, nil},
	{ai_move, 0, nil},
}

var soldier_move_pain1 = mmove_t{
	soldier.FRAME_pain101,
	soldier.FRAME_pain105,
	soldier_frames_pain1,
	soldier_run,
}

var soldier_frames_pain2 = []mframe_t{
	{ai_move, -13, nil},
	{ai_move, -1, nil},
	{ai_move, 2, nil},
	{ai_move, 4, nil},
	{ai_move, 2, nil},
	{ai_move, 3, nil},
	{ai_move, 2, nil},
}

var soldier_move_pain2 = mmove_t{
	soldier.FRAME_pain201,
	soldier.FRAME_pain207,
	soldier_frames_pain2,
	soldier_run,
}

var soldier_frames_pain3 = []mframe_t{
	{ai_move, -8, nil},
	{ai_move, 10, nil},
	{ai_move, -4, nil},
	{ai_move, -1, nil},
	{ai_move, -3, nil},
	{ai_move, 0, nil},
	{ai_move, 3, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 1, nil},
	{ai_move, 0, nil},
	{ai_move, 1, nil},
	{ai_move, 2, nil},
	{ai_move, 4, nil},
	{ai_move, 3, nil},
	{ai_move, 2, nil},
}

var soldier_move_pain3 = mmove_t{
	soldier.FRAME_pain301,
	soldier.FRAME_pain318,
	soldier_frames_pain3,
	soldier_run,
}

var soldier_frames_pain4 = []mframe_t{
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, -10, nil},
	{ai_move, -6, nil},
	{ai_move, 8, nil},
	{ai_move, 4, nil},
	{ai_move, 1, nil},
	{ai_move, 0, nil},
	{ai_move, 2, nil},
	{ai_move, 5, nil},
	{ai_move, 2, nil},
	{ai_move, -1, nil},
	{ai_move, -1, nil},
	{ai_move, 3, nil},
	{ai_move, 2, nil},
	{ai_move, 0, nil},
}

var soldier_move_pain4 = mmove_t{
	soldier.FRAME_pain401,
	soldier.FRAME_pain417,
	soldier_frames_pain4,
	soldier_run,
}

func soldier_pain(self, other *edict_t, kick float32, damage int, G *qGame) {
	// float r;
	// int n;

	if self == nil || G == nil {
		return
	}

	if self.Health < (self.max_health / 2) {
		self.s.Skinnum |= 1
	}

	if G.level.time < self.pain_debounce_time {
		if (self.velocity[2] > 100) &&
			((self.monsterinfo.currentmove == &soldier_move_pain1) ||
				(self.monsterinfo.currentmove == &soldier_move_pain2) ||
				(self.monsterinfo.currentmove == &soldier_move_pain3)) {
			self.monsterinfo.currentmove = &soldier_move_pain4
		}

		return
	}

	self.pain_debounce_time = G.level.time + 3

	// n := self.s.skinnum | 1;

	// if (n == 1)
	// {
	// 	gi.sound(self, CHAN_VOICE, sound_pain_light, 1, ATTN_NORM, 0);
	// }
	// else if (n == 3)
	// {
	// 	gi.sound(self, CHAN_VOICE, sound_pain, 1, ATTN_NORM, 0);
	// }
	// else
	// {
	// 	gi.sound(self, CHAN_VOICE, sound_pain_ss, 1, ATTN_NORM, 0);
	// }

	if self.velocity[2] > 100 {
		self.monsterinfo.currentmove = &soldier_move_pain4
		return
	}

	if G.skill.Int() == SKILL_HARDPLUS {
		return /* no pain anims in nightmare */
	}

	r := shared.Frandk()

	if r < 0.33 {
		self.monsterinfo.currentmove = &soldier_move_pain1
	} else if r < 0.66 {
		self.monsterinfo.currentmove = &soldier_move_pain2
	} else {
		self.monsterinfo.currentmove = &soldier_move_pain3
	}
}

var blaster_flash = []int{
	shared.MZ2_SOLDIER_BLASTER_1,
	shared.MZ2_SOLDIER_BLASTER_2,
	shared.MZ2_SOLDIER_BLASTER_3,
	shared.MZ2_SOLDIER_BLASTER_4,
	shared.MZ2_SOLDIER_BLASTER_5,
	shared.MZ2_SOLDIER_BLASTER_6,
	shared.MZ2_SOLDIER_BLASTER_7,
	shared.MZ2_SOLDIER_BLASTER_8,
}

var shotgun_flash = []int{
	shared.MZ2_SOLDIER_SHOTGUN_1,
	shared.MZ2_SOLDIER_SHOTGUN_2,
	shared.MZ2_SOLDIER_SHOTGUN_3,
	shared.MZ2_SOLDIER_SHOTGUN_4,
	shared.MZ2_SOLDIER_SHOTGUN_5,
	shared.MZ2_SOLDIER_SHOTGUN_6,
	shared.MZ2_SOLDIER_SHOTGUN_7,
	shared.MZ2_SOLDIER_SHOTGUN_8,
}

var machinegun_flash = []int{
	shared.MZ2_SOLDIER_MACHINEGUN_1,
	shared.MZ2_SOLDIER_MACHINEGUN_2,
	shared.MZ2_SOLDIER_MACHINEGUN_3,
	shared.MZ2_SOLDIER_MACHINEGUN_4,
	shared.MZ2_SOLDIER_MACHINEGUN_5,
	shared.MZ2_SOLDIER_MACHINEGUN_6,
	shared.MZ2_SOLDIER_MACHINEGUN_7,
	shared.MZ2_SOLDIER_MACHINEGUN_8,
}

func (G *qGame) soldier_fire(self *edict_t, flash_number int) {
	// vec3_t start;
	// vec3_t forward, right, up;
	// vec3_t aim;
	// vec3_t dir;
	// vec3_t end;
	// float r, u;
	// int flash_index;

	if self == nil || G == nil {
		return
	}

	var flash_index int
	if self.s.Skinnum < 2 {
		flash_index = blaster_flash[flash_number]
	} else if self.s.Skinnum < 4 {
		flash_index = shotgun_flash[flash_number]
	} else {
		flash_index = machinegun_flash[flash_number]
	}

	forward := make([]float32, 3)
	right := make([]float32, 3)
	shared.AngleVectors(self.s.Angles[:], forward, right, nil)
	start := make([]float32, 3)
	gProjectSource(self.s.Origin[:], shared.MonsterFlashOffset[flash_index], forward, right, start)

	aim := make([]float32, 3)
	if (flash_number == 5) || (flash_number == 6) {
		copy(aim, forward)
	} else {
		end := make([]float32, 3)
		copy(end, self.enemy.s.Origin[:])
		end[2] += float32(self.enemy.viewheight)
		shared.VectorSubtract(end, start, aim)
		dir := make([]float32, 3)
		vectoangles(aim, dir)
		up := make([]float32, 3)
		shared.AngleVectors(dir, forward, right, up)

		r := shared.Crandk() * 1000
		u := shared.Crandk() * 500
		shared.VectorMA(start, 8192, forward, end)
		shared.VectorMA(end, r, right, end)
		shared.VectorMA(end, u, up, end)

		shared.VectorSubtract(end, start, aim)
		shared.VectorNormalize(aim)
	}

	if self.s.Skinnum <= 1 {
		G.monster_fire_blaster(self, start, aim, 5, 600, flash_index, shared.EF_BLASTER)
	} else if self.s.Skinnum <= 3 {
		G.monster_fire_shotgun(self, start, aim, 2, 1,
			DEFAULT_SHOTGUN_HSPREAD, DEFAULT_SHOTGUN_VSPREAD,
			DEFAULT_SHOTGUN_COUNT, flash_index)
	} else {
		println("monster_fire_bullet")
		// 	if ((self->monsterinfo.aiflags & AI_HOLD_FRAME) == 0) {
		// 		self->monsterinfo.pausetime = level.time + (3 + randk() % 8) * FRAMETIME;
		// 	}

		// 	monster_fire_bullet(self, start, aim, 2, 4,
		// 			DEFAULT_BULLET_HSPREAD, DEFAULT_BULLET_VSPREAD,
		// 			flash_index);

		// 	if (level.time >= self->monsterinfo.pausetime)
		// 	{
		// 		self->monsterinfo.aiflags &= ~AI_HOLD_FRAME;
		// 	}
		// 	else
		// 	{
		// 		self->monsterinfo.aiflags |= AI_HOLD_FRAME;
		// 	}
	}
}

/* ATTACK1 (blaster/shotgun) */
func soldier_fire1(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	G.soldier_fire(self, 0)
}

func soldier_attack1_refire1(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	if self.s.Skinnum > 1 {
		return
	}

	if self.enemy.Health <= 0 {
		return
	}

	if ((G.skill.Int() == SKILL_HARDPLUS) &&
		(shared.Frandk() < 0.5)) || (range_(self, self.enemy) == RANGE_MELEE) {
		self.monsterinfo.nextframe = soldier.FRAME_attak102
	} else {
		self.monsterinfo.nextframe = soldier.FRAME_attak110
	}
}

func soldier_attack1_refire2(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	if self.s.Skinnum < 2 {
		return
	}

	if self.enemy.Health <= 0 {
		return
	}

	if ((G.skill.Int() == SKILL_HARDPLUS) &&
		(shared.Frandk() < 0.5)) || (range_(self, self.enemy) == RANGE_MELEE) {
		self.monsterinfo.nextframe = soldier.FRAME_attak102
	}
}

var soldier_frames_attack1 = []mframe_t{
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_fire1},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_attack1_refire1},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_cock},
	{ai_charge, 0, soldier_attack1_refire2},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
}

var soldier_move_attack1 = mmove_t{
	soldier.FRAME_attak101,
	soldier.FRAME_attak112,
	soldier_frames_attack1,
	soldier_run,
}

/* ATTACK2 (blaster/shotgun) */
func soldier_fire2(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	G.soldier_fire(self, 1)
}

func soldier_attack2_refire1(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	if self.s.Skinnum > 1 {
		return
	}

	if self.enemy.Health <= 0 {
		return
	}

	if ((G.skill.Int() == SKILL_HARDPLUS) &&
		(shared.Frandk() < 0.5)) || (range_(self, self.enemy) == RANGE_MELEE) {
		self.monsterinfo.nextframe = soldier.FRAME_attak204
	} else {
		self.monsterinfo.nextframe = soldier.FRAME_attak216
	}
}

func soldier_attack2_refire2(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	if self.s.Skinnum < 2 {
		return
	}

	if self.enemy.Health <= 0 {
		return
	}

	if ((G.skill.Int() == SKILL_HARDPLUS) &&
		(shared.Frandk() < 0.5)) || (range_(self, self.enemy) == RANGE_MELEE) {
		self.monsterinfo.nextframe = soldier.FRAME_attak204
	}
}

var soldier_frames_attack2 = []mframe_t{
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_fire2},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_attack2_refire1},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_cock},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_attack2_refire2},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
}

var soldier_move_attack2 = mmove_t{
	soldier.FRAME_attak201,
	soldier.FRAME_attak218,
	soldier_frames_attack2,
	soldier_run,
}

/* ATTACK3 (duck and shoot) */
func (G *qGame) soldier_duck_down(self *edict_t) {

	if self == nil {
		return
	}

	if (self.monsterinfo.aiflags & AI_DUCKED) != 0 {
		return
	}

	self.monsterinfo.aiflags |= AI_DUCKED
	self.maxs[2] -= 32
	self.takedamage = DAMAGE_YES
	self.monsterinfo.pausetime = G.level.time + 1
	G.gi.Linkentity(self)
}

func soldier_duck_up(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	self.monsterinfo.aiflags &^= AI_DUCKED
	self.maxs[2] += 32
	self.takedamage = DAMAGE_AIM
	G.gi.Linkentity(self)
}

func soldier_fire3(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	G.soldier_duck_down(self)
	G.soldier_fire(self, 2)
}

func soldier_attack3_refire(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	if (G.level.time + 0.4) < self.monsterinfo.pausetime {
		self.monsterinfo.nextframe = soldier.FRAME_attak303
	}
}

var soldier_frames_attack3 = []mframe_t{
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_fire3},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_attack3_refire},
	{ai_charge, 0, soldier_duck_up},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
}

var soldier_move_attack3 = mmove_t{
	soldier.FRAME_attak301,
	soldier.FRAME_attak309,
	soldier_frames_attack3,
	soldier_run,
}

/* ATTACK4 (machinegun) */
func soldier_fire4(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	G.soldier_fire(self, 3)
}

var soldier_frames_attack4 = []mframe_t{
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, soldier_fire4},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
	{ai_charge, 0, nil},
}

var soldier_move_attack4 = mmove_t{
	soldier.FRAME_attak401,
	soldier.FRAME_attak406,
	soldier_frames_attack4,
	soldier_run,
}

/* ATTACK6 (run & shoot) */
func soldier_fire8(self *edict_t, G *qGame) {

	if self == nil || G == nil {
		return
	}

	G.soldier_fire(self, 7)
}

func soldier_attack6_refire(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if self.enemy.Health <= 0 {
		return
	}

	if range_(self, self.enemy) < RANGE_MID {
		return
	}

	if G.skill.Int() == SKILL_HARDPLUS {
		self.monsterinfo.nextframe = soldier.FRAME_runs03
	}
}

var soldier_frames_attack6 = []mframe_t{
	{ai_charge, 10, nil},
	{ai_charge, 4, nil},
	{ai_charge, 12, nil},
	{ai_charge, 11, soldier_fire8},
	{ai_charge, 13, nil},
	{ai_charge, 18, nil},
	{ai_charge, 15, nil},
	{ai_charge, 14, nil},
	{ai_charge, 11, nil},
	{ai_charge, 8, nil},
	{ai_charge, 11, nil},
	{ai_charge, 12, nil},
	{ai_charge, 12, nil},
	{ai_charge, 17, soldier_attack6_refire},
}

var soldier_move_attack6 = mmove_t{
	soldier.FRAME_runs01,
	soldier.FRAME_runs14,
	soldier_frames_attack6,
	soldier_run,
}

func soldier_attack(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	if self.s.Skinnum < 4 {
		if shared.Frandk() < 0.5 {
			self.monsterinfo.currentmove = &soldier_move_attack1
		} else {
			self.monsterinfo.currentmove = &soldier_move_attack2
		}
	} else {
		self.monsterinfo.currentmove = &soldier_move_attack4
	}
}

func soldier_sight(self, other *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	// if (random() < 0.5) {
	// 	gi.sound(self, CHAN_VOICE, sound_sight1, 1, ATTN_NORM, 0);
	// } else {
	// 	gi.sound(self, CHAN_VOICE, sound_sight2, 1, ATTN_NORM, 0);
	// }

	if (G.skill.Int() > SKILL_EASY) && (range_(self, self.enemy) >= RANGE_MID) {
		if shared.Frandk() > 0.5 {
			self.monsterinfo.currentmove = &soldier_move_attack6
		}
	}
}

func soldier_fire6(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	G.soldier_fire(self, 5)
}

func soldier_fire7(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	G.soldier_fire(self, 6)
}

func soldier_dead(self *edict_t, G *qGame) {
	if self == nil || G == nil {
		return
	}

	copy(self.mins[:], []float32{-16, -16, -24})
	copy(self.maxs[:], []float32{16, 16, -8})
	self.movetype = MOVETYPE_TOSS
	self.svflags |= shared.SVF_DEADMONSTER
	self.nextthink = 0
	G.gi.Linkentity(self)
}

var soldier_frames_death1 = []mframe_t{
	{ai_move, 0, nil},
	{ai_move, -10, nil},
	{ai_move, -10, nil},
	{ai_move, -10, nil},
	{ai_move, -5, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, soldier_fire6},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, soldier_fire7},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
}

var soldier_move_death1 = mmove_t{
	soldier.FRAME_death101,
	soldier.FRAME_death136,
	soldier_frames_death1,
	soldier_dead,
}

var soldier_frames_death2 = []mframe_t{
	{ai_move, -5, nil},
	{ai_move, -5, nil},
	{ai_move, -5, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
}

var soldier_move_death2 = mmove_t{
	soldier.FRAME_death201,
	soldier.FRAME_death235,
	soldier_frames_death2,
	soldier_dead,
}

var soldier_frames_death3 = []mframe_t{
	{ai_move, -5, nil},
	{ai_move, -5, nil},
	{ai_move, -5, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
}

var soldier_move_death3 = mmove_t{
	soldier.FRAME_death301,
	soldier.FRAME_death345,
	soldier_frames_death3,
	soldier_dead,
}

var soldier_frames_death4 = []mframe_t{
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
}

var soldier_move_death4 = mmove_t{
	soldier.FRAME_death401,
	soldier.FRAME_death453,
	soldier_frames_death4,
	soldier_dead,
}

var soldier_frames_death5 = []mframe_t{
	{ai_move, -5, nil},
	{ai_move, -5, nil},
	{ai_move, -5, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},

	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
}

var soldier_move_death5 = mmove_t{
	soldier.FRAME_death501,
	soldier.FRAME_death524,
	soldier_frames_death5,
	soldier_dead,
}

var soldier_frames_death6 = []mframe_t{
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
	{ai_move, 0, nil},
}

var soldier_move_death6 = mmove_t{
	soldier.FRAME_death601,
	soldier.FRAME_death610,
	soldier_frames_death6,
	soldier_dead,
}

func soldier_die(self, inflictor, attacker *edict_t, damage int, point []float32, G *qGame) {
	// int n;

	/* check for gib */
	// if (self.health <= self.gib_health) {
	// 	gi.sound(self, CHAN_VOICE, gi.soundindex("misc/udeath.wav"), 1, ATTN_NORM, 0);

	// 	for (n = 0; n < 3; n++) {
	// 		ThrowGib(self, "models/objects/gibs/sm_meat/tris.md2",
	// 				damage, GIB_ORGANIC);
	// 	}

	// 	ThrowGib(self, "models/objects/gibs/chest/tris.md2",
	// 			damage, GIB_ORGANIC);
	// 	ThrowHead(self, "models/objects/gibs/head2/tris.md2",
	// 			damage, GIB_ORGANIC);
	// 	self->deadflag = DEAD_DEAD;
	// 	return;
	// }

	if self.deadflag == DEAD_DEAD {
		return
	}

	/* regular death */
	self.deadflag = DEAD_DEAD
	self.takedamage = DAMAGE_YES
	self.s.Skinnum |= 1

	// if (self.s.skinnum == 1) {
	// 	gi.sound(self, CHAN_VOICE, sound_death_light, 1, ATTN_NORM, 0);
	// } else if (self.s.skinnum == 3) {
	// 	gi.sound(self, CHAN_VOICE, sound_death, 1, ATTN_NORM, 0);
	// } else {
	// 	gi.sound(self, CHAN_VOICE, sound_death_ss, 1, ATTN_NORM, 0);
	// }

	if math.Abs(float64((self.s.Origin[2]+float32(self.viewheight))-point[2])) <= 4 {
		/* head shot */
		self.monsterinfo.currentmove = &soldier_move_death3
		return
	}

	n := shared.Randk() % 5

	if n == 0 {
		self.monsterinfo.currentmove = &soldier_move_death1
	} else if n == 1 {
		self.monsterinfo.currentmove = &soldier_move_death2
	} else if n == 2 {
		self.monsterinfo.currentmove = &soldier_move_death4
	} else if n == 3 {
		self.monsterinfo.currentmove = &soldier_move_death5
	} else {
		self.monsterinfo.currentmove = &soldier_move_death6
	}
}

func (G *qGame) spMonsterSoldierX(self *edict_t) {
	if self == nil {
		return
	}

	soldier_move_start_run_var = &soldier_move_start_run
	soldier_move_run_var = &soldier_move_run

	soldier_move_stand1.endfunc = soldier_stand
	soldier_move_stand3.endfunc = soldier_stand

	self.s.Modelindex = G.gi.Modelindex("models/monsters/soldier/tris.md2")
	self.monsterinfo.scale = soldier.MODEL_SCALE
	self.mins = [3]float32{-16, -16, -24}
	self.maxs = [3]float32{16, 16, 32}
	self.movetype = MOVETYPE_STEP
	self.solid = shared.SOLID_BBOX

	// sound_idle = gi.soundindex("soldier/solidle1.wav");
	// sound_sight1 = gi.soundindex("soldier/solsght1.wav");
	// sound_sight2 = gi.soundindex("soldier/solsrch1.wav");
	// sound_cock = gi.soundindex("infantry/infatck3.wav");

	self.Mass = 100

	self.pain = soldier_pain
	self.die = soldier_die

	self.monsterinfo.stand = soldier_stand
	self.monsterinfo.walk = soldier_walk
	self.monsterinfo.run = soldier_run
	// self.monsterinfo.dodge = soldier_dodge
	self.monsterinfo.attack = soldier_attack
	self.monsterinfo.melee = nil
	self.monsterinfo.sight = soldier_sight

	G.gi.Linkentity(self)

	self.monsterinfo.stand(self, G)

	G.walkmonster_start(self)
}

/*
 * QUAKED monster_soldier (1 .5 0) (-16 -16 -24) (16 16 32) Ambush Trigger_Spawn Sight
 */
func spMonsterSoldier(self *edict_t, G *qGame) error {
	if self == nil {
		return nil
	}

	if G.deathmatch.Bool() {
		G.gFreeEdict(self)
		return nil
	}

	G.spMonsterSoldierX(self)

	// sound_pain = gi.soundindex("soldier/solpain1.wav");
	// sound_death = gi.soundindex("soldier/soldeth1.wav");
	G.gi.Soundindex("soldier/solatck1.wav")

	self.s.Skinnum = 2
	self.Health = 30
	self.gib_health = -30
	return nil
}
