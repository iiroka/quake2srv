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
 * Server entity handling. Just encodes the entties of a client side
 * frame into network / local communication packages and sends them to
 * the appropriate clients.
 *
 * =======================================================================
 */
package server

import (
	"log"
	"quake2srv/shared"
)

/*
 * Writes a delta update of an entity_state_t list to the message.
 */
func (T *qServer) svEmitPacketEntities(from, to *client_frame_t, msg *shared.QWritebuf) {
	msg.WriteByte(shared.SvcPacketentities)

	var from_num_entities int
	if from == nil {
		from_num_entities = 0
	} else {
		from_num_entities = from.num_entities
	}

	newindex := 0
	oldindex := 0
	var newent *shared.Entity_state_t
	var oldent *shared.Entity_state_t

	for newindex < to.num_entities || oldindex < from_num_entities {
		if msg.Cursize > shared.MAX_MSGLEN-150 {
			break
		}

		var newnum int
		if newindex >= to.num_entities {
			newnum = 9999
		} else {
			newent = &T.svs.client_entities[(to.first_entity+
				newindex)%T.svs.num_client_entities]
			newnum = newent.Number
		}

		var oldnum int
		if oldindex >= from_num_entities {
			oldnum = 9999
		} else {
			oldent = &T.svs.client_entities[(from.first_entity+
				oldindex)%T.svs.num_client_entities]
			oldnum = oldent.Number
		}

		if newnum == oldnum {
			/* delta update from old position. because the force
			parm is false, this will not result in any bytes
			being emited if the entity has not changed at all
			note that players are always 'newentities', this
			updates their oldorigin always and prevents warping */
			msg.WriteDeltaEntity(oldent, newent,
				false, newent.Number <= T.maxclients.Int())
			oldindex++
			newindex++
			continue
		}

		if newnum < oldnum {
			/* this is a new entity, send it from the baseline */
			msg.WriteDeltaEntity(&T.sv.baselines[newnum], newent, true, true)
			newindex++
			continue
		}

		if newnum > oldnum {
			/* the old entity isn't present in the new message */
			bits := shared.U_REMOVE

			if oldnum >= 256 {
				bits |= shared.U_NUMBER16 | shared.U_MOREBITS1
			}

			msg.WriteByte(bits & 255)

			if (bits & 0x0000ff00) != 0 {
				msg.WriteByte((bits >> 8) & 255)
			}

			if (bits & shared.U_NUMBER16) != 0 {
				msg.WriteShort(oldnum)
			} else {
				msg.WriteByte(oldnum)
			}

			oldindex++
			continue
		}
	}

	msg.WriteShort(0)
}

