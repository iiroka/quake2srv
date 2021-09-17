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
 * The collision model. Slaps "boxes" through the world and checks if
 * they collide with the world model, entities or other boxes.
 *
 * =======================================================================
 */
package common

import (
	"log"
	"math"
	"quake2srv/shared"
	"strconv"
)

type cnode_t struct {
	plane    *shared.Cplane_t
	children [2]int /* negative numbers are leafs */
}

type cbrushside_t struct {
	plane   *shared.Cplane_t
	surface *shared.Mapsurface_t
}

type cleaf_t struct {
	contents       int
	cluster        int
	area           int
	firstleafbrush uint16
	numleafbrushes uint16
}

type cbrush_t struct {
	contents       int
	numsides       int
	firstbrushside int
	checkcount     int /* to avoid repeated testings */
}

type carea_t struct {
	numareaportals  int
	firstareaportal int
	floodnum        int /* if two areas have equal floodnums, they are connected */
	floodvalid      int
}

type qCollision struct {
	map_visibility [shared.MAX_MAP_VISIBILITY]byte
	// DG: is casted to int32_t* in SV_FatPVS() so align accordingly
	pvsrow                    [shared.MAX_MAP_LEAFS / 8]byte
	phsrow                    [shared.MAX_MAP_LEAFS / 8]byte
	map_areas                 [shared.MAX_MAP_AREAS]carea_t
	map_brushes               [shared.MAX_MAP_BRUSHES]cbrush_t
	map_brushsides            [shared.MAX_MAP_BRUSHSIDES]cbrushside_t
	map_name                  string
	map_entitystring          string
	box_brush                 *cbrush_t
	box_leaf                  *cleaf_t
	map_leafs                 [shared.MAX_MAP_LEAFS]cleaf_t
	map_cmodels               [shared.MAX_MAP_MODELS]shared.Cmodel_t
	map_nodes                 [shared.MAX_MAP_NODES + 6]cnode_t /* extra for box hull */
	box_planes                []shared.Cplane_t
	map_planes                [shared.MAX_MAP_PLANES + 6]shared.Cplane_t /* extra for box hull */
	map_noareas               *shared.CvarT
	map_areaportals           [shared.MAX_MAP_AREAPORTALS]shared.Dareaportal_t
	map_vis                   shared.Dvis_t
	box_headnode              int
	checkcount                int
	emptyleaf, solidleaf      int
	floodvalid                int
	leaf_mins, leaf_maxs      []float32
	leaf_count, leaf_maxcount int
	leaf_list                 []int
	leaf_topnode              int
	numareaportals            int
	numareas                  int
	numbrushes                int
	numbrushsides             int
	numclusters               int
	numcmodels                int
	numentitychars            int
	numleafbrushes            int
	numleafs                  int /* allow leaf funcs to be called without a map */
	numnodes                  int
	numplanes                 int
	numtexinfo                int
	numvisibility             int
	trace_contents            int
	map_surfaces              [shared.MAX_MAP_TEXINFO]shared.Mapsurface_t
	nullsurface               shared.Mapsurface_t
	portalopen                [shared.MAX_MAP_AREAPORTALS]bool
	trace_ispoint             bool /* optimized case */
	trace_trace               shared.Trace_t
	map_leafbrushes           [shared.MAX_MAP_LEAFBRUSHES]uint16
	trace_start, trace_end    [3]float32
	trace_mins, trace_maxs    [3]float32
	trace_extents             [3]float32
}

/* 1/32 epsilon to keep floating point happy */
const DIST_EPSILON = 0.03125

func (T *qCommon) floodArea_r(area *carea_t, floodnum int) {

	if area.floodvalid == T.collision.floodvalid {
		if area.floodnum == floodnum {
			return
		}

		T.Com_Error(shared.ERR_DROP, "FloodArea_r: reflooded")
		return
	}

	area.floodnum = floodnum
	area.floodvalid = T.collision.floodvalid

	for i := 0; i < area.numareaportals; i++ {
		p := &T.collision.map_areaportals[area.firstareaportal+i]
		if T.collision.portalopen[p.Portalnum] {
			T.floodArea_r(&T.collision.map_areas[p.Otherarea], floodnum)
		}
	}
}

func (T *qCommon) floodAreaConnections() {

	/* all current floods are now invalid */
	T.collision.floodvalid++
	floodnum := 0

	/* area 0 is not used */
	for i := 1; i < T.collision.numareas; i++ {
		area := &T.collision.map_areas[i]

		if area.floodvalid == T.collision.floodvalid {
			continue /* already flooded into */
		}

		floodnum++
		T.floodArea_r(area, floodnum)
	}
}

func (T *qCommon) CMAreasConnected(area1, area2 int) bool {
	if T.collision.map_noareas.Bool() {
		return true
	}

	if (area1 > T.collision.numareas) || (area2 > T.collision.numareas) {
		T.Com_Error(shared.ERR_DROP, "area > numareas")
		return false
	}

	if T.collision.map_areas[area1].floodnum == T.collision.map_areas[area2].floodnum {
		return true
	}

	return false
}

/*
 * Writes a length byte followed by a bit vector of all the areas
 * that area in the same flood as the area parameter
 *
 * This is used by the client refreshes to cull visibility
 */
func (T *qCommon) CMWriteAreaBits(buffer []byte, area int) int {

	bytes := (T.collision.numareas + 7) >> 3

	if T.collision.map_noareas.Bool() {
		/* for debugging, send everything */
		for i := range buffer {
			buffer[i] = 0xFF
		}
	} else {
		for i := range buffer {
			buffer[i] = 0
		}

		floodnum := T.collision.map_areas[area].floodnum

		for i := 0; i < T.collision.numareas; i++ {
			if (T.collision.map_areas[i].floodnum == floodnum) || area == 0 {
				buffer[i>>3] |= 1 << (i & 7)
			}
		}
	}

	return bytes
}

/*
 * Returns true if any leaf under headnode has a cluster that
 * is potentially visible
 */
