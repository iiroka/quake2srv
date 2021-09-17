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
 *  The prototypes for most file formats used by Quake II
 *
 * =======================================================================
 */
package shared

import "log"

type QFileHandle interface {
	Close()
	Read(len int) []byte
}

/* .MD2 triangle model file format */

const IDALIASHEADER = (('2' << 24) + ('P' << 16) + ('D' << 8) + 'I')
const ALIAS_VERSION = 8

const (
	MAX_TRIANGLES = 4096
	MAX_VERTS     = 2048
	MAX_FRAMES    = 512
	MAX_MD2SKINS  = 32
	MAX_SKINNAME  = 64
)

type Dstvert_t struct {
	S, T int16
}

const Dstvert_size = 2 * 2

func Dstvert(data []byte) Dstvert_t {
	d := Dstvert_t{}
	d.S = ReadInt16(data[0*2:])
	d.T = ReadInt16(data[1*2:])
	return d
}

type Dtriangle_t struct {
	Index_xyz [3]int16
	Index_st  [3]int16
}

const Dtriangle_size = 6 * 4

func Dtriangle(data []byte) Dtriangle_t {
	d := Dtriangle_t{}
	for i := 0; i < 3; i++ {
		d.Index_xyz[i] = ReadInt16(data[i*2:])
		d.Index_st[i] = ReadInt16(data[(i+3)*2:])
	}
	return d
}

type Dtrivertx_t struct {
	V                [3]byte /* scaled byte to fit in frame mins/maxs */
	Lightnormalindex byte
}

const Dtrivertx_size = 4

func Dtrivertx(data []byte) Dtrivertx_t {
	d := Dtrivertx_t{}
	for i := 0; i < 3; i++ {
		d.V[i] = data[i]
	}
	d.Lightnormalindex = data[3]
	return d
}

// #define DTRIVERTX_V0 0
// #define DTRIVERTX_V1 1
// #define DTRIVERTX_V2 2
// #define DTRIVERTX_LNI 3
// #define DTRIVERTX_SIZE 4

type Daliasframe_t struct {
	Scale     [3]float32    /* multiply byte verts by this */
	Translate [3]float32    /* then add this */
	Name      string        /* frame name from grabbing */
	Verts     []Dtrivertx_t /* variable sized */
}

const daliasframe_size = 6*4 + 16

func Daliasframe(data []byte, framesize int) Daliasframe_t {
	d := Daliasframe_t{}
	for i := 0; i < 3; i++ {
		d.Scale[i] = ReadFloat32(data[i*4:])
		d.Translate[i] = ReadFloat32(data[(3+i)*4:])
	}
	d.Name = ReadString(data[6*4:], 16)
	size := (framesize - daliasframe_size)
	if (size % Dtrivertx_size) != 0 {
		log.Fatal("Aliasframe size is wrong")
	}
	d.Verts = make([]Dtrivertx_t, size/Dtrivertx_size)
	for i := range d.Verts {
		d.Verts[i] = Dtrivertx(data[daliasframe_size+i*Dtrivertx_size:])
	}
	return d
}

// /* the glcmd format:
//  * - a positive integer starts a tristrip command, followed by that many
//  *   vertex structures.
//  * - a negative integer starts a trifan command, followed by -x vertexes
//  *   a zero indicates the end of the command list.
//  * - a vertex consists of a floating point s, a floating point t,
//  *   and an integer vertex index. */

type Dmdl_t struct {
	Ident   int32
	Version int32

	Skinwidth  int32
	Skinheight int32
	Framesize  int32 /* byte size of each frame */

	Num_skins  int32
	Num_xyz    int32
	Num_st     int32 /* greater than num_xyz for seams */
	Num_tris   int32
	Num_glcmds int32 /* dwords in strip/fan command list */
	Num_frames int32

	Ofs_skins  int32 /* each skin is a MAX_SKINNAME string */
	Ofs_st     int32 /* byte offset from start for stverts */
	Ofs_tris   int32 /* offset for dtriangles */
	Ofs_frames int32 /* offset for first frame */
	Ofs_glcmds int32
	Ofs_end    int32 /* end of file */
}

const Dmdl_size = 17 * 4

