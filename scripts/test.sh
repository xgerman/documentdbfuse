#!/usr/bin/env bash
set -euo pipefail

COMPOSE="docker compose"
MONGOFUSE_CONTAINER="mongofuse-mongofuse-1"
MOUNT_POINT="/mnt/db"
PASS=0
FAIL=0

cleanup() {
    echo "==> Cleaning up..."
    $COMPOSE down -v --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

run_test() {
    local desc="$1"
    shift
    if "$@"; then
        echo "  ✅ PASS: $desc"
        ((PASS++))
    else
        echo "  ❌ FAIL: $desc"
        ((FAIL++))
    fi
}

exec_in() {
    docker exec "$MONGOFUSE_CONTAINER" "$@"
}

echo "==> Starting services..."
$COMPOSE up -d --build

echo "==> Waiting for DocumentDB to be healthy..."
for i in $(seq 1 30); do
    if docker inspect --format='{{.State.Health.Status}}' mongofuse-documentdb-1 2>/dev/null | grep -q healthy; then
        echo "    DocumentDB ready after ~${i}s"
        break
    fi
    sleep 2
done

echo "==> Waiting for documentdbfuse to mount (up to 30s)..."
for i in $(seq 1 15); do
    if exec_in mountpoint -q "$MOUNT_POINT" 2>/dev/null; then
        echo "    FUSE mounted after ~$((i * 2))s"
        break
    fi
    sleep 2
done

if ! exec_in mountpoint -q "$MOUNT_POINT" 2>/dev/null; then
    echo "❌ FUSE mount never appeared. Container logs:"
    docker logs "$MONGOFUSE_CONTAINER" 2>&1 | tail -20
    exit 1
fi

echo ""
echo "==> Running filesystem tests..."

# Test: list root
run_test "ls mount point" exec_in ls "$MOUNT_POINT"

# Test: create a directory (database)
run_test "mkdir (create database)" exec_in mkdir -p "$MOUNT_POINT/testdb"

# Test: list shows new dir
run_test "ls shows new database" exec_in test -d "$MOUNT_POINT/testdb"

# Test: create a sub-directory (collection)
run_test "mkdir (create collection)" exec_in mkdir -p "$MOUNT_POINT/testdb/testcol"

# Test: write a document
run_test "echo (insert document)" \
    exec_in sh -c "echo '{\"name\":\"hello\",\"value\":42}' > $MOUNT_POINT/testdb/testcol/doc1.json"

# Test: read the document back
run_test "cat (read document)" exec_in cat "$MOUNT_POINT/testdb/testcol/doc1.json"

# Test: list collection shows document
run_test "ls collection shows document" exec_in ls "$MOUNT_POINT/testdb/testcol/"

# Test: remove document
run_test "rm (delete document)" exec_in rm "$MOUNT_POINT/testdb/testcol/doc1.json"

# Test: remove collection
run_test "rmdir (drop collection)" exec_in rmdir "$MOUNT_POINT/testdb/testcol"

# Test: remove database
run_test "rmdir (drop database)" exec_in rmdir "$MOUNT_POINT/testdb"

# --- Aggregation pipeline tests ---

# Test: ls with pipeline filter returns matching docs
run_test "ls pipeline (.match)" \
    exec_in sh -c "ls $MOUNT_POINT/sampledb/users/.match/city/Seattle/ | grep -q '.json'"

# Test: cat results.json returns JSON array
run_test "cat pipeline results.json" \
    exec_in sh -c "cat $MOUNT_POINT/sampledb/users/.match/city/Seattle/results.json | grep -q '_id'"

# Test: ls pipeline + sort + limit
run_test "ls pipeline (.sort + .limit)" \
    exec_in sh -c "ls $MOUNT_POINT/sampledb/users/.sort/-age/.limit/2/ | grep -q '.json'"

# Test: ls pipeline | xargs cat — read all matching documents (no filtering needed)
run_test "ls pipeline | xargs cat (read each matched doc)" \
    exec_in sh -c "cd $MOUNT_POINT/sampledb/users/.match/city/Seattle && ls | xargs -I{} cat {} | grep -q 'Seattle'"

# Test: cat .csv/results
run_test "cat .csv/results (CSV export)" \
    exec_in sh -c "cat $MOUNT_POINT/sampledb/users/.match/city/Seattle/.csv/results | grep -q '_id'"

# Test: cat .tsv/results
run_test "cat .tsv/results (TSV export)" \
    exec_in sh -c "cat $MOUNT_POINT/sampledb/users/.match/city/Seattle/.tsv/results | grep -q '_id'"

echo ""
echo "================================"
echo "Results: $PASS passed, $FAIL failed"
echo "================================"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
echo "All tests passed!"