func (T *qCommon) CMHeadnodeVisible(nodenum int, visbits []byte) bool {

	if nodenum < 0 {
		leafnum1 := -1 - nodenum
		cluster := T.collision.map_leafs[leafnum1].cluster

		if cluster == -1 {
			return false
		}

		if (visbits[cluster>>3] & (1 << (cluster & 7))) != 0 {
			return true
		}

		return false
	}

	node := &T.collision.map_nodes[nodenum]

	if T.CMHeadnodeVisible(node.children[0], visbits) {
		return true
	}

	return T.CMHeadnodeVisible(node.children[1], visbits)
}

/*
 * Set up the planes and nodes so that the six floats of a bounding box
 * can just be stored out and get a proper clipping hull structure.
 */
func (T *qCommon) initBoxHull() error {

	T.collision.box_headnode = T.collision.numnodes
	T.collision.box_planes = T.collision.map_planes[T.collision.numplanes:]

	if (T.collision.numnodes+6 > shared.MAX_MAP_NODES) ||
		(T.collision.numbrushes+1 > shared.MAX_MAP_BRUSHES) ||
		(T.collision.numleafbrushes+1 > shared.MAX_MAP_LEAFBRUSHES) ||
		(T.collision.numbrushsides+6 > shared.MAX_MAP_BRUSHSIDES) ||
		(T.collision.numplanes+12 > shared.MAX_MAP_PLANES) {
		return T.Com_Error(shared.ERR_DROP, "Not enough room for box tree")
	}

	T.collision.box_brush = &T.collision.map_brushes[T.collision.numbrushes]
	T.collision.box_brush.numsides = 6
	T.collision.box_brush.firstbrushside = T.collision.numbrushsides
	T.collision.box_brush.contents = shared.CONTENTS_MONSTER

	T.collision.box_leaf = &T.collision.map_leafs[T.collision.numleafs]
	T.collision.box_leaf.contents = shared.CONTENTS_MONSTER
	T.collision.box_leaf.firstleafbrush = uint16(T.collision.numleafbrushes)
	T.collision.box_leaf.numleafbrushes = 1

	T.collision.map_leafbrushes[T.collision.numleafbrushes] = uint16(T.collision.numbrushes)

	for i := 0; i < 6; i++ {
		side := i & 1

		/* brush sides */
		s := &T.collision.map_brushsides[T.collision.numbrushsides+i]
		s.plane = &T.collision.map_planes[T.collision.numplanes+i*2+side]
		s.surface = &T.collision.nullsurface

		/* nodes */
		c := &T.collision.map_nodes[T.collision.box_headnode+i]
		c.plane = &T.collision.map_planes[T.collision.numplanes+i*2]
		c.children[side] = -1 - T.collision.emptyleaf

		if i != 5 {
			c.children[side^1] = T.collision.box_headnode + i + 1
		} else {
			c.children[side^1] = -1 - T.collision.numleafs
		}

		/* planes */
		p := &T.collision.box_planes[i*2]
		p.Type = byte(i >> 1)
		p.Signbits = 0
		p.Normal[0] = 0
		p.Normal[1] = 0
		p.Normal[2] = 0
		p.Normal[i>>1] = 1

		p = &T.collision.box_planes[i*2+1]
		p.Type = byte(3 + (i >> 1))
		p.Signbits = 0
		p.Normal[0] = 0
		p.Normal[1] = 0
		p.Normal[2] = 0
		p.Normal[i>>1] = -1
	}
	return nil
}

/*
 * To keep everything totally uniform, bounding boxes are turned into
 * small BSP trees instead of being compared directly.
 */
func (T *qCommon) CMHeadnodeForBox(mins, maxs []float32) int {
	T.collision.box_planes[0].Dist = maxs[0]
	T.collision.box_planes[1].Dist = -maxs[0]
	T.collision.box_planes[2].Dist = mins[0]
	T.collision.box_planes[3].Dist = -mins[0]
	T.collision.box_planes[4].Dist = maxs[1]
	T.collision.box_planes[5].Dist = -maxs[1]
	T.collision.box_planes[6].Dist = mins[1]
	T.collision.box_planes[7].Dist = -mins[1]
	T.collision.box_planes[8].Dist = maxs[2]
	T.collision.box_planes[9].Dist = -maxs[2]
	T.collision.box_planes[10].Dist = mins[2]
	T.collision.box_planes[11].Dist = -mins[2]

	return T.collision.box_headnode
}

func (T *qCommon) cmPointLeafnum_r(p []float32, num int) int {

	for num >= 0 {
		node := T.collision.map_nodes[num]
		plane := node.plane

		var d float32
		if plane.Type < 3 {
			d = p[plane.Type] - plane.Dist
		} else {
			d = shared.DotProduct(plane.Normal[:], p) - plane.Dist
		}

		if d < 0 {
			num = node.children[1]
		} else {
			num = node.children[0]
		}
	}

	// #ifndef DEDICATED_ONLY
	// 	c_pointcontents++; /* optimize counter */
	// #endif

	return -1 - num
}

func (T *qCommon) CMPointLeafnum(p []float32) int {
	if T.collision.numplanes == 0 {
		return 0 /* sound may call this without map loaded */
	}

	return T.cmPointLeafnum_r(p, 0)
}

/*
 * Fills in a list of all the leafs touched
 */

func (T *qCommon) cmBoxLeafnums_r(nodenum int) {

	for {
		if nodenum < 0 {
			if T.collision.leaf_count >= T.collision.leaf_maxcount {
				return
			}

			T.collision.leaf_list[T.collision.leaf_count] = -1 - nodenum
			T.collision.leaf_count++
			return
		}

		node := &T.collision.map_nodes[nodenum]
		plane := node.plane
		s := shared.BoxOnPlaneSide(T.collision.leaf_mins, T.collision.leaf_maxs, plane)

		if s == 1 {
			nodenum = node.children[0]
		} else if s == 2 {
			nodenum = node.children[1]
		} else {
			/* go down both */
			if T.collision.leaf_topnode == -1 {
				T.collision.leaf_topnode = nodenum
			}

			T.cmBoxLeafnums_r(node.children[0])
			nodenum = node.children[1]
		}
	}
}

