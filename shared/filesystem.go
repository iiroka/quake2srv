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
 * The Quake II file system, implements generic file system operations
 * as well as the .pak file format and support for .pk3 files.
 *
 * =======================================================================
 */
package shared

import (
	"fmt"
	"log"
	"os"
	"strings"
)

/* The .pak files are just a linear collapse of a directory tree */

const IDPAKHEADER = (('K' << 24) + ('C' << 16) + ('A' << 8) + 'P')

type dpackfile_t struct {
	Name    string
	Filepos int32
	Filelen int32
}

func dpackFile(data []byte) dpackfile_t {
	r := dpackfile_t{}
	r.Name = ReadString(data, 56)
	r.Filepos = ReadInt32(data[56:])
	r.Filelen = ReadInt32(data[60:])
	return r
}

const dpackfile_size = 56 + 2*4

type dpackheader_t struct {
	Ident  int32 /* == IDPAKHEADER */
	Dirofs int32
	Dirlen int32
}

func dpackHeader(data []byte) dpackheader_t {
	r := dpackheader_t{}
	r.Ident = ReadInt32(data)
	r.Dirofs = ReadInt32(data[4:])
	r.Dirlen = ReadInt32(data[8:])
	return r
}

const dpackheader_size = 3 * 4

const MAX_FILES_IN_PACK = 4096
const MAX_HANDLES = 512

type fsPackFile_t struct {
	name   string
	size   int
	offset int64 /* Ignored in PK3 files. */
}

type fsPack_t struct {
	name string
	pak  *os.File
	// int numFiles;
	// FILE *pak;
	// unzFile *pk3;
	// qboolean isProtectedPak;
	files []fsPackFile_t
}

type fsSearchPath_t struct {
	path string    /* Only one used. */
	pack *fsPack_t /* (path or pack) */
}

const (
	maxHANDLES = 512
	maxMODS    = 32
	maxPAKS    = 100
)

type QFileSystem interface {
	LoadFile(path string) ([]byte, error)
}

type qFileSystem struct {
	fs_gamedir     string
	fs_debug       bool
	fs_searchPaths []fsSearchPath_t
}

/*
 * Filename are reletive to the quake search path. A null buffer will just
 * return the file length without loading.
 */
func (T *qFileSystem) LoadFile(path string) ([]byte, error) {
	// file_from_protected_pak = false;
	// handle = FS_HandleForFile(name, f);
	// Q_strlcpy(handle->name, name, sizeof(handle->name));
	// handle->mode = FS_READ;
	path = strings.ToLower(path)

	/* Search through the path, one element at a time. */
	for _, search := range T.fs_searchPaths {

		// Evil hack for maps.lst and players/
		// TODO: A flag to ignore paks would be better
		// if ((strcmp(fs_gamedirvar->string, "") == 0) && search->pack) {
		// 	if ((strcmp(name, "maps.lst") == 0)|| (strncmp(name, "players/", 8) == 0)) {
		// 		continue;
		// 	}
		// }

		/* Search inside a pack file. */
		if search.pack != nil {
			pack := search.pack

			for _, f := range pack.files {
				if f.name == path {
					/* Found it! */
					if T.fs_debug {
						log.Printf("FS_LoadFile: '%s' (found in '%s').\n", path, pack.name)
					}

					bfr := make([]byte, f.size)
					_, err := pack.pak.ReadAt(bfr, f.offset)
					if err != nil {
						log.Printf("FS_LoadFile: Failed to read from pack %v\n", err.Error())
						return nil, err
					}

					return bfr, nil
				}
			}
		} else {
			/* Search in a directory tree. */
			path := fmt.Sprintf("%v/%v", search.path, path)
			handle, err := os.Open(path)
			if err == nil {

				if T.fs_debug {
					log.Printf("FS_LoadFile: '%s' (found in '%s').\n", path, search.path)
				}

				st, _ := handle.Stat()
				size := st.Size()

				bfr := make([]byte, int(size))
				_, err := handle.Read(bfr)
				if err != nil {
					log.Printf("FS_LoadFile: Failed to read file %v\n", err.Error())
					return nil, err
				}

				handle.Close()
				return bfr, nil
			}

		}
	}
	// if T.fs_debug {
	log.Printf("FS_LoadFile: couldn't find '%s'.\n", path)
	// }
	return nil, nil
}

/*
 * Takes an explicit (not game tree related) path to a pak file.
 *
 * Loads the header and directory, adding the files at the beginning of the
 * list so they override previous pack files.
 */
