// Package dav exposes the shared folder over WebDAV so it can be mounted as a
// network drive without administrator rights.
package dav

import (
	"context"
	"errors"
	"net/http"
	"os"

	"golang.org/x/net/webdav"

	"braise/internal/files"
)

// FS implements webdav.FileSystem on top of the confined files.Manager.
type FS struct {
	M *files.Manager
}

func mapErr(err error) error {
	if errors.Is(err, files.ErrDenied) {
		// Hide the metadata directory and other forbidden paths as if they
		// did not exist. The webdav package maps unknown Stat errors to 405.
		return os.ErrNotExist
	}
	return err
}

// OpenFile opens or creates a file, refusing paths outside the shared root and
// the .braise metadata directory.
func (f FS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	full, err := f.M.Resolve(name)
	if err != nil {
		return nil, mapErr(err)
	}
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND) != 0 && f.M.ReadOnly() {
		return nil, os.ErrPermission
	}
	if fi, err := os.Lstat(full); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			return nil, os.ErrPermission
		}
		if flag&(os.O_WRONLY|os.O_RDWR) != 0 && fi.IsDir() {
			return nil, os.ErrPermission
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	file, err := os.OpenFile(full, flag, perm)
	if err != nil {
		return nil, err
	}
	return davFile{File: file}, nil
}

// Mkdir creates one directory.
func (f FS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if f.M.ReadOnly() {
		return os.ErrPermission
	}
	full, err := f.M.Resolve(name)
	if err != nil {
		return mapErr(err)
	}
	return os.Mkdir(full, perm)
}

// RemoveAll removes a file or directory tree. The root is protected.
func (f FS) RemoveAll(ctx context.Context, name string) error {
	if f.M.ReadOnly() {
		return os.ErrPermission
	}
	full, err := f.M.Resolve(name)
	if err != nil {
		return mapErr(err)
	}
	if full == f.M.Root() {
		return os.ErrPermission
	}
	return os.RemoveAll(full)
}

// Rename moves a file or directory inside the root.
func (f FS) Rename(ctx context.Context, oldName, newName string) error {
	if f.M.ReadOnly() {
		return os.ErrPermission
	}
	from, err := f.M.Resolve(oldName)
	if err != nil {
		return mapErr(err)
	}
	to, err := f.M.Resolve(newName)
	if err != nil {
		return mapErr(err)
	}
	if from == f.M.Root() || to == f.M.Root() {
		return os.ErrPermission
	}
	return os.Rename(from, to)
}

// Stat returns file information.
func (f FS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	_, fi, err := f.M.ResolveExisting(name)
	if err != nil {
		return nil, mapErr(err)
	}
	return fi, nil
}

// davFile hides the metadata directory and symlinks from WebDAV directory
// listings, even though they live under the shared root.
type davFile struct {
	*os.File
}

func (f davFile) Readdir(count int) ([]os.FileInfo, error) {
	infos, err := f.File.Readdir(count)
	if len(infos) == 0 {
		return infos, err
	}
	out := infos[:0]
	for _, fi := range infos {
		if fi.Name() == ".braise" || fi.Mode()&os.ModeSymlink != 0 {
			continue
		}
		out = append(out, fi)
	}
	return out, err
}

// Handler returns the WebDAV HTTP handler mounted at prefix.
func Handler(m *files.Manager, prefix string, logger func(*http.Request, error)) http.Handler {
	return &webdav.Handler{
		Prefix:     prefix,
		FileSystem: FS{M: m},
		LockSystem: webdav.NewMemLS(),
		Logger:     logger,
	}
}
