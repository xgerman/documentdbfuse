package fuse

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	fsops "github.com/xgerman/documentdbfuse/internal/documentdbfuse/fs"
)

var (
	entryTimeout = 1 * time.Second
	attrTimeout  = 1 * time.Second
)

// ---------------------------------------------------------------------------
// Root node — represents the mount root, lists databases
// ---------------------------------------------------------------------------

// Root is the top-level FUSE inode. It holds a reference to the
// filesystem operations layer that talks to MongoDB.
type Root struct {
	fs.Inode
	ops *fsops.Operations
}

var _ = (fs.NodeLookuper)((*Root)(nil))
var _ = (fs.NodeReaddirer)((*Root)(nil))

func (r *Root) path() string { return "/" }

func (r *Root) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	entries, err := r.ops.ReadDir(ctx, r.path())
	if err != nil {
		return nil, syscall.EIO
	}
	out := make([]fuse.DirEntry, len(entries))
	for i, name := range entries {
		out[i] = fuse.DirEntry{Name: name, Mode: syscall.S_IFDIR}
	}
	return fs.NewListDirStream(out), fs.OK
}

func (r *Root) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	entries, err := r.ops.ReadDir(ctx, r.path())
	if err != nil {
		return nil, syscall.EIO
	}
	for _, e := range entries {
		if e == name {
			child := &DatabaseNode{ops: r.ops, dbName: name}
			out.Mode = syscall.S_IFDIR | 0755
			out.SetEntryTimeout(entryTimeout)
			out.SetAttrTimeout(attrTimeout)
			stable := fs.StableAttr{Mode: syscall.S_IFDIR}
			return r.NewInode(ctx, child, stable), fs.OK
		}
	}
	return nil, syscall.ENOENT
}

// ---------------------------------------------------------------------------
// DatabaseNode — a database directory, lists collections
// ---------------------------------------------------------------------------

type DatabaseNode struct {
	fs.Inode
	ops    *fsops.Operations
	dbName string
}

var _ = (fs.NodeLookuper)((*DatabaseNode)(nil))
var _ = (fs.NodeReaddirer)((*DatabaseNode)(nil))
var _ = (fs.NodeMkdirer)((*DatabaseNode)(nil))

func (d *DatabaseNode) path() string { return "/" + d.dbName }

func (d *DatabaseNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	entries, err := d.ops.ReadDir(ctx, d.path())
	if err != nil {
		return nil, syscall.EIO
	}
	out := make([]fuse.DirEntry, len(entries))
	for i, name := range entries {
		out[i] = fuse.DirEntry{Name: name, Mode: syscall.S_IFDIR}
	}
	return fs.NewListDirStream(out), fs.OK
}

func (d *DatabaseNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	entries, err := d.ops.ReadDir(ctx, d.path())
	if err != nil {
		return nil, syscall.EIO
	}
	for _, e := range entries {
		if e == name {
			child := &CollectionNode{ops: d.ops, dbName: d.dbName, collName: name}
			out.Mode = syscall.S_IFDIR | 0755
			out.SetEntryTimeout(entryTimeout)
			out.SetAttrTimeout(attrTimeout)
			stable := fs.StableAttr{Mode: syscall.S_IFDIR}
			return d.NewInode(ctx, child, stable), fs.OK
		}
	}
	return nil, syscall.ENOENT
}

func (d *DatabaseNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	p := d.path() + "/" + name
	if err := d.ops.MkDir(ctx, p); err != nil {
		return nil, syscall.EIO
	}
	child := &CollectionNode{ops: d.ops, dbName: d.dbName, collName: name}
	out.Mode = syscall.S_IFDIR | 0755
	out.SetEntryTimeout(entryTimeout)
	out.SetAttrTimeout(attrTimeout)
	stable := fs.StableAttr{Mode: syscall.S_IFDIR}
	return d.NewInode(ctx, child, stable), fs.OK
}

// ---------------------------------------------------------------------------
// CollectionNode — a collection directory, lists documents as .json files
// ---------------------------------------------------------------------------

type CollectionNode struct {
	fs.Inode
	ops      *fsops.Operations
	dbName   string
	collName string
}