func (T *qFileSystem) loadPAK(packPath string) (*fsPack_t, error) {
	//  int i; /* Loop counter. */
	//  int numFiles; /* Number of files in PAK. */
	//  FILE *handle; /* File handle. */
	//  fsPackFile_t *files; /* List of files in PAK. */
	//  fsPack_t *pack; /* PAK file. */
	//  dpackheader_t header; /* PAK file header. */
	//  dpackfile_t *info = NULL; /* PAK info. */

	handle, err := os.Open(packPath)
	if err != nil {
		return nil, nil
	}

	bfr := make([]byte, dpackheader_size)
	handle.Read(bfr)

	header := dpackHeader(bfr)
	if header.Ident != IDPAKHEADER {
		handle.Close()
		log.Fatalf("loadPAK: '%v' is not a pack file\n", packPath)
	}

	numFiles := header.Dirlen / dpackfile_size

	if (numFiles == 0) || (header.Dirlen < 0) || (header.Dirofs < 0) {
		handle.Close()
		log.Fatalf("loadPAK: '%v' is too short.", packPath)
	}

	if numFiles > MAX_FILES_IN_PACK {
		log.Printf("loadPAK: '%s' has %v > %v files\n",
			packPath, numFiles, MAX_FILES_IN_PACK)
	}

	bfr = make([]byte, header.Dirlen)

	files := make([]fsPackFile_t, numFiles)

	handle.ReadAt(bfr, int64(header.Dirofs))

	/* Parse the directory. */
	for i := 0; i < int(numFiles); i++ {
		info := dpackFile(bfr[i*dpackfile_size:])
		files[i].name = strings.ToLower(info.Name)
		files[i].size = int(info.Filelen)
		files[i].offset = int64(info.Filepos)
	}

	pack := fsPack_t{}
	pack.name = packPath
	pack.pak = handle
	pack.files = files

	log.Printf("Added packfile '%v' (%v files).\n", packPath, numFiles)

	return &pack, nil
}

func (T *qFileSystem) addDirToSearchPath(dir string, create bool) error {

	// Set the current directory as game directory. This
	// is somewhat fragile since the game directory MUST
	// be the last directory added to the search path.
	T.fs_gamedir = dir

	// 	if (create) {
	// 		FS_CreatePath(fs_gamedir);
	// 	}

	// Add the directory itself.
	search := fsSearchPath_t{}
	search.path = fmt.Sprintf("%v/%v", dir, BASEDIRNAME)
	T.fs_searchPaths = append(T.fs_searchPaths, search)

	foundFile := false

	// We need to add numbered paks in the directory in
	// sequence and all other paks after them. Otherwise
	// the gamedata may break.
	// 	for (i = 0; i < sizeof(fs_packtypes) / sizeof(fs_packtypes[0]); i++) {
	for j := 0; j < maxPAKS; j++ {
		path := fmt.Sprintf("%v/%v/pak%v.pak", dir, BASEDIRNAME, j)

		// 			switch (fs_packtypes[i].format)
		// 			{
		// 				case PAK:
		pack, err := T.loadPAK(path)
		if err != nil {
			return nil
		}

		// 					if (pack)
		// 					{
		// 						pack->isProtectedPak = true;
		// 					}

		// 					break;
		// 				case PK3:
		// 					pack = FS_LoadPK3(path);

		// 					if (pack)
		// 					{
		// 						pack->isProtectedPak = false;
		// 					}

		// 					break;
		// 			}

		// 			if (pack == NULL)
		// 			{
		// 				continue;
		// 			}

		if pack != nil {
			search = fsSearchPath_t{}
			search.pack = pack
			T.fs_searchPaths = append(T.fs_searchPaths, search)
			foundFile = true
		}
	}
	// 	}

	// 	// And as said above all other pak files.
	// 	for (i = 0; i < sizeof(fs_packtypes) / sizeof(fs_packtypes[0]); i++) {
	// 		Com_sprintf(path, sizeof(path), "%s/*.%s", dir, fs_packtypes[i].suffix);

	// 		// Nothing here, next pak type please.
	// 		if ((list = FS_ListFiles(path, &nfiles, 0, 0)) == NULL)
	// 		{
	// 			continue;
	// 		}

	// 		Com_sprintf(path, sizeof(path), "%s/pak*.%s", dir, fs_packtypes[i].suffix);

	// 		for (j = 0; j < nfiles - 1; j++)
	// 		{
	// 			// If the pak starts with the string 'pak' it's ignored.
	// 			// This is somewhat stupid, it would be better to ignore
	// 			// just pak%d...
	// 			if (glob_match(path, list[j]))
	// 			{
	// 				continue;
	// 			}

	// 			switch (fs_packtypes[i].format)
	// 			{
	// 				case PAK:
	// 					pack = FS_LoadPAK(list[j]);
	// 					break;
	// 				case PK3:
	// 					pack = FS_LoadPK3(list[j]);
	// 					break;
	// 			}

	// 			if (pack == NULL)
	// 			{
	// 				continue;
	// 			}

	// 			pack->isProtectedPak = false;

	// 			search = Z_Malloc(sizeof(fsSearchPath_t));
	// 			search->pack = pack;
	// 			search->next = fs_searchPaths;
	// 			fs_searchPaths = search;
	// 		}

	// 		FS_FreeList(list, nfiles);
	// 	}
	if !foundFile {
		log.Fatalf("%v does not seem to be correct Quake2 directory\n", dir)
	}
	return nil
}

// --------

func InitFilesystem(basepath string, debug bool) QFileSystem {
	q := &qFileSystem{}
	q.fs_debug = debug
	q.addDirToSearchPath(basepath, false)
	return q
}