func (T *qCommon) cmBoxLeafnums_headnode(mins, maxs []float32, list []int,
	listsize, headnode int, topnode *int) int {
	T.collision.leaf_list = list
	T.collision.leaf_count = 0
	T.collision.leaf_maxcount = listsize
	T.collision.leaf_mins = mins
	T.collision.leaf_maxs = maxs

	T.collision.leaf_topnode = -1

	T.cmBoxLeafnums_r(headnode)

	if topnode != nil {
		*topnode = T.collision.leaf_topnode
	}

	return T.collision.leaf_count
}

func (T *qCommon) CMBoxLeafnums(mins, maxs []float32, list []int, listsize int, topnode *int) int {
	return T.cmBoxLeafnums_headnode(mins, maxs, list, listsize, T.collision.map_cmodels[0].Headnode, topnode)
}

func (T *qCommon) CMPointContents(p []float32, headnode int) int {

	if T.collision.numnodes == 0 { /* map not loaded */
		return 0
	}

	l := T.cmPointLeafnum_r(p, headnode)

	return T.collision.map_leafs[l].contents
}

/*
 * Handles offseting and rotation of the end points for moving and
 * rotating entities
 */
func (T *qCommon) CMTransformedPointContents(p []float32, headnode int, origin, angles []float32) int {

	/* subtract origin offset */
	p_l := make([]float32, 3)
	shared.VectorSubtract(p, origin, p_l)

	/* rotate start and end into the models frame of reference */
	if (headnode != T.collision.box_headnode) &&
		(angles[0] != 0 || angles[1] != 0 || angles[2] != 0) {
		forward := make([]float32, 3)
		right := make([]float32, 3)
		up := make([]float32, 3)
		shared.AngleVectors(angles, forward, right, up)

		temp := make([]float32, 3)
		copy(temp, p_l)
		p_l[0] = shared.DotProduct(temp, forward)
		p_l[1] = -shared.DotProduct(temp, right)
		p_l[2] = shared.DotProduct(temp, up)
	}

	l := T.cmPointLeafnum_r(p_l, headnode)

	return T.collision.map_leafs[l].contents
}

func (T *qCommon) clipBoxToBrush(mins, maxs, p1, p2 []float32, trace *shared.Trace_t, brush *cbrush_t) {
	// int i, j;
	// cplane_t *plane, *clipplane;
	// float dist;
	// float enterfrac, leavefrac;
	// vec3_t ofs;
	// float d1, d2;
	// qboolean getout, startout;
	// float f;
	// cbrushside_t *side, *leadside;

	var enterfrac float32 = -1
	var leavefrac float32 = 1
	var clipplane *shared.Cplane_t = nil

	if brush.numsides == 0 {
		return
	}

	// #ifndef DEDICATED_ONLY
	// 	c_brush_traces++;
	// #endif

	getout := false
	startout := false
	var leadside *cbrushside_t = nil

	for i := 0; i < brush.numsides; i++ {
		side := &T.collision.map_brushsides[brush.firstbrushside+i]
		plane := side.plane

		var dist float32
		if !T.collision.trace_ispoint {
			/* general box case
			push the plane out
			apropriately for mins/maxs */
			var ofs [3]float32
			for j := 0; j < 3; j++ {
				if plane.Normal[j] < 0 {
					ofs[j] = maxs[j]
				} else {
					ofs[j] = mins[j]
				}
			}

			dist = shared.DotProduct(ofs[:], plane.Normal[:])
			dist = plane.Dist - dist
		} else {
			/* special point case */
			dist = plane.Dist
		}

		d1 := shared.DotProduct(p1, plane.Normal[:]) - dist
		d2 := shared.DotProduct(p2, plane.Normal[:]) - dist

		if d2 > 0 {
			getout = true /* endpoint is not in solid */
		}

		if d1 > 0 {
			startout = true
		}

		/* if completely in front of face, no intersection */
		if (d1 > 0) && (d2 >= d1) {
			return
		}

		if (d1 <= 0) && (d2 <= 0) {
			continue
		}

		/* crosses face */
		if d1 > d2 {
			/* enter */
			f := (d1 - DIST_EPSILON) / (d1 - d2)

			if f > enterfrac {
				enterfrac = f
				clipplane = plane
				leadside = side
			}
		} else {
			/* leave */
			f := (d1 + DIST_EPSILON) / (d1 - d2)

			if f < leavefrac {
				leavefrac = f
			}
		}
	}

	if !startout {
		/* original point was inside brush */
		trace.Startsolid = true

		if !getout {
			trace.Allsolid = true
		}

		return
	}

	if enterfrac < leavefrac {
		if (enterfrac > -1) && (enterfrac < trace.Fraction) {
			if enterfrac < 0 {
				enterfrac = 0
			}

			if clipplane == nil {
				log.Fatal("clipplane was NULL!\n")
			}

			trace.Fraction = enterfrac
			trace.Plane = *clipplane
			trace.Surface = &leadside.surface.C
			trace.Contents = brush.contents
		}
	}
}

func (T *qCommon) testBoxInBrush(mins, maxs, p1 []float32, trace *shared.Trace_t, brush *cbrush_t) {

	if brush.numsides == 0 {
		return
	}

	for i := 0; i < brush.numsides; i++ {
		side := &T.collision.map_brushsides[brush.firstbrushside+i]
		plane := side.plane

		/* general box case
		push the plane out
		apropriately for mins/maxs */
		var ofs [3]float32
		for j := 0; j < 3; j++ {
			if plane.Normal[j] < 0 {
				ofs[j] = maxs[j]
			} else {
				ofs[j] = mins[j]
			}
		}

		dist := shared.DotProduct(ofs[:], plane.Normal[:])
		dist = plane.Dist - dist

		d1 := shared.DotProduct(p1, plane.Normal[:]) - dist

		/* if completely in front of face, no intersection */
		if d1 > 0 {
			return
		}
	}

	/* inside this brush */
	trace.Startsolid = true
	trace.Allsolid = true
	trace.Fraction = 0
	trace.Contents = brush.contents
}

