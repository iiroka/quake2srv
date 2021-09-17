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
 * The Quake II CVAR subsystem. Implements dynamic variable handling.
 *
 * =======================================================================
 */
package common

import (
	"log"
	"quake2srv/shared"
	"strings"
)

func infoValidate(s string) bool {
	if strings.ContainsAny(s, "\\\";") {
		return false
	}
	return true
}

func (Q *qCommon) cvarFindVar(name string) *shared.CvarT {
	// cvar_t *var;
	// int i;

	// /* An ugly hack to rewrite changed CVARs */
	// for (i = 0; i < sizeof(replacements) / sizeof(replacement_t); i++)
	// {
	// 	if (!strcmp(var_name, replacements[i].old))
	// 	{
	// 		Com_Printf("cvar %s ist deprecated, use %s instead\n", replacements[i].old, replacements[i].new);

	// 		var_name = replacements[i].new;
	// 	}
	// }

	if v, ok := Q.cvarVars[name]; ok {
		return v
	}
	return nil
}

func (T *qCommon) Cvar_VariableBool(var_name string) bool {
	v := T.cvarFindVar(var_name)
	if v == nil {
		return false
	}
	return v.Bool()
}

func (T *qCommon) Cvar_VariableInt(var_name string) int {
	v := T.cvarFindVar(var_name)
	if v == nil {
		return 0
	}
	return v.Int()
}

func (T *qCommon) Cvar_VariableString(var_name string) string {
	v := T.cvarFindVar(var_name)
	if v == nil {
		return ""
	}
	return v.String
}

/*
 * If the variable already exists, the value will not be set
 * The flags will be or'ed in if the variable exists.
 */
func (Q *qCommon) Cvar_Get(var_name, var_value string, flags int) *shared.CvarT {
	// cvar_t *var;
	// cvar_t **pos;

	if (flags & (shared.CVAR_USERINFO | shared.CVAR_SERVERINFO)) != 0 {
		if !infoValidate(var_name) {
			log.Printf("invalid info cvar name\n")
			return nil
		}
	}

	v := Q.cvarFindVar(var_name)
	if v != nil {
		v.Flags |= flags

		// 	if (!var_value) {
		// 		var->default_string = CopyString("");
		// 	} else {
		v.DefaultString = string(var_value)
		// 	}

		return v
	}

	// if (!var_value)
	// {
	// 	return NULL;
	// }

	if (flags & (shared.CVAR_USERINFO | shared.CVAR_SERVERINFO)) != 0 {
		if !infoValidate(var_value) {
			log.Printf("invalid info cvar value\n")
			return nil
		}
	}

	// // if $game is the default one ("baseq2"), then use "" instead because
	// // other code assumes this behavior (e.g. FS_BuildGameSpecificSearchPath())
	// if(strcmp(var_name, "game") == 0 && strcmp(var_value, BASEDIRNAME) == 0)
	// {
	// 	var_value = "";
	// }

	v = &shared.CvarT{}
	v.Name = string(var_name)
	v.String = string(var_value)
	v.DefaultString = string(var_value)
	v.Modified = true
	v.Flags = flags

	Q.cvarVars[var_name] = v

	return v
}

func (T *qCommon) Cvar_Set2(var_name, value string, force bool) *shared.CvarT {

	v := T.cvarFindVar(var_name)
	if v == nil {
		return T.Cvar_Get(var_name, value, 0)
	}

	if (v.Flags & (shared.CVAR_USERINFO | shared.CVAR_SERVERINFO)) != 0 {
		if !infoValidate(value) {
			log.Printf("invalid info cvar value\n")
			return v
		}
	}

	// if $game is the default one ("baseq2"), then use "" instead because
	// other code assumes this behavior (e.g. FS_BuildGameSpecificSearchPath())
	// if(strcmp(var_name, "game") == 0 && strcmp(value, BASEDIRNAME) == 0) {
	// 	value = "";
	// }

	if !force {
		if (v.Flags & shared.CVAR_NOSET) != 0 {
			log.Printf("%s is write protected.\n", var_name)
			return v
		}

		if (v.Flags & shared.CVAR_LATCH) != 0 {
			if v.LatchedString != nil {
				if value == *v.LatchedString {
					return v
				}

				v.LatchedString = nil
			} else {
				if value == v.String {
					return v
				}
			}

			if T.ServerState() != 0 {
				log.Printf("%v will be changed for next game.\n", var_name)
				v.LatchedString = &value
			} else {
				v.String = string(value)

				// if (!strcmp(var->name, "game")) {
				// 	FS_BuildGameSpecificSearchPath(var->string);
				// }
			}

			return v
		}
	} else {
		v.LatchedString = nil
	}

	if value == v.String {
		return v
	}

	v.Modified = true

	if (v.Flags & shared.CVAR_USERINFO) != 0 {
		T.userinfoModified = true
	}

	v.String = string(value)

	return v
}