func (T *qServer) svWritePlayerstateToClient(from, to *client_frame_t,
	msg *shared.QWritebuf) {

	ps := &to.ps

	var ops *shared.Player_state_t
	if from == nil {
		ops = &shared.Player_state_t{}
	} else {
		ops = &from.ps
	}

	/* determine what needs to be sent */
	pflags := 0

	if ps.Pmove.Pm_type != ops.Pmove.Pm_type {
		pflags |= shared.PS_M_TYPE
	}

	if (ps.Pmove.Origin[0] != ops.Pmove.Origin[0]) ||
		(ps.Pmove.Origin[1] != ops.Pmove.Origin[1]) ||
		(ps.Pmove.Origin[2] != ops.Pmove.Origin[2]) {
		pflags |= shared.PS_M_ORIGIN
	}

	if (ps.Pmove.Velocity[0] != ops.Pmove.Velocity[0]) ||
		(ps.Pmove.Velocity[1] != ops.Pmove.Velocity[1]) ||
		(ps.Pmove.Velocity[2] != ops.Pmove.Velocity[2]) {
		pflags |= shared.PS_M_VELOCITY
	}

	if ps.Pmove.Pm_time != ops.Pmove.Pm_time {
		pflags |= shared.PS_M_TIME
	}

	if ps.Pmove.Pm_flags != ops.Pmove.Pm_flags {
		pflags |= shared.PS_M_FLAGS
	}

	if ps.Pmove.Gravity != ops.Pmove.Gravity {
		pflags |= shared.PS_M_GRAVITY
	}

	if (ps.Pmove.Delta_angles[0] != ops.Pmove.Delta_angles[0]) ||
		(ps.Pmove.Delta_angles[1] != ops.Pmove.Delta_angles[1]) ||
		(ps.Pmove.Delta_angles[2] != ops.Pmove.Delta_angles[2]) {
		pflags |= shared.PS_M_DELTA_ANGLES
	}

	if (ps.Viewoffset[0] != ops.Viewoffset[0]) ||
		(ps.Viewoffset[1] != ops.Viewoffset[1]) ||
		(ps.Viewoffset[2] != ops.Viewoffset[2]) {
		pflags |= shared.PS_VIEWOFFSET
	}

	if (ps.Viewangles[0] != ops.Viewangles[0]) ||
		(ps.Viewangles[1] != ops.Viewangles[1]) ||
		(ps.Viewangles[2] != ops.Viewangles[2]) {
		pflags |= shared.PS_VIEWANGLES
	}

	if (ps.Kick_angles[0] != ops.Kick_angles[0]) ||
		(ps.Kick_angles[1] != ops.Kick_angles[1]) ||
		(ps.Kick_angles[2] != ops.Kick_angles[2]) {
		pflags |= shared.PS_KICKANGLES
	}

	if (ps.Blend[0] != ops.Blend[0]) ||
		(ps.Blend[1] != ops.Blend[1]) ||
		(ps.Blend[2] != ops.Blend[2]) ||
		(ps.Blend[3] != ops.Blend[3]) {
		pflags |= shared.PS_BLEND
	}

	if ps.Fov != ops.Fov {
		pflags |= shared.PS_FOV
	}

	if ps.Rdflags != ops.Rdflags {
		pflags |= shared.PS_RDFLAGS
	}

	if ps.Gunframe != ops.Gunframe {
		pflags |= shared.PS_WEAPONFRAME
	}

	pflags |= shared.PS_WEAPONINDEX

	/* write it */
	msg.WriteByte(shared.SvcPlayerinfo)
	msg.WriteShort(pflags)

	/* write the pmove_state_t */
	if (pflags & shared.PS_M_TYPE) != 0 {
		msg.WriteByte(int(ps.Pmove.Pm_type) & 0xFF)
	}

	if (pflags & shared.PS_M_ORIGIN) != 0 {
		msg.WriteShort(int(ps.Pmove.Origin[0]))
		msg.WriteShort(int(ps.Pmove.Origin[1]))
		msg.WriteShort(int(ps.Pmove.Origin[2]))
	}

	if (pflags & shared.PS_M_VELOCITY) != 0 {
		msg.WriteShort(int(ps.Pmove.Velocity[0]))
		msg.WriteShort(int(ps.Pmove.Velocity[1]))
		msg.WriteShort(int(ps.Pmove.Velocity[2]))
	}

	if (pflags & shared.PS_M_TIME) != 0 {
		msg.WriteByte(int(ps.Pmove.Pm_time))
	}

	if (pflags & shared.PS_M_FLAGS) != 0 {
		msg.WriteByte(int(ps.Pmove.Pm_flags))
	}

	if (pflags & shared.PS_M_GRAVITY) != 0 {
		msg.WriteShort(int(ps.Pmove.Gravity))
	}

	if (pflags & shared.PS_M_DELTA_ANGLES) != 0 {
		msg.WriteShort(int(ps.Pmove.Delta_angles[0]))
		msg.WriteShort(int(ps.Pmove.Delta_angles[1]))
		msg.WriteShort(int(ps.Pmove.Delta_angles[2]))
	}

	/* write the rest of the player_state_t */
	if (pflags & shared.PS_VIEWOFFSET) != 0 {
		msg.WriteChar(int(ps.Viewoffset[0] * 4))
		msg.WriteChar(int(ps.Viewoffset[1] * 4))
		msg.WriteChar(int(ps.Viewoffset[2] * 4))
	}

	if (pflags & shared.PS_VIEWANGLES) != 0 {
		msg.WriteAngle16(ps.Viewangles[0])
		msg.WriteAngle16(ps.Viewangles[1])
		msg.WriteAngle16(ps.Viewangles[2])
	}

	if (pflags & shared.PS_KICKANGLES) != 0 {
		msg.WriteChar(int(ps.Kick_angles[0] * 4))
		msg.WriteChar(int(ps.Kick_angles[1] * 4))
		msg.WriteChar(int(ps.Kick_angles[2] * 4))
	}

	if (pflags & shared.PS_WEAPONINDEX) != 0 {
		msg.WriteByte(ps.Gunindex)
	}

	if (pflags & shared.PS_WEAPONFRAME) != 0 {
		msg.WriteByte(ps.Gunframe)
		msg.WriteChar(int(ps.Gunoffset[0] * 4))
		msg.WriteChar(int(ps.Gunoffset[1] * 4))
		msg.WriteChar(int(ps.Gunoffset[2] * 4))
		msg.WriteChar(int(ps.Gunangles[0] * 4))
		msg.WriteChar(int(ps.Gunangles[1] * 4))
		msg.WriteChar(int(ps.Gunangles[2] * 4))
	}

	if (pflags & shared.PS_BLEND) != 0 {
		msg.WriteByte(int(ps.Blend[0] * 255))
		msg.WriteByte(int(ps.Blend[1] * 255))
		msg.WriteByte(int(ps.Blend[2] * 255))
		msg.WriteByte(int(ps.Blend[3] * 255))
	}

	if (pflags & shared.PS_FOV) != 0 {
		msg.WriteByte(int(ps.Fov) & 0xFF)
	}

	if (pflags & shared.PS_RDFLAGS) != 0 {
		msg.WriteByte(ps.Rdflags)
	}

	/* send stats */
	statbits := 0

	for i := 0; i < shared.MAX_STATS; i++ {
		if ps.Stats[i] != ops.Stats[i] {
			statbits |= 1 << i
		}
	}

	msg.WriteLong(statbits)

	for i := 0; i < shared.MAX_STATS; i++ {
		if (statbits & (1 << i)) != 0 {
			msg.WriteShort(int(ps.Stats[i]))
		}
	}
}