var _ = (fs.NodeLookuper)((*CollectionNode)(nil))
var _ = (fs.NodeReaddirer)((*CollectionNode)(nil))
var _ = (fs.NodeCreater)((*CollectionNode)(nil))
var _ = (fs.NodeUnlinker)((*CollectionNode)(nil))
var _ = (fs.NodeRmdirer)((*CollectionNode)(nil))

func (c *CollectionNode) path() string {
	return "/" + c.dbName + "/" + c.collName
}

func (c *CollectionNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	entries, err := c.ops.ReadDir(ctx, c.path())
	if err != nil {
		return nil, syscall.EIO
	}
	out := make([]fuse.DirEntry, len(entries))
	for i, name := range entries {
		out[i] = fuse.DirEntry{Name: name, Mode: syscall.S_IFREG}
	}
	return fs.NewListDirStream(out), fs.OK
}

func (c *CollectionNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	// Pipeline segment — enter pipeline traversal mode
	if strings.HasPrefix(name, ".") {
		child := &PipelineNode{
			ops:          c.ops,
			dbName:       c.dbName,
			collName:     c.collName,
			pathSegments: []string{name},
		}
		out.Mode = syscall.S_IFDIR | 0755
		out.SetEntryTimeout(entryTimeout)
		out.SetAttrTimeout(attrTimeout)
		stable := fs.StableAttr{Mode: syscall.S_IFDIR}
		return c.NewInode(ctx, child, stable), fs.OK
	}

	filePath := c.path() + "/" + name
	data, err := c.ops.ReadFile(ctx, filePath)
	if err != nil {
		return nil, syscall.ENOENT
	}
	child := &DocumentNode{
		ops:      c.ops,
		dbName:   c.dbName,
		collName: c.collName,
		fileName: name,
	}
	out.Mode = syscall.S_IFREG | 0644
	out.Size = uint64(len(data))
	out.SetEntryTimeout(entryTimeout)
	out.SetAttrTimeout(0)
	stable := fs.StableAttr{Mode: syscall.S_IFREG}
	return c.NewInode(ctx, child, stable), fs.OK
}