func Dmdl(data []byte) Dmdl_t {
	d := Dmdl_t{}
	d.Ident = ReadInt32(data[0*4:])
	d.Version = ReadInt32(data[1*4:])

	d.Skinwidth = ReadInt32(data[2*4:])
	d.Skinheight = ReadInt32(data[3*4:])
	d.Framesize = ReadInt32(data[4*4:])

	d.Num_skins = ReadInt32(data[5*4:])
	d.Num_xyz = ReadInt32(data[6*4:])
	d.Num_st = ReadInt32(data[7*4:])
	d.Num_tris = ReadInt32(data[8*4:])
	d.Num_glcmds = ReadInt32(data[9*4:])
	d.Num_frames = ReadInt32(data[10*4:])

	d.Ofs_skins = ReadInt32(data[11*4:])
	d.Ofs_st = ReadInt32(data[12*4:])
	d.Ofs_tris = ReadInt32(data[13*4:])
	d.Ofs_frames = ReadInt32(data[14*4:])
	d.Ofs_glcmds = ReadInt32(data[15*4:])
	d.Ofs_end = ReadInt32(data[16*4:])
	return d
}

/* .SP2 sprite file format */

const IDSPRITEHEADER = (('2' << 24) + ('S' << 16) + ('D' << 8) + 'I') /* little-endian "IDS2" */
const SPRITE_VERSION = 2

type Dsprframe_t struct {
	Width, Height      int32
	Origin_x, Origin_y int32  /* raster coordinates inside pic */
	Name               string /* name of pcx file */
}

const Dsprframe_size = 4*4 + MAX_SKINNAME

func Dsprframe(data []byte) Dsprframe_t {
	d := Dsprframe_t{}
	d.Width = ReadInt32(data[0*4:])
	d.Height = ReadInt32(data[1*4:])
	d.Origin_x = ReadInt32(data[2*4:])
	d.Origin_y = ReadInt32(data[3*4:])
	d.Name = ReadString(data[4*4:], MAX_SKINNAME)
	return d
}

type Dsprite_t struct {
	Ident     int32
	Version   int32
	Numframes int32
	Frames    []Dsprframe_t /* variable sized */
}

func Dsprite(data []byte) Dsprite_t {
	d := Dsprite_t{}
	d.Ident = ReadInt32(data[0*4:])
	d.Version = ReadInt32(data[1*4:])
	d.Numframes = ReadInt32(data[2*4:])
	d.Frames = make([]Dsprframe_t, d.Numframes)
	for i := range d.Frames {
		d.Frames[i] = Dsprframe(data[3*4+i*Dsprframe_size:])
	}
	return d
}

/* .WAL texture file format */

const MIPLEVELS = 4

type Miptex_t struct {
	Name     string
	Width    uint32
	Height   uint32
	Offsets  [MIPLEVELS]uint32 /* four mip maps stored */
	Animname string            /* next frame in animation chain */
	Flags    int32
	Contents int32
}

func Miptex(data []byte) Miptex_t {
	d := Miptex_t{}
	d.Name = ReadString(data, 32)
	d.Width = ReadUint32(data[32:])
	d.Height = ReadUint32(data[32+4:])
	for i := 0; i < MIPLEVELS; i++ {
		d.Offsets[i] = ReadUint32(data[32+(2+i)*4:])
	}
	d.Animname = ReadString(data[32+(2+MIPLEVELS)*4:], 32)
	d.Flags = ReadInt32(data[2*32+(2+MIPLEVELS)*4:])
	d.Contents = ReadInt32(data[2*32+(3+MIPLEVELS)*4:])
	return d
}

/* .BSP file format */

const IDBSPHEADER = (('P' << 24) + ('S' << 16) + ('B' << 8) + 'I') /* little-endian "IBSP" */
const BSPVERSION = 38

/* upper design bounds: leaffaces, leafbrushes, planes, and
 * verts are still bounded by 16 bit short limits */
const (
	MAX_MAP_MODELS    = 1024
	MAX_MAP_BRUSHES   = 8192
	MAX_MAP_ENTITIES  = 2048
	MAX_MAP_ENTSTRING = 0x40000
	MAX_MAP_TEXINFO   = 8192

	MAX_MAP_AREAS       = 256
	MAX_MAP_AREAPORTALS = 1024
	MAX_MAP_PLANES      = 65536
	MAX_MAP_NODES       = 65536
	MAX_MAP_BRUSHSIDES  = 65536
	MAX_MAP_LEAFS       = 65536
	MAX_MAP_VERTS       = 65536
	MAX_MAP_FACES       = 65536
	MAX_MAP_LEAFFACES   = 65536
	MAX_MAP_LEAFBRUSHES = 65536
	MAX_MAP_PORTALS     = 65536
	MAX_MAP_EDGES       = 128000
	MAX_MAP_SURFEDGES   = 256000
	MAX_MAP_LIGHTING    = 0x200000
	MAX_MAP_VISIBILITY  = 0x100000

	/* key / value pair sizes */

	MAX_KEY   = 32
	MAX_VALUE = 1024
)

