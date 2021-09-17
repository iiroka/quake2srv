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
 * Interface to the world model. Clipping and stuff like that...
 *
 * =======================================================================
 */
package server

import (
	"log"
	"math"
	"quake2srv/shared"
)

const AREA_DEPTH = 4
const AREA_NODES = 32
const MAX_TOTAL_ENT_LEAFS = 128

type areanode_t struct {
	axis           int /* -1 = leaf node */
	dist           float32
	children       [2]*areanode_t
	trigger_edicts shared.Link_t
	solid_edicts   shared.Link_t
}

/* ClearLink is used for new headnodes */
func ClearLink(l *shared.Link_t) {
	l.Prev = l
	l.Next = l
}

func RemoveLink(l *shared.Link_t) {
	l.Next.Prev = l.Prev
	l.Prev.Next = l.Next
}

func InsertLinkBefore(l, before *shared.Link_t) {
	l.Next = before
	l.Prev = before.Prev
	l.Prev.Next = l
	l.Next.Prev = l
}

/*
 * Builds a uniformly subdivided tree for the given world size
 */
func (T *qServer) svCreateAreaNode(depth int, mins, maxs []float32) *areanode_t {

	anode := &T.sv_areanodes[T.sv_numareanodes]
	T.sv_numareanodes++

	ClearLink(&anode.trigger_edicts)
	ClearLink(&anode.solid_edicts)

	if depth == AREA_DEPTH {
		anode.axis = -1
		anode.children[0] = nil
		anode.children[1] = nil
		return anode
	}

	var size [3]float32
	shared.VectorSubtract(maxs, mins, size[:])

	if size[0] > size[1] {
		anode.axis = 0
	} else {
		anode.axis = 1
	}

	anode.dist = 0.5 * (maxs[anode.axis] + mins[anode.axis])
	var mins1 [3]float32
	var mins2 [3]float32
	var maxs1 [3]float32
	var maxs2 [3]float32
	copy(mins1[:], mins)
	copy(mins2[:], mins)
	copy(maxs1[:], maxs)
	copy(maxs2[:], maxs)

	maxs1[anode.axis] = anode.dist
	mins2[anode.axis] = anode.dist

	anode.children[0] = T.svCreateAreaNode(depth+1, mins2[:], maxs2[:])
	anode.children[1] = T.svCreateAreaNode(depth+1, mins1[:], maxs1[:])

	return anode
}

func (T *qServer) svClearWorld() {
	// memset(sv_areanodes, 0, sizeof(sv_areanodes));
	T.sv_numareanodes = 0
	T.svCreateAreaNode(0, T.sv.models[1].Mins[:], T.sv.models[1].Maxs[:])
}

func (T *qServer) svUnlinkEdict(ent shared.Edict_s) {
	if ent.Area().Prev == nil {
		return /* not linked in anywhere */
	}

	RemoveLink(ent.Area())
	ent.Area().Prev = nil
	ent.Area().Next = nil
}

