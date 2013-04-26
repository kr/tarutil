// Experimental package tarutil provides utility functions for tar
// files. You should expect its interface to change.
package tarutil

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	Link = 1 << iota
	Symlink
	Chown
	Chmod
	Chtimes
)

// ExtractAll reads tar entries from r until EOF and creates
// filesystem entries rooted in root. It extracts everything it
// can but returns the first error it encounters. It cleans paths
// read from the tar file and will not create entries outside
// root.
//
// Behavior changes according to flag, bitwise-or of the
// following constants:
//
//   Link     attempt to create hard links
//   Symlink  attempt to create symlinks
//   Chown    attempt to set file owner and group
//   Chmod    attempt to set file mode
//   Chtimes  attempt to set atime and mtime
//
// If Chmod is unset, files are created with mode 0666 (subject to
// umask) and directories are created with mode 0777 (subject to
// umask).
//
// Flag Chown uses only uid and gid, ignoring user name and group
// name.
func ExtractAll(r io.Reader, root string, flag int) error {
	var err error
	tr := tar.NewReader(r)
	for {
		hdr, err1 := tr.Next()
		if err == nil && err1 != io.EOF {
			err = err1
		}
		if err1 != nil {
			break
		}
		err1 = extractOne(hdr, tr, root, flag)
		if err == nil {
			err = err1
		}
	}
	return err
}

func extractOne(hdr *tar.Header, r io.Reader, root string, flag int) error {
	// clean before joining to remove all .. elements
	path := filepath.Join(root, filepath.Clean(hdr.Name))
	targ := filepath.Join(root, filepath.Clean(hdr.Linkname))
	switch hdr.Typeflag {
	case tar.TypeReg, tar.TypeRegA:
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if _, err = io.Copy(f, r); err != nil {
			return err
		}
		if err = f.Close(); err != nil {
			return err
		}
	case tar.TypeLink:
		if flag&Link != 0 {
			if err := os.Link(targ, path); err != nil {
				return err
			}
		}
	case tar.TypeSymlink:
		if flag&Symlink != 0 {
			if err := os.Symlink(targ, path); err != nil {
				return err
			}
		}
	case tar.TypeDir:
		if err := os.MkdirAll(path, 0777); err != nil {
			return err
		}
	case tar.TypeCont, tar.TypeXHeader, tar.TypeXGlobalHeader:
		return nil
	case tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
		return fmt.Errorf("tarutil: unsupported type %q: %s", hdr.Typeflag, hdr.Name)
	}
	if flag&Chtimes != 0 {
		if err := os.Chtimes(path, hdr.AccessTime, hdr.ModTime); err != nil {
			return err
		}
	}
	if flag&Chmod != 0 {
		mode := os.FileMode(hdr.Mode)
		if err := os.Chmod(path, mode); err != nil {
			return err
		}
	}
	if flag&Chown != 0 {
		uid, gid := hdr.Uid, hdr.Gid
		if err := os.Lchown(path, uid, gid); err != nil {
			return err
		}
	}
	return nil
}