func (T *qCommon) Cvar_ForceSet(var_name, value string) *shared.CvarT {
	return T.Cvar_Set2(var_name, value, true)
}

func (T *qCommon) Cvar_Set(var_name, value string) *shared.CvarT {
	return T.Cvar_Set2(var_name, value, false)
}

func (T *qCommon) Cvar_FullSet(var_name, value string, flags int) *shared.CvarT {

	v := T.cvarFindVar(var_name)
	if v == nil {
		return T.Cvar_Get(var_name, value, flags)
	}

	v.Modified = true

	if (v.Flags & shared.CVAR_USERINFO) != 0 {
		T.userinfoModified = true
	}

	// if $game is the default one ("baseq2"), then use "" instead because
	// other code assumes this behavior (e.g. FS_BuildGameSpecificSearchPath())
	// if(strcmp(var_name, "game") == 0 && strcmp(value, BASEDIRNAME) == 0)
	// {
	// 	value = "";
	// }

	v.String = string(value)
	v.Flags = flags

	return v
}

/*
 * Handles variable inspection and changing from the console
 */
func (T *qCommon) cvar_Command(args []string) bool {

	/* check variables */
	v := T.cvarFindVar(args[0])
	if v == nil {
		return false
	}

	/* perform a variable print or set */
	if len(args) == 1 {
		log.Printf("\"%s\" is \"%s\"\n", v.Name, v.String)
		return true
	}

	/* Another evil hack: The user has just changed 'game' trough
	the console. We reset userGivenGame to that value, otherwise
	we would revert to the initialy given game at disconnect. */
	//  if (strcmp(v->name, "game") == 0)
	//  {
	// 	 Q_strlcpy(userGivenGame, Cmd_Argv(1), sizeof(userGivenGame));
	//  }

	T.Cvar_Set(v.Name, args[1])
	return true
}

/*
 * Allows setting and defining of arbitrary cvars from console
 */
func cvar_Set_f(args []string, arg interface{}) error {
	//  char *firstarg;
	//  int c, i;
	T := arg.(*qCommon)

	//  c = Cmd_Argc();

	if (len(args) != 3) && (len(args) != 4) {
		log.Printf("usage: set <variable> <value> [u / s]\n")
		return nil
	}

	firstarg := args[1]

	//  /* An ugly hack to rewrite changed CVARs */
	//  for (i = 0; i < sizeof(replacements) / sizeof(replacement_t); i++)
	//  {
	// 	 if (!strcmp(firstarg, replacements[i].old))
	// 	 {
	// 		 firstarg = replacements[i].new;
	// 	 }
	//  }

	if len(args) == 4 {
		flags := 0

		if args[3] == "u" {
			flags = shared.CVAR_USERINFO
		} else if args[3] == "s" {
			flags = shared.CVAR_SERVERINFO
		} else {
			log.Printf("flags can only be 'u' or 's'\n")
			return nil
		}

		T.Cvar_FullSet(firstarg, args[2], flags)
	} else {
		T.Cvar_Set(firstarg, args[2])
	}
	return nil
}

/*
 * Reads in all archived cvars
 */
func (T *qCommon) cvar_Init() {
	//  Cmd_AddCommand("cvarlist", Cvar_List_f);
	//  Cmd_AddCommand("dec", Cvar_Inc_f);
	//  Cmd_AddCommand("inc", Cvar_Inc_f);
	//  Cmd_AddCommand("reset", Cvar_Reset_f);
	//  Cmd_AddCommand("resetall", Cvar_ResetAll_f);
	T.Cmd_AddCommand("set", cvar_Set_f, T)
	//  Cmd_AddCommand("toggle", Cvar_Toggle_f);
}
