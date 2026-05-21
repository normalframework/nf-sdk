// Copyright (c) 2023, Normal Software Inc.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.
// 3. Neither the name of Normal Software Inc. nor the names of its
//    contributors may be used to endorse or promote products derived from
//    this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	sparkplug "github.com/normalframework/sparkplug-historian/sparkplug_b"
)

// TestMain wires up a real Postgres connection (defaults match
// test/docker-compose.yml) before running any tests, and drops all
// historian tables afterwards so the run is idempotent.
func TestMain(m *testing.M) {
	PGHOST = getEnv("TEST_PGHOST", "localhost")
	PGPORT = getEnvInt("TEST_PGPORT", 15432)
	PGUSER = getEnv("TEST_PGUSER", "postgres")
	PGPASSWORD = getEnv("TEST_PGPASSWORD", "postgres")
	PGDATABASE = getEnv("TEST_PGDATABASE", "sparkplug_test")
	PGSCHEMA = "public"
	SPARKPLUG_GROUP_ID = "test-group"

	tmp, err := os.MkdirTemp("", "sparkplug-sql-test-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot create temp dir:", err)
		os.Exit(1)
	}
	DATA_DIR = tmp
	defer os.RemoveAll(tmp)

	ctx := context.Background()
	pool, err = pgxpool.ConnectConfig(ctx, makeDbConfig())
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot connect to test DB (run: docker compose -f test/docker-compose.yml up -d):", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := applyDDL(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "DDL failed:", err)
		os.Exit(1)
	}

	nodes = make(map[string]*node_state)
	aliases = make(map[string]string)

	code := m.Run()

	pool.Exec(ctx, `DROP TABLE IF EXISTS metrics, metadata, nodes,
		node_connections, node_metadata, ontology CASCADE`)
	os.Exit(code)
}

func applyDDL(ctx context.Context) error {
	ddl, err := os.ReadFile("tables.sql")
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, string(ddl))
	return err
}

// ---- proto pointer helpers ----

func sp(s string) *string   { return &s }
func u64(n uint64) *uint64  { return &n }
func boolp(b bool) *bool    { return &b }

// ---- tests ----