/* ================================================================== */

type Lump_t struct {
	Fileofs int32
	Filelen int32
}

const Lump_size = 2 * 4

const (
	LUMP_ENTITIES    = 0
	LUMP_PLANES      = 1
	LUMP_VERTEXES    = 2
	LUMP_VISIBILITY  = 3
	LUMP_NODES       = 4
	LUMP_TEXINFO     = 5
	LUMP_FACES       = 6
	LUMP_LIGHTING    = 7
	LUMP_LEAFS       = 8
	LUMP_LEAFFACES   = 9
	LUMP_LEAFBRUSHES = 10
	LUMP_EDGES       = 11
	LUMP_SURFEDGES   = 12
	LUMP_MODELS      = 13
	LUMP_BRUSHES     = 14
	LUMP_BRUSHSIDES  = 15
	LUMP_POP         = 16
	LUMP_AREAS       = 17
	LUMP_AREAPORTALS = 18
	HEADER_LUMPS     = 19
)

type Dheader_t struct {
	Ident   int32
	Version int32
	Lumps   []Lump_t
}

const Dheader_size = 2*4 + HEADER_LUMPS*Lump_size

func DheaderCreate(data []byte) Dheader_t {
	d := Dheader_t{}
	d.Ident = ReadInt32(data[0:])
	d.Version = ReadInt32(data[4:])
	d.Lumps = make([]Lump_t, HEADER_LUMPS)
	for i := 0; i < HEADER_LUMPS; i++ {
		d.Lumps[i].Fileofs = ReadInt32(data[2*4+i*Lump_size:])
		d.Lumps[i].Filelen = ReadInt32(data[2*4+i*Lump_size+4:])
	}
	return d
}

type Dmodel_t struct {
	Mins                [3]float32
	Maxs                [3]float32
	Origin              [3]float32 /* for sounds or lights */
	Headnode            int32
	Firstface, Numfaces int32 /* submodels just draw faces without
	walking the bsp tree */
}

const Dmodel_size = 12 * 4

func Dmodel(data []byte) Dmodel_t {
	d := Dmodel_t{}
	for i := 0; i < 3; i++ {
		d.Mins[i] = ReadFloat32(data[i*4:])
		d.Maxs[i] = ReadFloat32(data[(3+i)*4:])
		d.Origin[i] = ReadFloat32(data[(6+i)*4:])
	}
	d.Headnode = ReadInt32(data[9*4:])
	d.Firstface = ReadInt32(data[10*4:])
	d.Numfaces = ReadInt32(data[11*4:])
	return d
}

type Dvertex_t struct {
	Point [3]float32
}

const Dvertex_size = 3 * 4

func Dvertex(data []byte) Dvertex_t {
	d := Dvertex_t{}
	for i := 0; i < 3; i++ {
		d.Point[i] = ReadFloat32(data[i*4:])
	}
	return d
}

/* 0-2 are axial planes */
const PLANE_X = 0
const PLANE_Y = 1
const PLANE_Z = 2

/* 3-5 are non-axial planes snapped to the nearest */
const PLANE_ANYX = 3
const PLANE_ANYY = 4
const PLANE_ANYZ = 5

/* planes (x&~1) and (x&~1)+1 are always opposites */

type Dplane_t struct {
	Normal [3]float32
	Dist   float32
	Type   int32 /* PLANE_X - PLANE_ANYZ */
}

const Dplane_size = 5 * 4

func Dplane(data []byte) Dplane_t {
	d := Dplane_t{}
	for i := 0; i < 3; i++ {
		d.Normal[i] = ReadFloat32(data[i*4:])
	}
	d.Dist = ReadFloat32(data[3*4:])
	d.Type = ReadInt32(data[4*4:])
	return d
}

/* contents flags are seperate bits
 * - given brush can contribute multiple content bits
 * - multiple brushes can be in a single leaf */