func (T *qServer) svLinkEdict(ent shared.Edict_s) {

	if ent.Area().Prev != nil {
		T.svUnlinkEdict(ent) /* unlink from old position */
	}

	if ent == T.ge.Edict(0) {
		return /* don't add the world */
	}

	if !ent.Inuse() {
		return
	}

	/* set the size */
	shared.VectorSubtract(ent.Maxs(), ent.Mins(), ent.Size())

	/* encode the size into the entity_state for client prediction */
	if (ent.Solid() == shared.SOLID_BBOX) && (ent.Svflags()&shared.SVF_DEADMONSTER) == 0 {
		/* assume that x/y are equal and symetric */
		i := int(ent.Maxs()[0] / 8)

		if i < 1 {
			i = 1
		}

		if i > 31 {
			i = 31
		}

		/* z is not symetric */
		j := int(-ent.Mins()[2]) / 8

		if j < 1 {
			j = 1
		}

		if j > 31 {
			j = 31
		}

		/* and z maxs can be negative... */
		k := int(ent.Maxs()[2]+32) / 8

		if k < 1 {
			k = 1
		}

		if k > 63 {
			k = 63
		}

		ent.S().Solid = (k << 10) | (j << 5) | i
	} else if ent.Solid() == shared.SOLID_BSP {
		ent.S().Solid = 31 /* a solid_bbox will never create this value */
	} else {
		ent.S().Solid = 0
	}

	// /* set the abs box */
	if (ent.Solid() == shared.SOLID_BSP) &&
		(ent.S().Angles[0] != 0 || ent.S().Angles[1] != 0 ||
			ent.S().Angles[2] != 0) {
		/* expand for rotation */

		var max float32 = 0

		for i := 0; i < 3; i++ {
			v := float32(math.Abs(float64(ent.Mins()[i])))

			if v > max {
				max = v
			}

			v = float32(math.Abs(float64(ent.Maxs()[i])))

			if v > max {
				max = v
			}
		}

		for i := 0; i < 3; i++ {
			ent.Absmin()[i] = ent.S().Origin[i] - max
			ent.Absmax()[i] = ent.S().Origin[i] + max
		}
	} else {
		/* normal */
		shared.VectorAdd(ent.S().Origin[:], ent.Mins(), ent.Absmin())
		shared.VectorAdd(ent.S().Origin[:], ent.Maxs(), ent.Absmax())
	}

	/* because movement is clipped an epsilon away from an actual edge,
	we must fully check even when bounding boxes don't quite touch */
	ent.Absmin()[0] -= 1
	ent.Absmin()[1] -= 1
	ent.Absmin()[2] -= 1
	ent.Absmax()[0] += 1
	ent.Absmax()[1] += 1
	ent.Absmax()[2] += 1

	/* link to PVS leafs */
	ent.SetNumClusters(0)
	ent.SetAreanum(0)
	ent.SetAreanum2(0)

	/* get all leafs, including solids */
	var leafs [MAX_TOTAL_ENT_LEAFS]int
	var topnode int
	num_leafs := T.common.CMBoxLeafnums(ent.Absmin(), ent.Absmax(),
		leafs[:], MAX_TOTAL_ENT_LEAFS, &topnode)

	/* set areas */
	var clusters [MAX_TOTAL_ENT_LEAFS]int
	for i := 0; i < num_leafs; i++ {
		clusters[i] = T.common.CMLeafCluster(leafs[i])
		area := T.common.CMLeafArea(leafs[i])

		if area != 0 {
			/* doors may legally straggle two areas,
			but nothing should evern need more than that */
			if ent.Areanum() != 0 && (ent.Areanum() != area) {
				if ent.Areanum2() != 0 && (ent.Areanum2() != area) &&
					(T.sv.state == ss_loading) {
					log.Printf("Object touching 3 areas at %f %f %f\n",
						ent.Absmin()[0], ent.Absmin()[1], ent.Absmin()[2])
				}

				ent.SetAreanum2(area)
			} else {
				ent.SetAreanum(area)
			}
		}
	}

	if num_leafs >= MAX_TOTAL_ENT_LEAFS {
		/* assume we missed some leafs, and mark by headnode */
		ent.SetNumClusters(-1)
		ent.SetHeadnode(topnode)
	} else {
		ent.SetNumClusters(0)

		for i := 0; i < num_leafs; i++ {
			if clusters[i] == -1 {
				continue /* not a visible leaf */
			}

			found := false
			for j := 0; j < i; j++ {
				if clusters[j] == clusters[i] {
					found = true
					break
				}
			}

			if !found {
				if ent.NumClusters() == shared.MAX_ENT_CLUSTERS {
					/* assume we missed some leafs, and mark by headnode */
					ent.SetNumClusters(-1)
					ent.SetHeadnode(topnode)
					break
				}

				ent.Clusternums()[ent.NumClusters()] = clusters[i]
				ent.SetNumClusters(ent.NumClusters() + 1)
			}
		}
	}

	/* if first time, make sure old_origin is valid */
	if ent.Linkcount() == 0 {
		copy(ent.S().Old_origin[:], ent.S().Origin[:])
	}

	ent.SetLinkcount(ent.Linkcount() + 1)

	if ent.Solid() == shared.SOLID_NOT {
		return
	}

	/* find the first node that the ent's box crosses */
	node := &T.sv_areanodes[0]

	for {
		if node.axis == -1 {
			break
		}

		if ent.Absmin()[node.axis] > node.dist {
			node = node.children[0]
		} else if ent.Absmax()[node.axis] < node.dist {
			node = node.children[1]
		} else {
			break /* crosses the node */
		}
	}

	/* link it in */
	ent.Area().Self = ent
	if ent.Solid() == shared.SOLID_TRIGGER {
		InsertLinkBefore(ent.Area(), &node.trigger_edicts)
	} else {
		InsertLinkBefore(ent.Area(), &node.solid_edicts)
	}
}

