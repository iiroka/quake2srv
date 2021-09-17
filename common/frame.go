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
 * Platform independent initialization, main loop and frame handling.
 *
 * =======================================================================
 */
package common

import (
	"log"
	"time"
)

func (Q *qCommon) execConfigs(gameStartUp bool) error {
	Q.Cbuf_AddText("exec default.cfg\n")
	Q.Cbuf_AddText("exec yq2.cfg\n")
	Q.Cbuf_AddText("exec config.cfg\n")
	Q.Cbuf_AddText("exec autoexec.cfg\n")

	if gameStartUp {
		/* Process cmd arguments only startup. */
		Q.cbuf_AddEarlyCommands(true)
	}

	return Q.Cbuf_Execute()
}

func (Q *qCommon) Init(args []string) error {
	Q.args = args
	Q.startTime = time.Now()
	// Jump point used in emergency situations.
	// 	if (setjmp(abortframe)) {
	// 		Sys_Error("Error during initialization");
	// 	}

	// 	if (checkForHelp(argc, argv))
	// 	{
	// 		// ok, --help or similar commandline option was given
	// 		// and info was printed, exit the game now
	// 		exit(1);
	// 	}

	// 	// Print the build and version string
	// 	Qcommon_Buildstring();

	// 	// Seed PRNG
	// 	randk_seed();

	// 	// Initialize zone malloc().
	// 	z_chain.next = z_chain.prev = &z_chain;

	// 	// Start early subsystems.
	// 	COM_InitArgv(argc, argv);
	// 	Swap_Init();
	// 	Cbuf_Init();
	Q.cmdInit()
	Q.cvar_Init()

	// #ifndef DEDICATED_ONLY
	// 	Key_Init();
	// #endif

	/* we need to add the early commands twice, because
	   a basedir or cddir needs to be set before execing
	   config files, but we want other parms to override
	   the settings of the config files */
	Q.cbuf_AddEarlyCommands(false)
	if err := Q.Cbuf_Execute(); err != nil {
		return err
	}

	// 	// remember the initial game name that might have been set on commandline
	// 	{
	// 		cvar_t* gameCvar = Cvar_Get("game", "", CVAR_LATCH | CVAR_SERVERINFO);
	// 		const char* game = "";

	// 		if(gameCvar->string && gameCvar->string[0])
	// 		{
	// 			game = gameCvar->string;
	// 		}

	// 		Q_strlcpy(userGivenGame, game, sizeof(userGivenGame));
	// 	}

	// The filesystems needs to be initialized after the cvars.
	// if err := Q.initFilesystem(); err != nil {
	// 	return err
	// }

	// Add and execute configuration files.
	if err := Q.execConfigs(true); err != nil {
		return err
	}

	// 	// Zone malloc statistics.
	// 	Cmd_AddCommand("z_stats", Z_Stats_f);

	// cvars

	// 	cl_maxfps = Cvar_Get("cl_maxfps", "60", CVAR_ARCHIVE);

	// 	developer = Cvar_Get("developer", "0", 0);
	// 	fixedtime = Cvar_Get("fixedtime", "0", 0);

	// 	logfile_active = Cvar_Get("logfile", "1", CVAR_ARCHIVE);
	// 	modder = Cvar_Get("modder", "0", 0);
	// 	timescale = Cvar_Get("timescale", "1", 0);

	// 	char *s;
	// 	s = va("%s %s %s %s", YQ2VERSION, YQ2ARCH, BUILD_DATE, YQ2OSTYPE);
	// 	Cvar_Get("version", s, CVAR_SERVERINFO | CVAR_NOSET);

	// 	// We can't use the clients "quit" command when running dedicated.
	// 	if (dedicated->value)
	// 	{
	// 		Cmd_AddCommand("quit", Com_Quit);
	// 	}

	// 	// Start late subsystem.
	// 	Sys_Init();
	// 	NET_Init();
	// 	Netchan_Init();
	Q.server.Init()

	// Everythings up, let's add + cmds from command line.
	if !Q.cbuf_AddLateCommands() {
		Q.Cbuf_AddText("dedicated_start\n")
		if err := Q.Cbuf_Execute(); err != nil {
			return err
		}
	}

	log.Printf("==== Yamagi Quake II Initialized ====\n\n")
	log.Printf("*************************************\n\n")

	// Call the main loop
	// 	Qcommon_Mainloop();
	Q.running = true
	return nil
}