const (
	/* lower bits are stronger, and will eat weaker brushes completely */
	CONTENTS_SOLID        = 1 /* an eye is never valid in a solid */
	CONTENTS_WINDOW       = 2 /* translucent, but not watery */
	CONTENTS_AUX          = 4
	CONTENTS_LAVA         = 8
	CONTENTS_SLIME        = 16
	CONTENTS_WATER        = 32
	CONTENTS_MIST         = 64
	LAST_VISIBLE_CONTENTS = 64

	/* remaining contents are non-visible, and don't eat brushes */
	CONTENTS_AREAPORTAL = 0x8000

	CONTENTS_PLAYERCLIP  = 0x10000
	CONTENTS_MONSTERCLIP = 0x20000

	/* currents can be added to any other contents, and may be mixed */
	CONTENTS_CURRENT_0    = 0x40000
	CONTENTS_CURRENT_90   = 0x80000
	CONTENTS_CURRENT_180  = 0x100000
	CONTENTS_CURRENT_270  = 0x200000
	CONTENTS_CURRENT_UP   = 0x400000
	CONTENTS_CURRENT_DOWN = 0x800000

	CONTENTS_ORIGIN = 0x1000000 /* removed before bsping an entity */

	CONTENTS_MONSTER     = 0x2000000 /* should never be on a brush, only in game */
	CONTENTS_DEADMONSTER = 0x4000000
	CONTENTS_DETAIL      = 0x8000000  /* brushes to be added after vis leafs */
	CONTENTS_TRANSLUCENT = 0x10000000 /* auto set if any surface has trans */
	CONTENTS_LADDER      = 0x20000000

	SURF_LIGHT = 0x1 /* value will hold the light strength */

	SURF_SLICK = 0x2 /* effects game physics */

	SURF_SKY     = 0x4 /* don't draw, but add to skybox */
	SURF_WARP    = 0x8 /* turbulent water warp */
	SURF_TRANS33 = 0x10
	SURF_TRANS66 = 0x20
	SURF_FLOWING = 0x40 /* scroll towards angle */
	SURF_NODRAW  = 0x80 /* don't bother referencing the texture */
)

type Dnode_t struct {
	Planenum  int32
	Children  [2]int32 /* negative numbers are -(leafs+1), not nodes */
	Mins      [3]int16 /* for frustom culling */
	Maxs      [3]int16
	Firstface uint16
	Numfaces  uint16 /* counting both sides */
}

const Dnode_size = 3*4 + 8*2

func Dnode(data []byte) Dnode_t {
	d := Dnode_t{}
	d.Planenum = ReadInt32(data[0:])
	d.Children[0] = ReadInt32(data[1*4:])
	d.Children[1] = ReadInt32(data[2*4:])
	for i := 0; i < 3; i++ {
		d.Mins[i] = ReadInt16(data[3*4+i*2:])
		d.Maxs[i] = ReadInt16(data[3*4+(3+i)*2:])
	}
	d.Firstface = ReadUint16(data[3*4+6*2:])
	d.Numfaces = ReadUint16(data[3*4+7*2:])
	return d
}

type Texinfo_t struct {
	Vecs        [2][4]float32 /* [s/t][xyz offset] */
	Flags       int32         /* miptex flags + overrides light emission, etc */
	Value       int32
	Texture     string /* texture name (textures*.wal) */
	Nexttexinfo int32  /* for animations, -1 = end of chain */
}

func Texinfo(data []byte) Texinfo_t {
	d := Texinfo_t{}
	for i := 0; i < 2; i++ {
		for j := 0; j < 4; j++ {
			d.Vecs[i][j] = ReadFloat32(data[(j+(i*4))*4:])
		}
	}
	d.Flags = ReadInt32(data[8*4:])
	d.Value = ReadInt32(data[9*4:])
	d.Texture = ReadString(data[10*4:], 32)
	d.Nexttexinfo = ReadInt32(data[10*4+32:])
	return d
}

const Texinfo_size = 11*4 + 32

/* note that edge 0 is never used, because negative edge
nums are used for counterclockwise use of the edge in
a face */
type Dedge_t struct {
	V [2]uint16 /* vertex numbers */
}

const Dedge_size = 2 * 2

func Dedge(data []byte) Dedge_t {
	d := Dedge_t{}
	d.V[0] = ReadUint16(data[0:])
	d.V[1] = ReadUint16(data[2:])
	return d
}

const MAXLIGHTMAPS = 4