func TestHistorian(t *testing.T) {
	const (
		nodeID   = "test-node"
		deviceID = "test-device"
	)
	topic := func(msgType string) []string {
		return []string{"spBv1.0", SPARKPLUG_GROUP_ID, msgType, nodeID, deviceID}
	}
	nbTopic := func(msgType string) []string {
		return []string{"spBv1.0", SPARKPLUG_GROUP_ID, msgType, nodeID}
	}

	// Initialise node state — required before any per-node function call.
	nodes_mu.Lock()
	nodes[nodeID] = &node_state{state: STATE_GOOD}
	nodes_mu.Unlock()

	now := uint64(time.Now().UnixMilli())

	// ------------------------------------------------------------------
	// NBIRTH — node connects, bdSeq recorded in node_connections
	// ------------------------------------------------------------------
	t.Run("NBIRTH creates node connection", func(t *testing.T) {
		payload := &sparkplug.Payload{
			Timestamp: u64(now),
			Seq:       u64(0),
			Metrics: []*sparkplug.Payload_Metric{
				{
					Name:  sp(BD_SEQUENCE),
					Value: &sparkplug.Payload_Metric_LongValue{LongValue: 42},
				},
			},
		}
		createNodeConnection(nbTopic("NBIRTH"), payload)

		ctx := context.Background()
		var bdSeq int
		err := pool.QueryRow(ctx,
			`SELECT bdSeq FROM node_connections WHERE node_name = $1`, nodeID,
		).Scan(&bdSeq)
		if err != nil {
			t.Fatalf("query node_connections: %v", err)
		}
		if bdSeq != 42 {
			t.Errorf("expected bdSeq=42, got %d", bdSeq)
		}
	})

	// ------------------------------------------------------------------
	// DBIRTH — device birth populates alias + metadata tables
	// ------------------------------------------------------------------
	t.Run("DBIRTH populates metadata", func(t *testing.T) {
		payload := &sparkplug.Payload{
			Timestamp: u64(now),
			Seq:       u64(1),
			Metrics: []*sparkplug.Payload_Metric{
				{
					Name:      sp("temperature"),
					Alias:     u64(1),
					Timestamp: u64(now),
					Value:     &sparkplug.Payload_Metric_DoubleValue{DoubleValue: 22.5},
				},
				{
					Name:      sp("humidity"),
					Alias:     u64(2),
					Timestamp: u64(now),
					Value:     &sparkplug.Payload_Metric_DoubleValue{DoubleValue: 55.0},
				},
			},
		}

		updateAliases(topic("DBIRTH"), payload)

		ctx := context.Background()
		var count int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM metadata WHERE node_name=$1 AND device_name=$2`,
			nodeID, deviceID,
		).Scan(&count); err != nil {
			t.Fatalf("query metadata: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 metadata rows, got %d", count)
		}

		// In-memory alias map must also be populated.
		alias_mu.Lock()
		defer alias_mu.Unlock()
		key := fmt.Sprintf("%s\x00%s\x00%d", nodeID, deviceID, 1)
		if aliases[key] != "temperature" {
			t.Errorf("alias 1 not mapped to 'temperature', got %q", aliases[key])
		}
	})

	// ------------------------------------------------------------------
	// DDATA — metric values are inserted into the metrics table
	// ------------------------------------------------------------------
	t.Run("DDATA inserts metrics", func(t *testing.T) {
		ts := uint64(time.Now().UnixMilli())
		seq := uint64(2)
		payload := &sparkplug.Payload{
			Timestamp: u64(ts),
			Seq:       &seq,
			Metrics: []*sparkplug.Payload_Metric{
				{
					Alias:     u64(1),
					Timestamp: u64(ts),
					Value:     &sparkplug.Payload_Metric_DoubleValue{DoubleValue: 23.1},
				},
				{
					Alias:     u64(2),
					Timestamp: u64(ts),
					Value:     &sparkplug.Payload_Metric_DoubleValue{DoubleValue: 60.0},
				},
			},
		}

		insertData(topic("DDATA"), payload)

		ctx := context.Background()
		var count int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM metrics m
			 JOIN metadata md ON m.metric_id = md.id
			 WHERE md.node_name = $1 AND md.device_name = $2`,
			nodeID, deviceID,
		).Scan(&count); err != nil {
			t.Fatalf("query metrics: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 metric rows, got %d", count)
		}

		// Verify the stored scalar values.
		rows, err := pool.Query(ctx,
			`SELECT md.metric_name, m.value
			 FROM metrics m
			 JOIN metadata md ON m.metric_id = md.id
			 WHERE md.node_name = $1 AND md.device_name = $2
			 ORDER BY md.metric_name`,
			nodeID, deviceID,
		)
		if err != nil {
			t.Fatalf("query metric values: %v", err)
		}
		defer rows.Close()

		want := map[string]float64{"humidity": 60.0, "temperature": 23.1}
		for rows.Next() {
			var name string
			var val float64
			if err := rows.Scan(&name, &val); err != nil {
				t.Fatal(err)
			}
			if want[name] != val {
				t.Errorf("metric %q: expected %v, got %v", name, want[name], val)
			}
		}
	})

	// ------------------------------------------------------------------
	// NDEATH — aliases are invalidated in the database
	// ------------------------------------------------------------------
	t.Run("NDEATH invalidates aliases", func(t *testing.T) {
		invalidateAliases(nodeID)

		ctx := context.Background()
		var nullCount int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM metadata
			 WHERE node_name = $1 AND metric_alias IS NULL`,
			nodeID,
		).Scan(&nullCount); err != nil {
			t.Fatalf("query metadata: %v", err)
		}
		if nullCount != 2 {
			t.Errorf("expected 2 rows with null alias after NDEATH, got %d", nullCount)
		}
	})
}