func (Q *qCommon) MainLoop() error {
	// 	long long newtime;
	oldtime := time.Now()

	/* The mainloop. The legend. */
	for Q.running {
		// #ifndef DEDICATED_ONLY
		// 		// Throttle the game a little bit.
		// 		if (busywait->value)
		// 		{
		// 			long long spintime = Sys_Microseconds();

		// 			while (1)
		// 			{
		// 				/* Give the CPU a hint that this is a very tight
		// 				   spinloop. One PAUSE instruction each loop is
		// 				   enough to reduce power consumption and head
		// 				   dispersion a lot, it's 95°C against 67°C on
		// 				   a Kaby Lake laptop. */
		// #if defined (__GNUC__) && (__i386 || __x86_64__)
		// 				asm("pause");
		// #elif defined(__aarch64__) || (defined(__ARM_ARCH) && __ARM_ARCH >= 7) || defined(__ARM_ARCH_6K__)
		// 				asm("yield");
		// #endif

		// 				if (Sys_Microseconds() - spintime >= 5)
		// 				{
		// 					break;
		// 				}
		// 			}
		// 		}
		// 		else
		// 		{
		time.Sleep(5 * time.Microsecond)
		// 			Sys_Nanosleep(5000);
		// 		}
		// #else
		// 		Sys_Nanosleep(850000);
		// #endif

		newtime := time.Now()
		Q.frame(int(newtime.Sub(oldtime).Microseconds()))
		oldtime = newtime
	}
	log.Println("EXIT GAME")
	return nil
}

func (Q *qCommon) frame(usec int) error {
	// For the dedicated server terminal console.
	// char *s;

	// Target packetframerate.
	// int pfps;

	/* A packetframe runs the server and the client,
	   but not the renderer. The minimal interval of
	   packetframes is about 10.000 microsec. If run
	   more often the movement prediction in pmove.c
	   breaks. That's the Q2 variant if the famous
	   125hz bug. */
	packetframe := true

	/* Tells the client to shutdown.
	   Used by the signal handlers. */
	// if (quitnextframe) {
	// 	Cbuf_AddText("quit");
	// }

	/* In case of ERR_DROP we're jumping here. Don't know
	   if that' really save but it seems to work. So leave
	   it alone. */

	// Timing debug crap. Just for historical reasons.
	// if (fixedtime->value) {
	// 	usec = (int)fixedtime->value;
	// }
	// else if (timescale->value)
	// {
	// 	usec *= timescale->value;
	// }

	// Save global time for network- und input code.
	Q.curtime = Q.Sys_Milliseconds()

	// Target framerate.
	// pfps = (int)cl_maxfps->value;

	// Calculate timings.
	Q.packetdelta += usec
	Q.servertimedelta += usec

	// // Network frame time.
	// if (packetdelta < (1000000.0f / pfps)) {
	// 	packetframe = false;
	// }

	// // Dedicated server terminal console.
	// do {
	// 	s = Sys_ConsoleInput();

	// 	if (s) {
	// 		Cbuf_AddText(va("%s\n", s));
	// 	}
	// } while (s);

	if err := Q.Cbuf_Execute(); err != nil {
		return err
	}

	// // Run the serverframe.
	if packetframe {
		if err := Q.server.Frame(Q.servertimedelta); err != nil {
			return err
		}
		Q.servertimedelta = 0

		// Reset deltas if necessary.
		Q.packetdelta = 0
	}
	return nil
}

func (T *qCommon) SetServerState(state int) {
	T.server_state = state
}

func (T *qCommon) ServerState() int {
	return T.server_state
}

func (T *qCommon) Sys_Milliseconds() int {
	return int(time.Now().Sub(T.startTime).Milliseconds())
}

func (T *qCommon) Curtime() int {
	return T.curtime
}

func (T *qCommon) Quit() {
	T.running = false
}
