# DocumentDBFUSE

Mount any MongoDB-compatible database as a filesystem via FUSE. Linux only.

Browse collections with `ls`, read documents with `cat`, search with `grep`, write with `echo` — no driver or SDK required. Every tool that works with files just works with your database.

## Quick Start with Docker

```bash
# Clone and start (DocumentDB-local + DocumentDBFUSE)
git clone https://github.com/xgerman/documentdbfuse.git
cd documentdbfuse
docker compose up -d

# Wait ~25s for DocumentDB to initialize, then:
docker exec documentdbfuse-documentdbfuse-1 ls /mnt/db/
```

## Quick Start from Source

```bash
go build -o bin/documentdbfuse ./cmd/documentdbfuse
./bin/documentdbfuse mount "mongodb://user:pass@localhost:27017" /mnt/db
```

## CRUD Operations

```bash
# Browse
ls /mnt/db/                              # list databases
ls /mnt/db/mydb/                         # list collections
ls /mnt/db/mydb/users/                   # list documents
cat /mnt/db/mydb/users/user1.json        # read a document

# Write
mkdir /mnt/db/mydb/newcoll                                      # create collection
echo '{"name":"Bob","age":30}' > /mnt/db/mydb/newcoll/bob.json  # insert document
echo '{"name":"Bob","age":31}' > /mnt/db/mydb/newcoll/bob.json  # replace document

# Delete
rm /mnt/db/mydb/newcoll/bob.json         # delete document
rmdir /mnt/db/mydb/newcoll               # drop collection
```

## Aggregation Pipeline Queries

Chain path segments to build MongoDB aggregation pipelines. Each segment maps directly to a native aggregation stage — no custom query language.

| Path Segment | Aggregation Stage | Example |
|---|---|---|
| `.match/field/value` | `{$match: {field: value}}` | `.match/status/active` |
| `.sort/field` | `{$sort: {field: 1}}` | `.sort/created_at` |
| `.sort/-field` | `{$sort: {field: -1}}` | `.sort/-created_at` |
| `.limit/N` | `{$limit: N}` | `.limit/10` |
| `.skip/N` | `{$skip: N}` | `.skip/20` |
| `.project/f1,f2` | `{$project: {f1:1, f2:1}}` | `.project/name,email` |

### List matching documents

```bash
# ls returns document IDs that match the query
ls /mnt/db/sampledb/users/.match/city/Seattle/
# → results.json  user1.json

ls /mnt/db/sampledb/users/.sort/-age/.limit/2/
# → results.json  user2.json  user3.json
```

### Read all results as JSON

```bash
# results.json returns the full aggregation output as a JSON array
cat /mnt/db/sampledb/users/.match/city/Seattle/results.json
# [
#   {
#     "_id": "user1",
#     "city": "Seattle",
#     ...
#   }
# ]
```

### Read individual matched documents

```bash
# cat a specific matched document
cat /mnt/db/sampledb/users/.match/city/Seattle/user1.json

# or read all matched docs one by one
ls /mnt/db/sampledb/users/.match/city/Seattle/ \
  | grep -v results.json \
  | xargs -I{} cat /mnt/db/sampledb/users/.match/city/Seattle/{}
```

### Chain multiple stages

```bash
# Active users, sorted by age descending, top 3, only name and email
cat /mnt/db/sampledb/users/.match/isActive/true/.sort/-age/.limit/3/.project/firstName,email/results.json
```

### Legacy .export/json syntax

`.export/json` still works as an explicit terminal if preferred:

```bash
cat /mnt/db/sampledb/users/.match/isActive/true/.export/json
```

## Use Cases

### Developer debugging
Browse your database without a client. Inspect documents, grep across collections, quick data fixes with `echo`.

### AI agent workspace
Agents explore and manipulate database data using `ls`/`cat`/`grep` — no MongoDB driver needed. Works with Claude Code, Cursor, or any tool that reads files.

### Scripting
```bash
# Find users in a city
ls /mnt/db/mydb/users/.match/city/Seattle/ | grep -v results

# Export filtered data to a file
cat /mnt/db/mydb/orders/.match/status/shipped/.limit/100/results.json > shipped.json

# Count matching documents
ls /mnt/db/mydb/users/.match/isActive/true/ | grep -v results | wc -l

# Read every matching document
ls /mnt/db/mydb/users/.match/city/Seattle/ \
  | grep -v results.json \
  | xargs -I{} cat /mnt/db/mydb/users/.match/city/Seattle/{}
```

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Unix Tools  │────▶│    FUSE      │────▶│   MongoDB    │
│  ls, cat,    │     │    Daemon    │     │   Server     │
│  grep, echo  │◀────│   (Go)      │◀────│  (any)       │
└──────────────┘     └──────────────┘     └──────────────┘
```

DocumentDBFUSE connects as a standard MongoDB client. It works with:
- [DocumentDB](https://github.com/documentdb/documentdb) (including documentdb-local)
- MongoDB Community/Enterprise
- Any MongoDB wire protocol compatible server

## Filesystem Layout

```
/mnt/db/
├── sampledb/                              # database
│   ├── users/                             # collection
│   │   ├── user1.json                     # document (by _id)
│   │   ├── user2.json
│   │   └── .match/                        # aggregation pipeline
│   │       └── city/
│   │           └── Seattle/
│   │               ├── results.json       # full query results
│   │               ├── user1.json         # matched document
│   │               └── .sort/             # chain more stages
│   │                   └── -age/
│   │                       └── .limit/
│   │                           └── 1/
│   │                               ├── results.json
│   │                               └── user1.json
│   └── orders/
└── admin/
```

## Docker

The `docker-compose.yml` starts DocumentDB-local and DocumentDBFUSE together:

```bash
docker compose up -d          # start both services
docker compose down           # stop and cleanup
```

The DocumentDBFUSE container needs FUSE access:
- `cap_add: [SYS_ADMIN]`
- `devices: ["/dev/fuse"]`
- `security_opt: [apparmor:unconfined]`

Connection string uses TLS (DocumentDB-local default):
```
mongodb://testuser:testpass123@documentdb:10260/?directConnection=true&tls=true&tlsInsecure=true
```

## Development

```bash
git clone https://github.com/xgerman/documentdbfuse.git
cd documentdbfuse
go build -o bin/documentdbfuse ./cmd/documentdbfuse
go test ./...
```

### Run integration tests

```bash
./scripts/test.sh
```

## Status

Early prototype. Working:
- ✅ `ls` — list databases, collections, documents
- ✅ `cat` — read documents as JSON
- ✅ `echo >` — insert/replace documents
- ✅ `rm` — delete documents
- ✅ `mkdir` / `rmdir` — create/drop collections
- ✅ Aggregation pipeline paths (`.match`, `.sort`, `.limit`, `.skip`, `.project`)
- ✅ `ls | xargs cat` on pipeline results
- ✅ Docker Compose with DocumentDB-local

## License

MIT