func (c *CollectionNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (inode *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	// Don't insert yet — wait for the Write call with actual content.
	// The Write handler will upsert the document.
	child := &DocumentNode{
		ops:      c.ops,
		dbName:   c.dbName,
		collName: c.collName,
		fileName: name,
	}
	out.Mode = syscall.S_IFREG | 0644
	out.Size = 0
	out.SetEntryTimeout(entryTimeout)
	out.SetAttrTimeout(0)
	stable := fs.StableAttr{Mode: syscall.S_IFREG}
	return c.NewInode(ctx, child, stable), nil, 0, fs.OK
}

func (c *CollectionNode) Unlink(ctx context.Context, name string) syscall.Errno {
	filePath := c.path() + "/" + name
	if err := c.ops.Remove(ctx, filePath, false); err != nil {
		return syscall.EIO
	}
	return fs.OK
}

func (c *CollectionNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	// Rmdir on a collection node's child doesn't make sense (documents aren't dirs).
	return syscall.ENOTSUP
}

// ---------------------------------------------------------------------------
// DatabaseNode also supports Rmdir to drop collections listed under it.
// We already have Mkdir; Rmdir delegates to the operations layer.
// ---------------------------------------------------------------------------

var _ = (fs.NodeRmdirer)((*DatabaseNode)(nil))

func (d *DatabaseNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	p := d.path() + "/" + name
	if err := d.ops.Remove(ctx, p, true); err != nil {
		return syscall.EIO
	}
	return fs.OK
}

// ---------------------------------------------------------------------------
// DocumentNode — a single document, represented as a .json file
// ---------------------------------------------------------------------------

type DocumentNode struct {
	fs.Inode
	ops      *fsops.Operations
	dbName   string
	collName string
	fileName string
}

var _ = (fs.NodeOpener)((*DocumentNode)(nil))
var _ = (fs.NodeReader)((*DocumentNode)(nil))
var _ = (fs.NodeWriter)((*DocumentNode)(nil))
var _ = (fs.NodeGetattrer)((*DocumentNode)(nil))

func (d *DocumentNode) path() string {
	return "/" + d.dbName + "/" + d.collName + "/" + d.fileName
}

func (d *DocumentNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	data, err := d.ops.ReadFile(ctx, d.path())
	if err != nil {
		out.Mode = syscall.S_IFREG | 0644
		out.Size = 0
		return fs.OK
	}
	out.Mode = syscall.S_IFREG | 0644
	out.Size = uint64(len(data))
	out.SetTimeout(0) // always re-check content size
	return fs.OK
}

func (d *DocumentNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, 0, fs.OK
}

func (d *DocumentNode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	data, err := d.ops.ReadFile(ctx, d.path())
	if err != nil {
		return nil, syscall.EIO
	}
	// Append a trailing newline for convenient shell usage
	data = append(data, '\n')
	if off >= int64(len(data)) {
		return fuse.ReadResultData(nil), fs.OK
	}
	return fuse.ReadResultData(data[off:]), fs.OK
}

func (d *DocumentNode) Write(ctx context.Context, fh fs.FileHandle, data []byte, off int64) (uint32, syscall.Errno) {
	content := data
	// Trim trailing whitespace/newlines that editors often add
	content = []byte(strings.TrimRight(string(content), "\n\r "))
	if err := d.ops.WriteFile(ctx, d.path(), content); err != nil {
		return 0, syscall.EIO
	}
	return uint32(len(data)), fs.OK
}

// ---------------------------------------------------------------------------
// PipelineNode — virtual directory for aggregation pipeline traversal.
// Each path segment (.match/field/value, .sort/field, etc.) is a directory.
// The terminal .export/json becomes a readable file with query results.
// ---------------------------------------------------------------------------

type PipelineNode struct {
	fs.Inode
	ops          *fsops.Operations
	dbName       string
	collName     string
	pathSegments []string // accumulated pipeline path segments
}

var _ = (fs.NodeLookuper)((*PipelineNode)(nil))
var _ = (fs.NodeReaddirer)((*PipelineNode)(nil))
var _ = (fs.NodeOpener)((*PipelineNode)(nil))
var _ = (fs.NodeReader)((*PipelineNode)(nil))
var _ = (fs.NodeGetattrer)((*PipelineNode)(nil))

func (p *PipelineNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	// "results.json" — read current pipeline results as a file
	if name == "results.json" {
		child := &PipelineResultNode{
			ops:          p.ops,
			dbName:       p.dbName,
			collName:     p.collName,
			pathSegments: p.pathSegments,
		}
		out.Mode = syscall.S_IFREG | 0444
		out.Size = 4096
		out.SetEntryTimeout(0)
		out.SetAttrTimeout(0)
		stable := fs.StableAttr{Mode: syscall.S_IFREG}
		return p.NewInode(ctx, child, stable), fs.OK
	}

	newSegments := append(append([]string{}, p.pathSegments...), name)

	// .export/format terminal — return a readable file
	if len(newSegments) >= 2 && newSegments[len(newSegments)-2] == ".export" {
		child := &PipelineResultNode{
			ops:          p.ops,
			dbName:       p.dbName,
			collName:     p.collName,
			pathSegments: newSegments,
		}
		out.Mode = syscall.S_IFREG | 0444
		out.Size = 4096
		out.SetEntryTimeout(0)
		out.SetAttrTimeout(0)
		stable := fs.StableAttr{Mode: syscall.S_IFREG}
		return p.NewInode(ctx, child, stable), fs.OK
	}

	// Otherwise keep traversing as a directory
	child := &PipelineNode{
		ops:          p.ops,
		dbName:       p.dbName,
		collName:     p.collName,
		pathSegments: newSegments,
	}
	out.Mode = syscall.S_IFDIR | 0755
	out.SetEntryTimeout(entryTimeout)
	out.SetAttrTimeout(attrTimeout)
	stable := fs.StableAttr{Mode: syscall.S_IFDIR}
	return p.NewInode(ctx, child, stable), fs.OK
}

func (p *PipelineNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = syscall.S_IFDIR | 0755
	out.SetTimeout(attrTimeout)
	return fs.OK
}

func (p *PipelineNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, 0, fs.OK
}

func (p *PipelineNode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	// Reading a pipeline directory returns aggregation results as JSON
	fullPath := "/" + p.dbName + "/" + p.collName + "/" + strings.Join(p.pathSegments, "/")
	data, err := p.ops.ReadFile(ctx, fullPath)
	if err != nil {
		return nil, syscall.EIO
	}
	data = append(data, '\n')
	if off >= int64(len(data)) {
		return fuse.ReadResultData(nil), fs.OK
	}
	return fuse.ReadResultData(data[off:]), fs.OK
}

func (p *PipelineNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	// Try to parse the accumulated segments as a complete pipeline.
	// If it parses successfully, run the query and list matching document IDs.
	pipeline, err := fsops.ParsePipeline(p.pathSegments)
	if err == nil && len(pipeline.Stages) > 0 {
		fullPath := "/" + p.dbName + "/" + p.collName + "/" + strings.Join(p.pathSegments, "/")
		dirEntries, readErr := p.ops.ReadDir(ctx, fullPath)
		if readErr == nil && len(dirEntries) > 0 {
			out := make([]fuse.DirEntry, 0, len(dirEntries)+1)
			out = append(out, fuse.DirEntry{Name: "results.json", Mode: syscall.S_IFREG})
			for _, name := range dirEntries {
				out = append(out, fuse.DirEntry{Name: name, Mode: syscall.S_IFREG})
			}
			return fs.NewListDirStream(out), fs.OK
		}
	}

	// Incomplete pipeline or no results — show available segments
	entries := []fuse.DirEntry{
		{Name: "results.json", Mode: syscall.S_IFREG},
		{Name: ".match", Mode: syscall.S_IFDIR},
		{Name: ".sort", Mode: syscall.S_IFDIR},
		{Name: ".limit", Mode: syscall.S_IFDIR},
		{Name: ".skip", Mode: syscall.S_IFDIR},
		{Name: ".project", Mode: syscall.S_IFDIR},
		{Name: ".export", Mode: syscall.S_IFDIR},
	}
	return fs.NewListDirStream(entries), fs.OK
}

// ---------------------------------------------------------------------------
// PipelineResultNode — a readable file that executes the aggregation pipeline.
// ---------------------------------------------------------------------------

type PipelineResultNode struct {
	fs.Inode
	ops          *fsops.Operations
	dbName       string
	collName     string
	pathSegments []string
}

var _ = (fs.NodeOpener)((*PipelineResultNode)(nil))
var _ = (fs.NodeReader)((*PipelineResultNode)(nil))
var _ = (fs.NodeGetattrer)((*PipelineResultNode)(nil))

func (r *PipelineResultNode) fullPath() string {
	return "/" + r.dbName + "/" + r.collName + "/" + strings.Join(r.pathSegments, "/")
}

func (r *PipelineResultNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	// Report a large estimated size so the kernel calls Read.
	// The actual data is fetched fresh in Read().
	out.Mode = syscall.S_IFREG | 0444
	out.Size = 4096 // estimated; actual content comes from Read
	out.SetTimeout(0)
	return fs.OK
}

func (r *PipelineResultNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, 0, fs.OK
}

func (r *PipelineResultNode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	data, err := r.ops.ReadFile(ctx, r.fullPath())
	if err != nil {
		return nil, syscall.EIO
	}
	data = append(data, '\n')
	if off >= int64(len(data)) {
		return fuse.ReadResultData(nil), fs.OK
	}
	return fuse.ReadResultData(data[off:]), fs.OK
}

// ---------------------------------------------------------------------------
// Server creates and returns a FUSE server for the given mount point.
// ---------------------------------------------------------------------------

func Server(mountPoint string, ops *fsops.Operations, extraOpts ...string) (*fuse.Server, error) {
	root := &Root{ops: ops}

	opts := &fs.Options{
		MountOptions: fuse.MountOptions{
			FsName:        "documentdbfuse",
			Name:          "documentdbfuse",
			DisableXAttrs: true,
			Debug:         false,
			Options:       extraOpts,
		},
		EntryTimeout: &entryTimeout,
		AttrTimeout:  &attrTimeout,
	}

	server, err := fs.Mount(mountPoint, root, opts)
	if err != nil {
		return nil, fmt.Errorf("fuse mount failed: %w", err)
	}
	return server, nil
}