func (T *qServer) svAreaEdicts_r(node *areanode_t) {

	/* touch linked edicts */
	var start *shared.Link_t
	if T.area_type == shared.AREA_SOLID {
		start = &node.solid_edicts
	} else {
		start = &node.trigger_edicts
	}

	var next *shared.Link_t
	for l := start.Next; l != start; l = next {
		next = l.Next
		check := l.Self

		if check.Solid() == shared.SOLID_NOT {
			continue /* deactivated */
		}

		if (check.Absmin()[0] > T.area_maxs[0]) ||
			(check.Absmin()[1] > T.area_maxs[1]) ||
			(check.Absmin()[2] > T.area_maxs[2]) ||
			(check.Absmax()[0] < T.area_mins[0]) ||
			(check.Absmax()[1] < T.area_mins[1]) ||
			(check.Absmax()[2] < T.area_mins[2]) {
			continue /* not touching */
		}

		if T.area_count == T.area_maxcount {
			log.Printf("SV_AreaEdicts: MAXCOUNT\n")
			return
		}

		T.area_list[T.area_count] = check
		T.area_count++
	}

	if node.axis == -1 {
		return /* terminal node */
	}

	/* recurse down both sides */
	if T.area_maxs[node.axis] > node.dist {
		T.svAreaEdicts_r(node.children[0])
	}

	if T.area_mins[node.axis] < node.dist {
		T.svAreaEdicts_r(node.children[1])
	}
}

func (T *qServer) svAreaEdicts(mins, maxs []float32, list []shared.Edict_s, maxcount, areatype int) int {
	T.area_mins = mins
	T.area_maxs = maxs
	T.area_list = list
	T.area_maxcount = maxcount
	T.area_type = areatype
	T.area_count = 0

	T.svAreaEdicts_r(&T.sv_areanodes[0])

	T.area_mins = nil
	T.area_maxs = nil
	T.area_list = nil
	T.area_maxcount = 0
	T.area_type = 0

	return T.area_count
}

func (T *qServer) svPointContents(p []float32) int {

	/* get base contents from world */
	contents := T.common.CMPointContents(p, T.sv.models[1].Headnode)

	/* or in contents from all the other entities */
	var touch [shared.MAX_EDICTS]shared.Edict_s
	num := T.svAreaEdicts(p, p, touch[:], shared.MAX_EDICTS, shared.AREA_SOLID)

	for i := 0; i < num; i++ {
		hit := touch[i]

		/* might intersect, so do an exact clip */
		headnode := T.svHullForEntity(hit)
		c2 := T.common.CMTransformedPointContents(p, headnode, hit.S().Origin[:], hit.S().Angles[:])

		contents |= c2
	}

	return contents
}

type moveclip_t struct {
	boxmins, boxmaxs [3]float32 /* enclose the test object along entire move */
	mins, maxs       []float32  /* size of the moving object */
	mins2, maxs2     [3]float32 /* size when clipping against mosnters */
	start, end       []float32
	trace            shared.Trace_t
	passedict        shared.Edict_s
	contentmask      int
}

/*
 * Returns a headnode that can be used for testing or clipping an
 * object of mins/maxs size. Offset is filled in to contain the
 * adjustment that must be added to the testing object's origin
 * to get a point to use with the returned hull.
 */
func (T *qServer) svHullForEntity(ent shared.Edict_s) int {
	/* decide which clipping hull to use, based on the size */
	if ent.Solid() == shared.SOLID_BSP {

		/* explicit hulls in the BSP model */
		model := T.sv.models[ent.S().Modelindex]

		if model == nil {
			log.Fatal("MOVETYPE_PUSH with a non bsp model ", ent.S().Modelindex)
		}

		return model.Headnode
	}

	/* create a temp hull from bounding box sizes */
	return T.common.CMHeadnodeForBox(ent.Mins(), ent.Maxs())
}