func (T *qCommon) traceToLeaf(leafnum int) {

	leaf := &T.collision.map_leafs[leafnum]

	if (leaf.contents & T.collision.trace_contents) == 0 {
		return
	}

	/* trace line against all brushes in the leaf */
	for k := 0; k < int(leaf.numleafbrushes); k++ {
		brushnum := T.collision.map_leafbrushes[int(leaf.firstleafbrush)+k]
		b := &T.collision.map_brushes[brushnum]

		if b.checkcount == T.collision.checkcount {
			continue /* already checked this brush in another leaf */
		}

		b.checkcount = T.collision.checkcount

		if (b.contents & T.collision.trace_contents) == 0 {
			continue
		}

		T.clipBoxToBrush(T.collision.trace_mins[:], T.collision.trace_maxs[:], T.collision.trace_start[:],
			T.collision.trace_end[:], &T.collision.trace_trace, b)

		if T.collision.trace_trace.Fraction == 0 {
			return
		}
	}
}

func (T *qCommon) testInLeaf(leafnum int) {

	leaf := &T.collision.map_leafs[leafnum]
	if (leaf.contents & T.collision.trace_contents) == 0 {
		return
	}

	/* trace line against all brushes in the leaf */
	for k := 0; k < int(leaf.numleafbrushes); k++ {
		brushnum := T.collision.map_leafbrushes[int(leaf.firstleafbrush)+k]
		b := &T.collision.map_brushes[brushnum]

		if b.checkcount == T.collision.checkcount {
			continue /* already checked this brush in another leaf */
		}

		b.checkcount = T.collision.checkcount

		if (b.contents & T.collision.trace_contents) == 0 {
			continue
		}

		T.testBoxInBrush(T.collision.trace_mins[:], T.collision.trace_maxs[:], T.collision.trace_start[:], &T.collision.trace_trace, b)

		if T.collision.trace_trace.Fraction == 0 {
			return
		}
	}
}

func (T *qCommon) recursiveHullCheck(num int, p1f, p2f float32, p1, p2 []float32) {

	if T.collision.trace_trace.Fraction <= p1f {
		return /* already hit something nearer */
	}

	/* if < 0, we are in a leaf node */
	if num < 0 {
		T.traceToLeaf(-1 - num)
		return
	}

	/* find the point distances to the seperating plane
	and the offset for the size of the box */
	node := T.collision.map_nodes[num]
	plane := node.plane

	var offset, t1, t2 float32
	if plane.Type < 3 {
		t1 = p1[plane.Type] - plane.Dist
		t2 = p2[plane.Type] - plane.Dist
		offset = T.collision.trace_extents[plane.Type]
	} else {
		t1 = shared.DotProduct(plane.Normal[:], p1) - plane.Dist
		t2 = shared.DotProduct(plane.Normal[:], p2) - plane.Dist

		if T.collision.trace_ispoint {
			offset = 0
		} else {
			offset = float32(math.Abs(float64(T.collision.trace_extents[0]*plane.Normal[0])) +
				math.Abs(float64(T.collision.trace_extents[1]*plane.Normal[1])) +
				math.Abs(float64(T.collision.trace_extents[2]*plane.Normal[2])))
		}
	}

	/* see which sides we need to consider */
	if (t1 >= offset) && (t2 >= offset) {
		T.recursiveHullCheck(node.children[0], p1f, p2f, p1, p2)
		return
	}

	if (t1 < -offset) && (t2 < -offset) {
		T.recursiveHullCheck(node.children[1], p1f, p2f, p1, p2)
		return
	}

	/* put the crosspoint DIST_EPSILON pixels on the near side */
	var frac, frac2 float32
	var side int
	if t1 < t2 {
		idist := 1.0 / (t1 - t2)
		side = 1
		frac2 = (t1 + offset + DIST_EPSILON) * idist
		frac = (t1 - offset + DIST_EPSILON) * idist
	} else if t1 > t2 {
		idist := 1.0 / (t1 - t2)
		side = 0
		frac2 = (t1 - offset - DIST_EPSILON) * idist
		frac = (t1 + offset + DIST_EPSILON) * idist
	} else {
		side = 0
		frac = 1
		frac2 = 0
	}

	/* move up to the node */
	if frac < 0 {
		frac = 0
	}

	if frac > 1 {
		frac = 1
	}

	midf := p1f + (p2f-p1f)*frac

	var mid [3]float32
	for i := 0; i < 3; i++ {
		mid[i] = p1[i] + frac*(p2[i]-p1[i])
	}

	T.recursiveHullCheck(node.children[side], p1f, midf, p1, mid[:])

	/* go past the node */
	if frac2 < 0 {
		frac2 = 0
	}

	if frac2 > 1 {
		frac2 = 1
	}

	midf = p1f + (p2f-p1f)*frac2

	for i := 0; i < 3; i++ {
		mid[i] = p1[i] + frac2*(p2[i]-p1[i])
	}

	T.recursiveHullCheck(node.children[side^1], midf, p2f, mid[:], p2)
}