func (T *qServer) svWriteFrameToClient(client *client_t, msg *shared.QWritebuf) {
	// client_frame_t *frame, *oldframe;
	// int lastframe;

	/* this is the frame we are creating */
	frame := &client.frames[T.sv.framenum&shared.UPDATE_MASK]

	var oldframe *client_frame_t
	var lastframe int
	if client.lastframe <= 0 {
		/* client is asking for a retransmit */
		oldframe = nil
		lastframe = -1
	} else if T.sv.framenum-client.lastframe >= (shared.UPDATE_BACKUP - 3) {
		/* client hasn't gotten a good message through in a long time */
		oldframe = nil
		lastframe = -1
	} else {
		// 	/* we have a valid message to delta from */
		oldframe = &client.frames[client.lastframe&shared.UPDATE_MASK]
		lastframe = client.lastframe
	}

	msg.WriteByte(shared.SvcFrame)
	msg.WriteLong(T.sv.framenum)
	msg.WriteLong(lastframe)            /* what we are delta'ing from */
	msg.WriteByte(client.surpressCount) /* rate dropped packets */
	client.surpressCount = 0

	/* send over the areabits */
	msg.WriteByte(frame.areabytes)
	msg.Write(frame.areabits[:frame.areabytes])

	/* delta encode the playerstate */
	T.svWritePlayerstateToClient(oldframe, frame, msg)

	/* delta encode the entities */
	T.svEmitPacketEntities(oldframe, frame, msg)
}

/*
 * The client will interpolate the view position,
 * so we can't use a single PVS point
 */
func (T *qServer) svFatPVS(org []float32) {

	var mins [3]float32
	var maxs [3]float32
	for i := 0; i < 3; i++ {
		mins[i] = org[i] - 8
		maxs[i] = org[i] + 8
	}

	var leafs [64]int
	count := T.common.CMBoxLeafnums(mins[:], maxs[:], leafs[:], 64, nil)

	if count < 1 {
		T.common.Com_Error(shared.ERR_FATAL, "SV_FatPVS: count < 1")
		return
	}

	numBytes := (T.common.CMNumClusters() + 7) / 8

	/* convert leafs to clusters */
	for i := 0; i < count; i++ {
		leafs[i] = T.common.CMLeafCluster(leafs[i])
	}
	copy(T.fatpvs[:numBytes], T.common.CMClusterPVS(leafs[0]))

	/* or in all the other leaf bits */
	for i := 1; i < count; i++ {
		found := false
		for j := 0; j < i; j++ {
			if leafs[i] == leafs[j] {
				found = true
				break
			}
		}

		if found {
			continue /* already have the cluster we want */
		}

		src := T.common.CMClusterPVS(leafs[i])

		for j := 0; j < numBytes; j++ {
			T.fatpvs[j] |= src[j]
		}
	}
}