func (T *qServer) svClipMoveToEntities(clip *moveclip_t) {

	var touchlist [shared.MAX_EDICTS]shared.Edict_s
	num := T.svAreaEdicts(clip.boxmins[:], clip.boxmaxs[:], touchlist[:], shared.MAX_EDICTS, shared.AREA_SOLID)

	/* be careful, it is possible to have an entity in this
	list removed before we get to it (killtriggered) */
	for i := 0; i < num; i++ {
		touch := touchlist[i]

		if touch.Solid() == shared.SOLID_NOT {
			continue
		}

		if touch == clip.passedict {
			continue
		}

		if clip.trace.Allsolid {
			return
		}

		if clip.passedict != nil {
			if touch.Owner() == clip.passedict {
				continue /* don't clip against own missiles */
			}

			if clip.passedict.Owner() == touch {
				continue /* don't clip against owner */
			}
		}

		if (clip.contentmask&shared.CONTENTS_DEADMONSTER) == 0 &&
			(touch.Svflags()&shared.SVF_DEADMONSTER) != 0 {
			continue
		}

		/* might intersect, so do an exact clip */
		headnode := T.svHullForEntity(touch)
		angles := touch.S().Angles[:]

		if touch.Solid() != shared.SOLID_BSP {
			angles = []float32{0, 0, 0} /* boxes don't rotate */
		}

		var trace shared.Trace_t
		if (touch.Svflags() & shared.SVF_MONSTER) != 0 {
			trace = T.common.CMTransformedBoxTrace(clip.start, clip.end,
				clip.mins2[:], clip.maxs2[:], headnode, clip.contentmask,
				touch.S().Origin[:], angles)
		} else {
			trace = T.common.CMTransformedBoxTrace(clip.start, clip.end,
				clip.mins, clip.maxs, headnode, clip.contentmask,
				touch.S().Origin[:], angles)
		}

		if trace.Allsolid || trace.Startsolid ||
			(trace.Fraction < clip.trace.Fraction) {
			trace.Ent = touch

			if clip.trace.Startsolid {
				clip.trace = trace
				clip.trace.Startsolid = true
			} else {
				clip.trace = trace
			}
		}
	}
}

func SV_TraceBounds(start, mins, maxs, end, boxmins, boxmaxs []float32) {

	for i := 0; i < 3; i++ {
		if end[i] > start[i] {
			boxmins[i] = start[i] + mins[i] - 1
			boxmaxs[i] = end[i] + maxs[i] + 1
		} else {
			boxmins[i] = end[i] + mins[i] - 1
			boxmaxs[i] = start[i] + maxs[i] + 1
		}
	}
}

/*
 * Moves the given mins/maxs volume through the world from start to end.
 * Passedict and edicts owned by passedict are explicitly not checked.
 */
func (T *qServer) svTrace(start, mins, maxs, end []float32, passedict shared.Edict_s, contentmask int) shared.Trace_t {
	//  moveclip_t clip;

	if mins == nil {
		mins = []float32{0, 0, 0}
	}

	if maxs == nil {
		maxs = []float32{0, 0, 0}
	}

	clip := moveclip_t{}

	/* clip to world */
	clip.trace = T.common.CMBoxTrace(start, end, mins, maxs, 0, contentmask)
	clip.trace.Ent = T.ge.Edict(0)

	if clip.trace.Fraction == 0 {
		return clip.trace /* blocked by the world */
	}

	clip.contentmask = contentmask
	clip.start = start
	clip.end = end
	clip.mins = mins
	clip.maxs = maxs
	clip.passedict = passedict

	copy(clip.mins2[:], mins)
	copy(clip.maxs2[:], maxs)

	/* create the bounding box of the entire move */
	SV_TraceBounds(start, clip.mins2[:], clip.maxs2[:], end, clip.boxmins[:], clip.boxmaxs[:])

	/* clip to other solid entities */
	T.svClipMoveToEntities(&clip)

	return clip.trace
}