func (T *qCommon) CMBoxTrace(start, end, mins, maxs []float32,
	headnode, brushmask int) shared.Trace_t {
	// int i;

	T.collision.checkcount++ /* for multi-check avoidance */

	// #ifndef DEDICATED_ONLY
	// 	c_traces++; /* for statistics, may be zeroed */
	// #endif

	/* fill in a default trace */
	T.collision.trace_trace = shared.Trace_t{}
	T.collision.trace_trace.Fraction = 1
	T.collision.trace_trace.Surface = &shared.Csurface_t{}

	if T.collision.numnodes == 0 { /* map not loaded */
		return T.collision.trace_trace
	}

	T.collision.trace_contents = brushmask
	copy(T.collision.trace_start[:], start)
	copy(T.collision.trace_end[:], end)
	copy(T.collision.trace_mins[:], mins)
	copy(T.collision.trace_maxs[:], maxs)

	/* check for position test special case */
	if (start[0] == end[0]) && (start[1] == end[1]) && (start[2] == end[2]) {

		c1 := make([]float32, 3)
		c2 := make([]float32, 3)
		shared.VectorAdd(start, mins, c1)
		shared.VectorAdd(start, maxs, c2)

		for i := 0; i < 3; i++ {
			c1[i] -= 1
			c2[i] += 1
		}

		var leafs [1024]int
		var topnode int
		numleafs := T.cmBoxLeafnums_headnode(c1, c2, leafs[:], 1024, headnode, &topnode)

		for i := 0; i < numleafs; i++ {
			T.testInLeaf(leafs[i])

			if T.collision.trace_trace.Allsolid {
				break
			}
		}

		copy(T.collision.trace_trace.Endpos[:], start)
		return T.collision.trace_trace
	}

	/* check for point special case */
	if (mins[0] == 0) && (mins[1] == 0) && (mins[2] == 0) &&
		(maxs[0] == 0) && (maxs[1] == 0) && (maxs[2] == 0) {
		T.collision.trace_ispoint = true
		T.collision.trace_extents[0] = 0
		T.collision.trace_extents[1] = 0
		T.collision.trace_extents[2] = 0
	} else {
		T.collision.trace_ispoint = false
		for i := 0; i < 3; i++ {
			if -mins[i] > maxs[i] {
				T.collision.trace_extents[i] = -mins[i]
			} else {
				T.collision.trace_extents[i] = maxs[i]
			}
		}
	}

	/* general sweeping through world */
	T.recursiveHullCheck(headnode, 0, 1, start, end)

	if T.collision.trace_trace.Fraction == 1 {
		copy(T.collision.trace_trace.Endpos[:], end)
	} else {
		for i := 0; i < 3; i++ {
			T.collision.trace_trace.Endpos[i] = start[i] + T.collision.trace_trace.Fraction*
				(end[i]-start[i])
		}
	}

	return T.collision.trace_trace
}

/*
 * Handles offseting and rotation of the end points for moving and
 * rotating entities
 */
func (T *qCommon) CMTransformedBoxTrace(start, end, mins, maxs []float32,
	headnode, brushmask int, origin, angles []float32) shared.Trace_t {

	/* subtract origin offset */
	start_l := make([]float32, 3)
	shared.VectorSubtract(start, origin, start_l)
	end_l := make([]float32, 3)
	shared.VectorSubtract(end, origin, end_l)

	/* rotate start and end into the models frame of reference */
	rotated := false
	if (headnode != T.collision.box_headnode) &&
		(angles[0] != 0 || angles[1] != 0 || angles[2] != 0) {
		rotated = true
	}

	if rotated {
		forward := make([]float32, 3)
		right := make([]float32, 3)
		up := make([]float32, 3)
		shared.AngleVectors(angles, forward, right, up)

		temp := make([]float32, 3)
		copy(temp, start_l)
		start_l[0] = shared.DotProduct(temp, forward)
		start_l[1] = -shared.DotProduct(temp, right)
		start_l[2] = shared.DotProduct(temp, up)

		copy(temp, end_l)
		end_l[0] = shared.DotProduct(temp, forward)
		end_l[1] = -shared.DotProduct(temp, right)
		end_l[2] = shared.DotProduct(temp, up)
	}

	/* sweep the box through the model */
	trace := T.CMBoxTrace(start_l, end_l, mins, maxs, headnode, brushmask)

	if rotated && (trace.Fraction != 1.0) {
		a := make([]float32, 3)
		shared.VectorNegate(angles, a)
		forward := make([]float32, 3)
		right := make([]float32, 3)
		up := make([]float32, 3)
		shared.AngleVectors(a, forward, right, up)

		temp := make([]float32, 3)
		copy(temp, trace.Plane.Normal[:])
		trace.Plane.Normal[0] = shared.DotProduct(temp, forward)
		trace.Plane.Normal[1] = -shared.DotProduct(temp, right)
		trace.Plane.Normal[2] = shared.DotProduct(temp, up)
	}

	trace.Endpos[0] = start[0] + trace.Fraction*(end[0]-start[0])
	trace.Endpos[1] = start[1] + trace.Fraction*(end[1]-start[1])
	trace.Endpos[2] = start[2] + trace.Fraction*(end[2]-start[2])

	return trace
}

func (T *qCommon) cmodLoadSubmodels(l shared.Lump_t, name string, buf []byte) error {

	if (l.Filelen % shared.Dmodel_size) != 0 {
		return T.Com_Error(shared.ERR_DROP, "Mod_LoadSubmodels: funny lump size")
	}

	count := l.Filelen / shared.Dmodel_size

	if count < 1 {
		return T.Com_Error(shared.ERR_DROP, "Map with no models")
	}

	if count > shared.MAX_MAP_MODELS {
		return T.Com_Error(shared.ERR_DROP, "Map has too many models")
	}

	T.collision.numcmodels = int(count)

	for i := 0; i < int(count); i++ {
		src := shared.Dmodel(buf[int(l.Fileofs)+i*shared.Dmodel_size:])
		out := &T.collision.map_cmodels[i]

		for j := 0; j < 3; j++ {
			/* spread the mins / maxs by a pixel */
			out.Mins[j] = src.Mins[j] - 1
			out.Maxs[j] = src.Maxs[j] + 1
			out.Origin[j] = src.Origin[j]
		}

		out.Headnode = int(src.Headnode)
	}
	return nil
}

func (T *qCommon) cmodLoadSurfaces(l shared.Lump_t, name string, buf []byte) error {
	if (l.Filelen % shared.Texinfo_size) != 0 {
		return T.Com_Error(shared.ERR_DROP, "Mod_LoadSubmodels: funny lump size")
	}

	count := l.Filelen / shared.Texinfo_size

	if count < 1 {
		return T.Com_Error(shared.ERR_DROP, "Map with no surfaces")
	}

	if count > shared.MAX_MAP_TEXINFO {
		return T.Com_Error(shared.ERR_DROP, "Map has too many surfaces")
	}

	T.collision.numtexinfo = int(count)
	for i := 0; i < int(count); i++ {
		src := shared.Texinfo(buf[int(l.Fileofs)+i*shared.Texinfo_size:])
		out := &T.collision.map_surfaces[i]

		out.C.Name = src.Texture
		out.Rname = src.Texture
		out.C.Flags = int(src.Flags)
		out.C.Value = int(src.Value)
	}
	return nil
}