type Dface_t struct {
	Planenum uint16
	Side     int16

	Firstedge int32 /* we must support > 64k edges */
	Numedges  int16
	Texinfo   int16

	/* lighting info */
	Styles   [MAXLIGHTMAPS]byte
	Lightofs int32 /* start of [numstyles*surfsize] samples */
}

const Dface_size = 4*2 + 2*4 + MAXLIGHTMAPS

func Dface(data []byte) Dface_t {
	d := Dface_t{}
	d.Planenum = ReadUint16(data[0*2:])
	d.Side = ReadInt16(data[1*2:])
	d.Firstedge = ReadInt32(data[2*2:])
	d.Numedges = ReadInt16(data[2*2+4:])
	d.Texinfo = ReadInt16(data[3*2+4:])
	copy(d.Styles[:], data[4*2+4:])
	d.Lightofs = ReadInt32(data[4*2+4+MAXLIGHTMAPS:])
	return d
}

type Dleaf_t struct {
	Contents int32 /* OR of all brushes (not needed?) */

	Cluster int16
	Area    int16

	Mins [3]int16 /* for frustum culling */
	Maxs [3]int16

	Firstleafface uint16
	Numleaffaces  uint16

	Firstleafbrush uint16
	Numleafbrushes uint16
}

const Dleaf_size = 4 + 12*2

func Dleaf(data []byte) Dleaf_t {
	d := Dleaf_t{}
	d.Contents = ReadInt32(data[0:])
	d.Cluster = ReadInt16(data[4:])
	d.Area = ReadInt16(data[4+2:])
	for i := 0; i < 3; i++ {
		d.Mins[i] = ReadInt16(data[4+(i+2)*2:])
		d.Maxs[i] = ReadInt16(data[4+(3+2+i)*2:])
	}
	d.Firstleafface = ReadUint16(data[4+8*2:])
	d.Numleaffaces = ReadUint16(data[4+9*2:])
	d.Firstleafbrush = ReadUint16(data[4+10*2:])
	d.Numleafbrushes = ReadUint16(data[4+11*2:])
	return d
}

type Dbrushside_t struct {
	Planenum uint16 /* facing out of the leaf */
	Texinfo  int16
}

const Dbrushside_size = 2 * 2

func Dbrushside(data []byte) Dbrushside_t {
	d := Dbrushside_t{}
	d.Planenum = ReadUint16(data[0:])
	d.Texinfo = ReadInt16(data[2:])
	return d
}

type Dbrush_t struct {
	Firstside int32
	Numsides  int32
	Contents  int32
}

const Dbrush_size = 3 * 4

func Dbrush(data []byte) Dbrush_t {
	d := Dbrush_t{}
	d.Firstside = ReadInt32(data[0:])
	d.Numsides = ReadInt32(data[4:])
	d.Contents = ReadInt32(data[2*4:])
	return d
}

const ANGLE_UP = -1
const ANGLE_DOWN = -2

/* the visibility lump consists of a header with a count, then
 * byte offsets for the PVS and PHS of each cluster, then the raw
 * compressed bit vectors */
const DVIS_PVS = 0
const DVIS_PHS = 1

type Dvis_t struct {
	Numclusters int32
	Bitofs      [][2]int32
}

func Dvis(data []byte) *Dvis_t {
	d := &Dvis_t{}
	d.Numclusters = ReadInt32(data)
	d.Bitofs = make([][2]int32, d.Numclusters)
	for i := 0; i < int(d.Numclusters); i++ {
		d.Bitofs[i][0] = ReadInt32(data[(1+2*i)*4:])
		d.Bitofs[i][1] = ReadInt32(data[(2+2*i)*4:])
	}
	return d
}

/* each area has a list of portals that lead into other areas
 * when portals are closed, other areas may not be visible or
 * hearable even if the vis info says that it should be */
type Dareaportal_t struct {
	Portalnum int32
	Otherarea int32
}

func Dareaportal(data []byte) Dareaportal_t {
	d := Dareaportal_t{}
	d.Portalnum = ReadInt32(data[0:])
	d.Otherarea = ReadInt32(data[4:])
	return d
}

const Dareaportal_size = 2 * 4

type Darea_t struct {
	Numareaportals  int32
	Firstareaportal int32
}

const Darea_size = 2 * 4

func Darea(data []byte) Darea_t {
	d := Darea_t{}
	d.Numareaportals = ReadInt32(data[0:])
	d.Firstareaportal = ReadInt32(data[4:])
	return d
}