/*
 * Decides which entities are going to be visible to the client, and
 * copies off the playerstat and areabits.
 */
func (T *qServer) svBuildClientFrame(client *client_t) {

	clent := client.edict

	if clent.Client() == nil {
		return /* not in game yet */
	}

	/* this is the frame we are creating */
	frame := &client.frames[T.sv.framenum&shared.UPDATE_MASK]

	frame.senttime = T.svs.realtime /* save it for ping calc later */

	/* find the client's PVS */
	var org [3]float32
	for i := 0; i < 3; i++ {
		org[i] = float32(clent.Client().Ps().Pmove.Origin[i])*0.125 +
			clent.Client().Ps().Viewoffset[i]
	}

	leafnum := T.common.CMPointLeafnum(org[:])
	clientarea := T.common.CMLeafArea(leafnum)
	clientcluster := T.common.CMLeafCluster(leafnum)

	/* calculate the visible areas */
	frame.areabytes = T.common.CMWriteAreaBits(frame.areabits[:], clientarea)

	/* grab the current player_state_t */
	frame.ps.Copy(*clent.Client().Ps())

	T.svFatPVS(org[:])
	clientphs := T.common.CMClusterPHS(clientcluster)

	/* build up the list of visible entities */
	frame.num_entities = 0
	frame.first_entity = T.svs.next_client_entities

	//  c_fullsend = 0;

	for e := 1; e < T.ge.NumEdicts(); e++ {
		ent := T.ge.Edict(e)

		/* ignore ents without visible models */
		if (ent.Svflags() & shared.SVF_NOCLIENT) != 0 {
			continue
		}

		/* ignore ents without visible models unless they have an effect */
		if ent.S().Modelindex == 0 && ent.S().Effects == 0 &&
			ent.S().Sound == 0 && ent.S().Event == 0 {
			continue
		}

		/* ignore if not touching a PV leaf */
		if ent != clent {
			/* check area */
			if !T.common.CMAreasConnected(clientarea, ent.Areanum()) {
				/* doors can legally straddle two areas,
				so we may need to check another one */
				if ent.Areanum2() == 0 ||
					!T.common.CMAreasConnected(clientarea, ent.Areanum()) {
					continue /* blocked by a door */
				}
			}

			/* beams just check one point for PHS */
			if (ent.S().Renderfx & shared.RF_BEAM) != 0 {
				l := ent.Clusternums()[0]

				if (clientphs[l>>3] & (1 << (l & 7))) == 0 {
					continue
				}
			} else {
				bitvector := T.fatpvs[:]

				if ent.NumClusters() == -1 {
					/* too many leafs for individual check, go by headnode */
					if !T.common.CMHeadnodeVisible(ent.Headnode(), bitvector) {
						continue
					}

					//  c_fullsend++;
				} else {
					/* check individual leafs */
					found := false
					for i := 0; i < ent.NumClusters(); i++ {
						l := ent.Clusternums()[i]
						if (bitvector[l>>3] & (1 << (l & 7))) != 0 {
							found = true
							break
						}
					}

					if !found {
						continue /* not visible */
					}
				}

				if ent.S().Modelindex == 0 {
					/* don't send sounds if they
					will be attenuated away */

					delta := make([]float32, 3)
					shared.VectorSubtract(org[:], ent.S().Origin[:], delta)
					len := shared.VectorLength(delta)

					if len > 400 {
						continue
					}
				}
			}
		}

		/* add it to the circular client_entities array */
		state := &T.svs.client_entities[T.svs.next_client_entities%T.svs.num_client_entities]

		if ent.S().Number != e {
			log.Printf("FIXING ENT->S.NUMBER!!!\n")
			ent.S().Number = e
		}

		state.Copy(*ent.S())

		/* don't mark players missiles as solid */
		if ent.Owner() == client.edict {
			state.Solid = 0
		}

		T.svs.next_client_entities++
		frame.num_entities++
	}
}