func (T *qCommon) cmodLoadNodes(l shared.Lump_t, name string, buf []byte) error {
	if (l.Filelen % shared.Dnode_size) != 0 {
		return T.Com_Error(shared.ERR_DROP, "cmodLoadNodes: funny lump size")
	}

	count := l.Filelen / shared.Dnode_size

	if count < 1 {
		return T.Com_Error(shared.ERR_DROP, "Map with no nodes")
	}

	if count > shared.MAX_MAP_NODES {
		return T.Com_Error(shared.ERR_DROP, "Map has too many nodes")
	}

	T.collision.numnodes = int(count)

	for i := 0; i < int(count); i++ {
		src := shared.Dnode(buf[int(l.Fileofs)+i*shared.Dnode_size:])
		out := &T.collision.map_nodes[i]

		out.plane = &T.collision.map_planes[src.Planenum]

		for j := 0; j < 2; j++ {
			out.children[j] = int(src.Children[j])
		}
	}
	return nil
}

func (T *qCommon) cmodLoadBrushes(l shared.Lump_t, name string, buf []byte) error {
	if (l.Filelen % shared.Dbrush_size) != 0 {
		return T.Com_Error(shared.ERR_DROP, "cmodLoadBrushes: funny lump size")
	}

	count := l.Filelen / shared.Dbrush_size

	if count > shared.MAX_MAP_BRUSHES {
		return T.Com_Error(shared.ERR_DROP, "Map has too many brushes")
	}

	T.collision.numbrushes = int(count)

	for i := 0; i < int(count); i++ {
		src := shared.Dbrush(buf[int(l.Fileofs)+i*shared.Dbrush_size:])
		out := &T.collision.map_brushes[i]

		out.firstbrushside = int(src.Firstside)
		out.numsides = int(src.Numsides)
		out.contents = int(src.Contents)
	}
	return nil
}

func (T *qCommon) cmodLoadLeafs(l shared.Lump_t, name string, buf []byte) error {
	if (l.Filelen % shared.Dleaf_size) != 0 {
		return T.Com_Error(shared.ERR_DROP, "cmodLoadLeafs: funny lump size")
	}

	count := l.Filelen / shared.Dleaf_size

	if count < 1 {
		return T.Com_Error(shared.ERR_DROP, "Map with no leafs")
	}

	/* need to save space for box planes */
	if count > shared.MAX_MAP_PLANES {
		return T.Com_Error(shared.ERR_DROP, "Map has too many leafs")
	}

	T.collision.numleafs = int(count)
	T.collision.numclusters = 0

	for i := 0; i < int(count); i++ {
		src := shared.Dleaf(buf[int(l.Fileofs)+i*shared.Dleaf_size:])
		out := &T.collision.map_leafs[i]

		out.contents = int(src.Contents)
		out.cluster = int(src.Cluster)
		out.area = int(src.Area)
		out.firstleafbrush = src.Firstleafbrush
		out.numleafbrushes = src.Numleafbrushes

		if out.cluster >= T.collision.numclusters {
			T.collision.numclusters = out.cluster + 1
		}
	}

	if T.collision.map_leafs[0].contents != shared.CONTENTS_SOLID {
		return T.Com_Error(shared.ERR_DROP, "Map leaf 0 is not CONTENTS_SOLID")
	}

	T.collision.solidleaf = 0
	T.collision.emptyleaf = -1

	for i := 1; i < T.collision.numleafs; i++ {
		if T.collision.map_leafs[i].contents == 0 {
			T.collision.emptyleaf = i
			break
		}
	}

	if T.collision.emptyleaf == -1 {
		return T.Com_Error(shared.ERR_DROP, "Map does not have an empty leaf")
	}
	return nil
}

func (T *qCommon) cmodLoadPlanes(l shared.Lump_t, name string, buf []byte) error {
	if (l.Filelen % shared.Dplane_size) != 0 {
		return T.Com_Error(shared.ERR_DROP, "cmodLoadPlanes: funny lump size")
	}

	count := l.Filelen / shared.Dplane_size

	if count < 1 {
		return T.Com_Error(shared.ERR_DROP, "Map with no planes")
	}

	/* need to save space for box planes */
	if count > shared.MAX_MAP_PLANES {
		return T.Com_Error(shared.ERR_DROP, "Map has too many planes")
	}

	T.collision.numplanes = int(count)

	for i := 0; i < int(count); i++ {
		src := shared.Dplane(buf[int(l.Fileofs)+i*shared.Dplane_size:])
		out := &T.collision.map_planes[i]

		bits := 0

		for j := 0; j < 3; j++ {
			out.Normal[j] = src.Normal[j]

			if out.Normal[j] < 0 {
				bits |= 1 << j
			}
		}

		out.Dist = src.Dist
		out.Type = byte(src.Type)
		out.Signbits = byte(bits)
	}
	return nil
}

func (T *qCommon) cmodLoadLeafBrushes(l shared.Lump_t, name string, buf []byte) error {
	if (l.Filelen % 2) != 0 {
		return T.Com_Error(shared.ERR_DROP, "cmodLoadLeafBrushes: funny lump size")
	}

	count := l.Filelen / 2

	if count < 1 {
		return T.Com_Error(shared.ERR_DROP, "Map with no leafbrushes")
	}

	/* need to save space for box planes */
	if count > shared.MAX_MAP_LEAFBRUSHES {
		return T.Com_Error(shared.ERR_DROP, "Map has too many leafbrushes")
	}

	T.collision.numleafbrushes = int(count)

	for i := 0; i < int(count); i++ {
		T.collision.map_leafbrushes[i] = shared.ReadUint16(buf[int(l.Fileofs)+i*2:])
	}
	return nil
}

