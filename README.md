# MongoFUSE

Mount any MongoDB-compatible database as a filesystem via FUSE (Linux) or NFS (macOS).

Browse collections with `ls`, read documents with `cat`, search with `grep`, write with `echo` — no driver or SDK required. Every tool that works with files just works with your database.

## Quick Start

```bash
# Build
go build -o bin/mongofuse ./cmd/mongofuse

# Mount a MongoDB database
mongofuse mount "mongodb://user:pass@localhost:27017" /mnt/db

# Explore
ls /mnt/db/                              # list databases
ls /mnt/db/mydb/                         # list collections
ls /mnt/db/mydb/users/                   # list document IDs
cat /mnt/db/mydb/users/507f1f77.json     # read a document

# Write
echo '{"name":"Bob","age":30}' > /mnt/db/mydb/users/new.json   # insert
echo '{"name":"Bob","age":31}' > /mnt/db/mydb/users/507f1f77.json  # replace

# Delete
rm /mnt/db/mydb/users/507f1f77.json      # delete document
rm -r /mnt/db/mydb/oldcoll/              # drop collection

# Create
mkdir /mnt/db/mydb/newcoll               # create collection

# Query with aggregation path segments
ls /mnt/db/mydb/orders/.match/status/shipped/.sort/created_at/.limit/10/
cat /mnt/db/mydb/orders/.match/status/shipped/.sort/created_at/.limit/10/.export/json
```

## Aggregation Pipeline Paths

Chain path segments to build MongoDB aggregation pipelines. Each segment maps to a native aggregation stage — no custom query language:

| Path Segment | Aggregation Stage | Example |
|---|---|---|
| `.match/field/value` | `{$match: {field: value}}` | `.match/status/active` |
| `.sort/field` | `{$sort: {field: 1}}` | `.sort/created_at` |
| `.sort/-field` | `{$sort: {field: -1}}` | `.sort/-created_at` |
| `.limit/N` | `{$limit: N}` | `.limit/10` |
| `.skip/N` | `{$skip: N}` | `.skip/20` |
| `.project/f1,f2` | `{$project: {f1:1, f2:1}}` | `.project/name,email` |

Segments can be chained in any order. The FUSE layer translates the full path into a single `aggregate()` call.

## Use Cases

### Developer debugging
Browse your database without a client. Inspect documents, grep across collections, quick data fixes with `echo`.

### AI agent workspace
Agents explore and manipulate database data using `ls`/`cat`/`grep` — no MongoDB driver needed. Works with Claude Code, Cursor, or any tool that reads files.

### Scripting
Pipe MongoDB data through Unix tools:
```bash
# Find all users named Alice
grep -l '"name":"Alice"' /mnt/db/mydb/users/*.json

# Export filtered data
cat /mnt/db/mydb/orders/.match/status/shipped/.limit/100/.export/json > shipped.json

# Count documents
ls /mnt/db/mydb/users/ | wc -l
```

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Unix Tools  │────▶│    FUSE /    │────▶│   MongoDB    │
│  ls, cat,    │     │    NFS       │     │   Server     │
│  grep, echo  │◀────│   Daemon     │◀────│  (any)       │
└──────────────┘     └──────────────┘     └──────────────┘
```

MongoFUSE connects as a standard MongoDB client. It works with:
- [DocumentDB](https://github.com/documentdb/documentdb) (including documentdb-local)
- MongoDB Community/Enterprise
- Any MongoDB wire protocol compatible server

## Filesystem Layout

```
/mnt/db/
├── admin/                          # databases
├── mydb/
│   ├── users/                      # collections
│   │   ├── 507f1f77bcf86cd7994.json  # documents (by _id)
│   │   ├── 507f1f77bcf86cd7995.json
│   │   └── .match/                 # aggregation path segments
│   │       └── status/
│   │           └── active/
│   │               ├── *.json      # filtered results
│   │               └── .sort/...   # chain more stages
│   └── orders/
└── ...
```

## Development

```bash
git clone https://github.com/xgerman/mongofuse.git
cd mongofuse
go build -o bin/mongofuse ./cmd/mongofuse
go test ./...
```

## Status

Early prototype. Core filesystem operations (ls, cat, echo, rm, mkdir) are the priority.

## License

MIT