func (T *qCommon) cmodLoadBrushSides(l shared.Lump_t, name string, buf []byte) error {
	if (l.Filelen % shared.Dbrushside_size) != 0 {
		return T.Com_Error(shared.ERR_DROP, "cmodLoadBrushSides: funny lump size")
	}

	count := l.Filelen / shared.Dbrushside_size

	/* need to save space for box planes */
	if count > shared.MAX_MAP_BRUSHSIDES {
		return T.Com_Error(shared.ERR_DROP, "Map has too many planes")
	}

	T.collision.numbrushsides = int(count)

	for i := 0; i < int(count); i++ {
		src := shared.Dbrushside(buf[int(l.Fileofs)+i*shared.Dbrushside_size:])
		out := &T.collision.map_brushsides[i]

		out.plane = &T.collision.map_planes[src.Planenum]
		j := int(src.Texinfo)

		if j >= T.collision.numtexinfo {
			return T.Com_Error(shared.ERR_DROP, "Bad brushside texinfo")
		}

		if j > 0 {
			out.surface = &T.collision.map_surfaces[j]
		}
	}
	return nil
}

func (T *qCommon) cmodLoadAreas(l shared.Lump_t, name string, buf []byte) error {
	if (l.Filelen % shared.Darea_size) != 0 {
		return T.Com_Error(shared.ERR_DROP, "cmodLoadAreas: funny lump size")
	}

	count := l.Filelen / shared.Darea_size

	if count > shared.MAX_MAP_AREAS {
		return T.Com_Error(shared.ERR_DROP, "Map has too many areas")
	}

	T.collision.numareas = int(count)

	for i := 0; i < int(count); i++ {
		src := shared.Darea(buf[int(l.Fileofs)+i*shared.Darea_size:])
		out := &T.collision.map_areas[i]

		out.numareaportals = int(src.Numareaportals)
		out.firstareaportal = int(src.Firstareaportal)
		out.floodvalid = 0
		out.floodnum = 0
	}
	return nil
}

func (T *qCommon) cmodLoadAreaPortals(l shared.Lump_t, name string, buf []byte) error {
	if (l.Filelen % shared.Dareaportal_size) != 0 {
		return T.Com_Error(shared.ERR_DROP, "cmodLoadAreaPortals: funny lump size")
	}

	count := l.Filelen / shared.Dareaportal_size

	if count > shared.MAX_MAP_AREAS {
		return T.Com_Error(shared.ERR_DROP, "Map has too many areas")
	}

	T.collision.numareaportals = int(count)

	for i := 0; i < int(count); i++ {
		T.collision.map_areaportals[i] = shared.Dareaportal(buf[int(l.Fileofs)+i*shared.Dareaportal_size:])
	}
	return nil
}

func (T *qCommon) cmodLoadVisibility(l shared.Lump_t, name string, buf []byte) error {
	T.collision.numvisibility = int(l.Filelen)

	if l.Filelen > shared.MAX_MAP_VISIBILITY {
		return T.Com_Error(shared.ERR_DROP, "Map has too large visibility lump")
	}

	copy(T.collision.map_visibility[:], buf[l.Fileofs:l.Fileofs+l.Filelen])

	T.collision.map_vis = *shared.Dvis(T.collision.map_visibility[:])
	return nil
}

func (T *qCommon) cmodLoadEntityString(l shared.Lump_t, name string, buf []byte) error {
	// if (sv_entfile->value) {
	// 	char s[MAX_QPATH];
	// 	char *buffer = NULL;
	// 	int nameLen, bufLen;

	// 	nameLen = strlen(name);
	// 	strcpy(s, name);
	// 	s[nameLen-3] = 'e';	s[nameLen-2] = 'n';	s[nameLen-1] = 't';
	// 	bufLen = FS_LoadFile(s, (void **)&buffer);

	// 	if (buffer != NULL && bufLen > 1)
	// 	{
	// 		if (bufLen + 1 > sizeof(map_entitystring))
	// 		{
	// 			Com_Printf("CMod_LoadEntityString: .ent file %s too large: %i > %lu.\n", s, bufLen, (unsigned long)sizeof(map_entitystring));
	// 			FS_FreeFile(buffer);
	// 		}
	// 		else
	// 		{
	// 			Com_Printf ("CMod_LoadEntityString: .ent file %s loaded.\n", s);
	// 			numentitychars = bufLen;
	// 			memcpy(map_entitystring, buffer, bufLen);
	// 			map_entitystring[bufLen] = 0; /* jit entity bug - null terminate the entity string! */
	// 			FS_FreeFile(buffer);
	// 			return;
	// 		}
	// 	}
	// 	else if (bufLen != -1)
	// 	{
	// 		/* If the .ent file is too small, don't load. */
	// 		Com_Printf("CMod_LoadEntityString: .ent file %s too small.\n", s);
	// 		FS_FreeFile(buffer);
	// 	}
	// }

	// numentitychars = l->filelen;
	// if (l.filelen + 1 > sizeof(map_entitystring)) {
	// 	Com_Error(ERR_DROP, "Map has too large entity lump");
	// }

	T.collision.map_entitystring = string(buf[l.Fileofs : l.Fileofs+l.Filelen])
	// memcpy(map_entitystring, cmod_base + l->fileofs, l->filelen);
	// map_entitystring[l->filelen] = 0;
	return nil
}

/*
 * Loads in the map and all submodels
 */
func (T *qCommon) CMLoadMap(name string, clientload bool, checksum *uint32) (*shared.Cmodel_t, error) {
	//  unsigned *buf;
	//  int i;
	//  dheader_t header;
	//  int length;
	//  static unsigned last_checksum;

	T.collision.map_noareas = T.Cvar_Get("map_noareas", "0", 0)

	if T.collision.map_name == name && (clientload || !T.Cvar_VariableBool("flushmap")) {
		// 	 *checksum = last_checksum;

		if !clientload {
			for i := range T.collision.portalopen {
				T.collision.portalopen[i] = false
			}
			T.floodAreaConnections()
		}

		return &T.collision.map_cmodels[0], nil /* still have the right version */
	}

	/* free old stuff */
	T.collision.numplanes = 0
	T.collision.numnodes = 0
	T.collision.numleafs = 0
	T.collision.numcmodels = 0
	T.collision.numvisibility = 0
	T.collision.numentitychars = 0
	T.collision.map_entitystring = ""
	T.collision.map_name = ""

	if len(name) == 0 {
		T.collision.numleafs = 1
		T.collision.numclusters = 1
		T.collision.numareas = 1
		*checksum = 0
		return &T.collision.map_cmodels[0], nil /* cinematic servers won't have anything at all */
	}

	buf, err := T.fs.LoadFile(name)
	if err != nil {
		return nil, err
	}
	if buf == nil {
		return nil, T.Com_Error(shared.ERR_DROP, "Couldn't load %s", name)
	}

	//  last_checksum = LittleLong(Com_BlockChecksum(buf, length));
	//  *checksum = last_checksum;

	header := shared.DheaderCreate(buf)
	if header.Version != shared.BSPVERSION {
		return nil, T.Com_Error(shared.ERR_DROP,
			"CMod_LoadBrushModel: %s has wrong version number (%v should be %v)",
			name, header.Version, shared.BSPVERSION)
	}

	//  cmod_base = (byte *)buf;

	/* load into heap */
	if err := T.cmodLoadSurfaces(header.Lumps[shared.LUMP_TEXINFO], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadLeafs(header.Lumps[shared.LUMP_LEAFS], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadLeafBrushes(header.Lumps[shared.LUMP_LEAFBRUSHES], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadPlanes(header.Lumps[shared.LUMP_PLANES], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadBrushes(header.Lumps[shared.LUMP_BRUSHES], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadBrushSides(header.Lumps[shared.LUMP_BRUSHSIDES], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadSubmodels(header.Lumps[shared.LUMP_MODELS], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadNodes(header.Lumps[shared.LUMP_NODES], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadAreas(header.Lumps[shared.LUMP_AREAS], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadAreaPortals(header.Lumps[shared.LUMP_AREAPORTALS], name, buf); err != nil {
		return nil, err
	}
	if err := T.cmodLoadVisibility(header.Lumps[shared.LUMP_VISIBILITY], name, buf); err != nil {
		return nil, err
	}
	/* From kmquake2: adding an extra parameter for .ent support. */
	if err := T.cmodLoadEntityString(header.Lumps[shared.LUMP_ENTITIES], name, buf); err != nil {
		return nil, err
	}

	//  FS_FreeFile(buf);

	if err := T.initBoxHull(); err != nil {
		return nil, err
	}

	for i := range T.collision.portalopen {
		T.collision.portalopen[i] = false
	}
	T.floodAreaConnections()

	T.collision.map_name = name
	return &T.collision.map_cmodels[0], nil
}

func (T *qCommon) CMInlineModel(name string) (*shared.Cmodel_t, error) {

	if len(name) == 0 || (name[0] != '*') {
		return nil, T.Com_Error(shared.ERR_DROP, "CM_InlineModel: bad name")
	}

	num, err := strconv.ParseInt(name[1:], 10, 32)
	if err != nil || (num < 1) || (int(num) >= T.collision.numcmodels) {
		return nil, T.Com_Error(shared.ERR_DROP, "CM_InlineModel: bad number")
	}

	return &T.collision.map_cmodels[num], nil
}

func (T *qCommon) CMLeafCluster(leafnum int) int {
	if (leafnum < 0) || (leafnum >= T.collision.numleafs) {
		T.Com_Error(shared.ERR_DROP, "CM_LeafCluster: bad number")
		return -1
	}

	return T.collision.map_leafs[leafnum].cluster
}

func (T *qCommon) CMLeafArea(leafnum int) int {
	if (leafnum < 0) || (leafnum >= T.collision.numleafs) {
		T.Com_Error(shared.ERR_DROP, "CM_LeafArea: bad number")
		return -1
	}

	return T.collision.map_leafs[leafnum].area
}

func (T *qCommon) CMNumClusters() int {
	return T.collision.numclusters
}

func (T *qCommon) CMNumInlineModels() int {
	return T.collision.numcmodels
}

func (T *qCommon) CMEntityString() string {
	return T.collision.map_entitystring
}

func (T *qCommon) cmDecompressVis(in, out []byte) {
	// int c;
	// byte *out_p;
	// int row;

	row := (T.collision.numclusters + 7) >> 3
	// out_p = out
	in_i := 0
	out_i := 0

	if in == nil || T.collision.numvisibility == 0 {
		/* no vis info, so make all visible */
		for row > 0 {
			out[out_i] = 0xff
			out_i++
			row--
		}

		return
	}

	for out_i < row {
		if in[in_i] != 0 {
			out[out_i] = in[in_i]
			out_i++
			in_i++
			continue
		}

		c := in[in_i+1]
		in_i += 2

		if out_i+int(c) > row {
			c = byte(row - out_i)
			log.Printf("warning: Vis decompression overrun\n")
		}

		for c > 0 {
			out[out_i] = 0
			out_i++
			c--
		}
	}
}

func (T *qCommon) CMClusterPVS(cluster int) []byte {
	if cluster == -1 {
		for i := 0; i < (T.collision.numclusters+7)>>3; i++ {
			T.collision.pvsrow[i] = 0
		}
	} else {
		T.cmDecompressVis(T.collision.map_visibility[T.collision.map_vis.Bitofs[cluster][shared.DVIS_PVS]:], T.collision.pvsrow[:])
	}

	return T.collision.pvsrow[:]
}

func (T *qCommon) CMClusterPHS(cluster int) []byte {
	if cluster == -1 {
		for i := 0; i < (T.collision.numclusters+7)>>3; i++ {
			T.collision.phsrow[i] = 0
		}
	} else {
		T.cmDecompressVis(T.collision.map_visibility[T.collision.map_vis.Bitofs[cluster][shared.DVIS_PHS]:], T.collision.phsrow[:])
	}

	return T.collision.phsrow[:]
}
