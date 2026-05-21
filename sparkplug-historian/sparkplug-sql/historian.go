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
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	sparkplug "github.com/normalframework/sparkplug-historian/sparkplug_b"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

// ---- env helpers ----

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "true", "1":
			return true
		case "false", "0":
			return false
		}
	}
	return fallback
}

// ---- MQTT TLS helpers ----

var (
	mqttCertFile = getEnv("MQTT_CERTFILE", "")
	mqttKeyFile  = getEnv("MQTT_KEYFILE", "")
	mqttCAFile   = getEnv("MQTT_CAFILE", "")
	mqttSkipVerify = getEnvBool("MQTT_SKIP_VERIFY", true)
)

func makeMQTTTLSConfig() *tls.Config {
	cer := tls.Certificate{}
	var err error
	if mqttCertFile != "" && mqttKeyFile != "" {
		cer, err = tls.LoadX509KeyPair(mqttCertFile, mqttKeyFile)
		if err != nil {
			log.Fatal().Err(err).Msg("Error loading client certificate")
		}
	}
	certpool := x509.NewCertPool()
	if mqttCAFile != "" {
		ca, err := ioutil.ReadFile(filepath.Clean(mqttCAFile))
		if err != nil {
			log.Fatal().Err(err).Msg("Error loading CA")
		}
		certpool.AppendCertsFromPEM(ca)
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cer},
		RootCAs:            certpool,
		ClientAuth:         tls.NoClientCert,
		InsecureSkipVerify: mqttSkipVerify,
	}
}

// ---- Sparkplug payload helpers ----

func compress(in []byte) string {
	buf := bytes.Buffer{}
	gw := gzip.NewWriter(&buf)
	gw.Write(in)
	gw.Close()
	return buf.String()
}

func unmarshalSparkplug(enc []byte, payloadFormat string) (*sparkplug.Payload, error) {
	payload := sparkplug.Payload{}
	if payloadFormat == "json+gzip" || payloadFormat == "proto+gzip" {
		buf := bytes.NewBuffer(enc)
		gr, err := gzip.NewReader(buf)
		if err != nil {
			return &payload, err
		}
		defer gr.Close()
		enc, err = ioutil.ReadAll(gr)
		if err != nil {
			return &payload, err
		}
	}
	if err := proto.Unmarshal(enc, &payload); err != nil {
		return &payload, err
	}
	return &payload, nil
}

// ---- config ----

var (
	NAMESPACE          = "spBv1.0"
	SPARKPLUG_GROUP_ID = getEnv("SPARKPLUG_GROUP_ID", "normalgw")
	MQTT_BROKER        = getEnv("MQTT_BROKER", "LOCALHOST")
	MQTT_PORT          = getEnvInt("MQTT_PORT", 1883)
	MQTT_CLIENT_ID     = getEnv("MQTT_CLIENT_ID", "sparkplug_historian")
	MQTT_USERNAME      = getEnv("MQTT_USERNAME", "")
	MQTT_PASSWORD      = getEnv("MQTT_PASSWORD", "")

	PGHOST     = getEnv("PGHOST", "localhost")
	PGUSER     = getEnv("PGUSER", "postgres")
	PGPASSWORD = getEnv("PGPASSWORD", "password")
	PGPORT     = getEnvInt("PGPORT", 5432)
	PGDATABASE = getEnv("PGDATABASE", "postgres")
	PGSCHEMA   = getEnv("PGSCHEMA", "public")

	PAYLOAD_FORMAT = getEnv("PAYLOAD_FORMAT", "proto+gzip")
	PGMAXCONNS     = getEnvInt("PGMAXCONNS", 30)

	debug = getEnvBool("DEBUG", false)

	db_timeout = 60
)

func dbCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(db_timeout)*time.Second)
}

var pool *pgxpool.Pool
var client mqtt.Client

const (
	MQTT_DISCONNECTED = iota
	MQTT_CONNECTED    = iota
)

const (
	STATE_GOOD       = iota
	STATE_RECOVERING = iota
	STATE_DESYNCED   = iota
)
const (
	INITIAL_NODE_STATE = STATE_DESYNCED
)

const (
	BD_SEQUENCE   = "bdSeq"
	SQL_BATCH_MAX = 50
)

var NoResults = errors.New("no result")

type node_state struct {
	mu                   sync.RWMutex
	state                int
	lgsn                 uint64
	recovery_id          string
	epoch_hdata_received bool
}

var nodes map[string]*node_state
var nodes_mu sync.Mutex
var alias_mu sync.Mutex
var aliases map[string]string

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func makeDbConfig() *pgxpool.Config {
	cfg, err := pgxpool.ParseConfig(fmt.Sprintf("postgres://%s:%s@%s:%d/%s?pool_max_conns=%d",
		PGUSER, PGPASSWORD, PGHOST, PGPORT, PGDATABASE, PGMAXCONNS))
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing database config")
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = PGSCHEMA
	return cfg
}

func updateAliases(topic []string, msg *sparkplug.Payload) {
	node_id := topic[3]
	device_id := topic[4]

	nodes_mu.Lock()
	state := nodes[node_id]
	nodes_mu.Unlock()

	state.mu.Lock()
	defer state.mu.Unlock()
	ctx, cancel := dbCtx()
	defer cancel()

	if msg.Metrics == nil {
		return
	}

	log.Info().
		Str("node_id", node_id).
		Str("device_id", device_id).
		Int("metrics", len(msg.Metrics)).
		Msg("DBIRTH: Updating aliases")

	for i := 0; i < len(msg.Metrics); i += SQL_BATCH_MAX {
		b := &pgx.Batch{}
		sqlStatement := `
INSERT INTO metadata (group_name, node_name, device_name, metric_name, metric_alias)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (group_name, node_name, device_name, metric_name)
DO
 UPDATE SET metric_alias = EXCLUDED.metric_alias;`
		alias_mu.Lock()
		for _, metric := range msg.Metrics[i:min(i+SQL_BATCH_MAX, len(msg.Metrics))] {
			if metric == nil || metric.Alias == nil || metric.Name == nil {
				continue
			}
			b.Queue(sqlStatement, SPARKPLUG_GROUP_ID, node_id, device_id, metric.Name, metric.Alias)
			aliases[fmt.Sprintf("%s\x00%s\x00%d", node_id, device_id, *metric.Alias)] = *metric.Name
		}
		alias_mu.Unlock()

		tic := time.Now()
		batchResults := pool.SendBatch(ctx, b)

		rows, qerr := batchResults.Query()
		if qerr != nil && !errors.Is(qerr, pgx.ErrNoRows) && !errors.Is(qerr, NoResults) {
			log.Error().Err(qerr).
				Str("node_id", node_id).
				Str("device_id", device_id).
				Msg("UpdateAliases")
		} else {
			log.Debug().
				Dur("dt", time.Since(tic)).
				Str("node_id", node_id).
				Str("device_id", device_id).
				Msg("UpdateAliases batch")
		}
		rows.Close()
		batchResults.Close()
	}
}

func invalidateAliases(node_id string) {
	ctx, cancel := dbCtx()
	defer cancel()
	log.Info().
		Str("node_id", node_id).
		Msg("NDEATH: invaldiating aliases")
	sqlStatement := `
UPDATE metadata
SET metric_alias = NULL
WHERE group_name = $1 AND node_name = $2
`
	for {
		_, err := pool.Exec(ctx, sqlStatement,
			SPARKPLUG_GROUP_ID, node_id)
		if err != nil {
			log.Error().Err(err).Msg("Error invalidating aliases; cannot continue since future inserts may be incorrect")
		} else {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func marshalProperties(props *sparkplug.Payload_PropertySet) ([]byte, error) {
	value_map := make(map[string]*sparkplug.Payload_PropertyValue)
	for i := 0; i < len(props.Keys); i++ {
		value_map[props.Keys[i]] = props.Values[i]
	}
	return json.Marshal(value_map)
}

func updateMetadata(topic []string, msg *sparkplug.Payload) {
	node_id := topic[3]
	group_id := topic[1]
	device_id := topic[4]

	nodes_mu.Lock()
	state := nodes[node_id]
	nodes_mu.Unlock()

	state.mu.Lock()
	defer state.mu.Unlock()
	ctx, cancel := dbCtx()
	defer cancel()
	log.Info().
		Str("node_id", node_id).
		Str("device_id", device_id).
		Msg("DBIRTH: Updating metadata")

	for i := 0; i < len(msg.Metrics); i += SQL_BATCH_MAX {
		b := &pgx.Batch{}
		updates := 0
		sqlStatement := `
UPDATE metadata
SET properties = $5
WHERE group_name = $1 AND node_name = $2 AND device_name = $3 AND metric_alias = $4`
		for _, metric := range msg.Metrics[i:min(i+SQL_BATCH_MAX, len(msg.Metrics))] {
			if metric == nil || metric.Properties == nil {
				continue
			}
			props_json, err := marshalProperties(metric.Properties)
			if err != nil {
				log.Warn().Err(err).Msg("Error encoding properties json")
				continue
			}
			updates++
			b.Queue(sqlStatement, group_id, node_id, device_id, metric.Alias, props_json)
		}
		batchResults := pool.SendBatch(ctx, b)

		rows, qerr := batchResults.Query()
		if qerr != nil {
			log.Error().Err(qerr).
				Str("node_id", node_id).
				Str("device_id", device_id).
				Msg("UpdateMetadata")
		}
		rows.Close()
		batchResults.Close()
	}
}

func upsertNodeMetadata(topic []string, msg *sparkplug.Payload) {
	node_id := topic[3]
	group_id := topic[1]

	ctx, cancel := dbCtx()
	defer cancel()
	log.Info().
		Str("node_id", node_id).
		Int("metrics", len(msg.Metrics)).
		Msg("NDATA: Upsert Node Data")
	tx, err := pool.Begin(ctx)
	if err != nil {
		panic(err)
	}

	b := &pgx.Batch{}
	sqlStatement := `
INSERT INTO node_metadata (name, group_name, node_name, metadata)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name, group_name, node_name)
DO
 UPDATE SET metadata = EXCLUDED.metadata;`
	for _, metric := range msg.Metrics {
		if metric.Name == nil || *metric.Name == "" {
			log.Warn().Msg("Skipping node metric with empty name")
			continue
		}
		props_json, err := marshalProperties(metric.Properties)
		if err != nil {
			log.Warn().Err(err).Msg("Error encoding node data properties json")
			continue
		}
		b.Queue(sqlStatement, *metric.Name, group_id, node_id, props_json)
	}
	batchResults := tx.SendBatch(ctx, b)

	var qerr error
	var rows pgx.Rows
	for qerr == nil {
		rows, qerr = batchResults.Query()
		rows.Close()
	}
	batchResults.Close()
	err = tx.Commit(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Error committing node metadata")
		tx.Rollback(context.Background())
	}
}

func makeScalar(msg *sparkplug.Payload_Metric) float64 {
	var rv float64 = math.NaN()
	if msg.IsNull != nil && *msg.IsNull {
		return rv
	}
	switch msg.Value.(type) {
	case *sparkplug.Payload_Metric_IntValue:
		rv = float64(msg.GetIntValue())
	case *sparkplug.Payload_Metric_LongValue:
		rv = float64(msg.GetLongValue())
	case *sparkplug.Payload_Metric_DoubleValue:
		rv = msg.GetDoubleValue()
	case *sparkplug.Payload_Metric_FloatValue:
		rv = float64(msg.GetFloatValue())
	case *sparkplug.Payload_Metric_BooleanValue:
		if msg.GetBooleanValue() {
			rv = 1
		} else {
			rv = 0
		}
	default:
		log.Debug().Str("val", fmt.Sprintf("%T", msg.Value)).Msg("UnsupportedDatatype")
		return rv
	}
	return rv
}

func insertData(topic []string, msg *sparkplug.Payload) {
	group_id := topic[1]
	node_id := topic[3]
	device_id := topic[4]

	nodes_mu.Lock()
	state := nodes[node_id]
	nodes_mu.Unlock()

	state.mu.RLock()
	defer state.mu.RUnlock()
	ctx, cancel := dbCtx()
	defer cancel()
	log.Debug().
		Str("node_id", node_id).
		Str("device_id", device_id).
		Int("count", len(msg.Metrics)).
		Uint64("seq", *msg.Seq).
		Msg("DDATA: Inserting Data")

	const insertStatement = `
INSERT INTO metrics (metric_id, time, value) VALUES (
 (SELECT id FROM metadata WHERE group_name = $1 AND
   node_name = $2 AND device_name = $3 AND metric_alias = $4), to_timestamp($5), $6)
ON CONFLICT (metric_id, time) DO NOTHING`
	err := pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		inserts := 0

		if state.state == STATE_GOOD {
			b.Queue(`
INSERT INTO nodes (group_name, node_name, lgsn) VALUES ($1, $2, $3)
ON CONFLICT (group_name, node_name)
DO UPDATE SET lgsn = $3`, group_id, node_id, *msg.Seq)
			inserts++
		}

		for _, metric := range msg.Metrics {
			if metric.Timestamp == nil || metric.Alias == nil {
				continue
			}
			log.Debug().
				Str("group", SPARKPLUG_GROUP_ID).
				Str("node_id", node_id).
				Str("device_id", device_id).
				Uint64("alias", *metric.Alias).
				Uint64("timestamp", *metric.Timestamp).
				Float64("scalar", makeScalar(metric)).
				Bool("is_historical", metric.IsHistorical != nil && *metric.IsHistorical).
				Msg("Insert")

			if metric.IsHistorical != nil && *metric.IsHistorical {
				state.epoch_hdata_received = true
			} else {
				b.Queue(insertStatement, SPARKPLUG_GROUP_ID,
					node_id, device_id,
					*metric.Alias,
					*metric.Timestamp/1000,
					makeScalar(metric),
				)
				inserts++
			}
		}

		batchResults := tx.SendBatch(ctx, b)
		for i := 0; i < inserts; i++ {
			_, err := batchResults.Exec()
			if err != nil {
				batchResults.Close()
				return err
			}
		}

		err := batchResults.Close()
		if err != nil {
			log.Warn().Err(err).Msg("Commit error")
			return err
		}
		return nil
	})
	if err != nil {
		log.Warn().Err(err).Msg("Skipping insert due to transaction error: adding to recovery log")
		state.state = STATE_RECOVERING
		logAllMetrics(node_id, device_id, msg.Metrics, false)
	} else {
		if state.state == STATE_GOOD && msg.Seq != nil {
			state.lgsn = *msg.Seq
		}
		logAllMetrics(node_id, device_id, msg.Metrics, true)
	}
}

func getBDSeq(payload *sparkplug.Payload) int {
	bdSeq := 0
	for _, metric := range payload.Metrics {
		if *metric.Name == BD_SEQUENCE {
			bdSeq = int(makeScalar(metric))
		}
	}
	return bdSeq
}

func createNodeConnection(topic []string, payload *sparkplug.Payload) {
	node_id := topic[3]
	group_id := topic[1]

	bdSeq := getBDSeq(payload)

	if bdSeq == 0 {
		log.Warn().Msg("createNodeConnection: BD_SEQUENCE not found in NBIRTH message")
		return
	}

	log.Info().
		Str("node_id", node_id).
		Str("group_id", group_id).
		Int("bdSeq", bdSeq).
		Msg("node connected")

	ctx, cancel := dbCtx()
	defer cancel()
	tx, err := pool.Begin(ctx)

	if err != nil {
		panic(err)
	}

	_, err = tx.Exec(ctx, `
INSERT INTO nodes (group_name, node_name, last_nbirth)
VALUES ($1, $2, to_timestamp($3))
ON CONFLICT (group_name, node_name)
DO UPDATE SET last_nbirth = to_timestamp($3)
`, group_id, node_id, *payload.Timestamp/1000)

	if err != nil {
		log.Warn().Err(err).Msg("Error inserting node")
		return
	}
	_, err = tx.Exec(ctx, `
INSERT INTO node_connections (group_name, node_name, nbirth, bdSeq)
VALUES ($1, $2, to_timestamp($3), $4)
`, group_id, node_id, *payload.Timestamp/1000, bdSeq)

	if err != nil {
		log.Warn().Err(err).Msg("Error inserting node connection")
		return
	}
	err = tx.Commit(ctx)

	if err != nil {
		log.Warn().Err(err).Msg("createNodeConnection: Error committing db transaction")
		tx.Rollback(context.Background())
		return
	}
}

func closeNodeConnection(topic []string, payload *sparkplug.Payload) {
	node_id := topic[3]
	group_id := topic[1]

	bdSeq := getBDSeq(payload)

	if bdSeq == 0 {
		log.Warn().Msg("closeNodeConnection: BD_SEQUENCE not found in NBIRTH message")
		return
	}

	log.Info().
		Str("node_id", node_id).
		Str("group_id", group_id).
		Int("bdSeq", bdSeq).
		Msg("node disconnected")

	ctx, cancel := dbCtx()
	defer cancel()
	tx, err := pool.Begin(ctx)

	if err != nil {
		panic(err)
	}

	_, err = tx.Exec(ctx, `
INSERT INTO nodes (group_name, node_name, last_ndeath)
VALUES ($1, $2, to_timestamp($3))
ON CONFLICT (group_name, node_name)
DO UPDATE SET last_ndeath = to_timestamp($3)
`, group_id, node_id, *payload.Timestamp/1000)

	if err != nil {
		log.Warn().Err(err).Msg("closeNodeConnection: Error inserting node")
		return
	}
	_, err = tx.Exec(ctx, `
UPDATE node_connections
SET ndeath = to_timestamp($4)
WHERE group_name = $1 AND node_name = $2 AND bdSeq = $3
`, group_id, node_id, bdSeq, *payload.Timestamp/1000)

	if err != nil {
		log.Warn().Err(err).Msg("Error updating node connection")
		return
	}
	err = tx.Commit(ctx)

	if err != nil {
		log.Warn().Err(err).Msg("closeNodeConnection: Error committing db transaction")
		tx.Rollback(context.Background())
		return
	}
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var topic = strings.Split(msg.Topic(), "/")
	log.Debug().Strs("topic", topic).Msg("Received message")
	if topic[0] != NAMESPACE || topic[1] != SPARKPLUG_GROUP_ID || len(topic) < 4 {
		log.Warn().
			Str("topic", msg.Topic()).
			Msg("Unexpected message received from broker")
		return
	}

	message_type := topic[2]
	node_id := topic[3]
	payload, err := unmarshalSparkplug(msg.Payload(), PAYLOAD_FORMAT)
	if err != nil {
		log.Warn().
			Str("topic", msg.Topic()).
			Err(err).
			Msg("Could not parse message")
		return
	}

	nodes_mu.Lock()

	if nodes[node_id] == nil {
		log.Info().Str("node_id", node_id).Msg("Starting recovery watcher")
		nodes[node_id] = &node_state{}
		go trackVersions(topic[1], node_id)
	}
	state := nodes[node_id]
	nodes_mu.Unlock()

	switch strings.ToUpper(message_type) {
	case "DBIRTH":
		if len(topic) != 5 {
			log.Warn().
				Str("topic", msg.Topic()).
				Msg("DBIRTH received with invalid topic name")
			return
		}
		go func() {
			updateAliases(topic, payload)
			updateMetadata(topic, payload)
		}()
	case "DDATA":
		go insertData(topic, payload)
	case "NDATA":
		if state != nil &&
			payload.Uuid != nil &&
			state.recovery_id == *payload.Uuid {
			log.Info().
				Str("node_id", node_id).
				Msg("Marking recovery complete")
			state.state = STATE_GOOD
			state.recovery_id = ""
		}

		if len(payload.Metrics) > 0 {
			upsertNodeMetadata(topic, payload)
		}

	case "NBIRTH":
		createNodeConnection(topic, payload)
	case "NDEATH":
		go invalidateAliases(node_id)
		state.state = STATE_DESYNCED
		closeNodeConnection(topic, payload)
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Info().Msg("MQTT Connected")
	subTopic := NAMESPACE + "/" + SPARKPLUG_GROUP_ID + "/#"
	if token := client.Subscribe(subTopic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatal().Err(token.Error()).Msg("Failed to subscribe")
	}
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Warn().Err(err).Msg("MQTT Connection lost: marking all nodes as recovering")
	for _, v := range nodes {
		v.state = STATE_DESYNCED
		v.epoch_hdata_received = false
	}
}

func testDatabase() bool {
	ctx, cancel := dbCtx()
	defer cancel()
	err := pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "select 1")
		return err
	})
	return err == nil
}

func loadVersion(group_id string, node_id string) uint64 {
	ctx, cancel := dbCtx()
	defer cancel()
	var lgsn uint64
	err := pool.QueryRow(ctx,
		"SELECT lgsn FROM nodes WHERE group_name = $1 AND node_name = $2",
		group_id, node_id).Scan(&lgsn)
	if err != nil {
		log.Warn().Err(err).Str("node_id", node_id).Msg("Error loading LSGN from database")
		return 0
	} else {
		return lgsn
	}
}

func makeRetryCmd(node_id string) string {
	nodes_mu.Lock()
	state := nodes[node_id]
	nodes_mu.Unlock()

	if state.recovery_id == "" {
		state.recovery_id = uuid.New().String()
	}
	lpos := "Log Position"
	msg := sparkplug.Payload{
		Uuid: &state.recovery_id,
		Metrics: []*sparkplug.Payload_Metric{
			{
				Name: &lpos,
				Value: &sparkplug.Payload_Metric_LongValue{
					LongValue: state.lgsn,
				},
			},
		},
	}
	bin, _ := proto.Marshal(&msg)
	return compress(bin)
}

// trackVersions manages per-node recovery state. A node is either GOOD (both
// MQTT and Postgres sessions working, LGSN advancing normally), RECOVERING
// (waiting for historical backfill after a gap), or DESYNCED (re-requesting
// data from the node).
func trackVersions(group_id string, node_id string) {
	var TRACK_VERSION_EPOCH = 300 * time.Second

	nodes_mu.Lock()
	state := nodes[node_id]
	nodes_mu.Unlock()

	state.lgsn = loadVersion(group_id, node_id)
	state.state = INITIAL_NODE_STATE
	log.Info().
		Str("node_id", node_id).
		Uint64("lgsn", state.lgsn).
		Msg("Loaded LGSN from database")

	for {
		log.Info().
			Str("node_id", node_id).
			Int("state", state.state).
			Bool("hdata", state.epoch_hdata_received).
			Msg("Checking recovery state")

		for !testDatabase() {
			log.Debug().Str("node_id", node_id).Msg("Waiting for database")
			time.Sleep(2 * time.Second)
		}

		switch state.state {
		case STATE_DESYNCED:
			log.Info().
				Uint64("seq", state.lgsn).
				Str("node_id", node_id).
				Msg("Sending retry request")
			token := client.Publish(NAMESPACE+"/"+SPARKPLUG_GROUP_ID+"/NCMD"+"/"+node_id,
				1, false, makeRetryCmd(node_id))
			if token.Wait() && token.Error() == nil {
				state.state = STATE_RECOVERING
			}
		case STATE_RECOVERING:
			if !state.epoch_hdata_received {
				state.state = STATE_DESYNCED
			}
		case STATE_GOOD:
		}
		state.epoch_hdata_received = false
		time.Sleep(TRACK_VERSION_EPOCH)
	}
}

func connect() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.KeepAlive = 300
	opts.PingTimeout = 60 * time.Second

	broker := fmt.Sprintf("tcp://%s:%d", MQTT_BROKER, MQTT_PORT)
	if mqttCertFile != "" || mqttKeyFile != "" || mqttCAFile != "" {
		log.Info().Msg("Configuring TLS")
		broker = fmt.Sprintf("tls://%s:%d", MQTT_BROKER, MQTT_PORT)
		opts.SetTLSConfig(makeMQTTTLSConfig())
	}

	opts.AddBroker(broker)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.AutoReconnect = true
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.SetClientID(MQTT_CLIENT_ID)
	if MQTT_USERNAME != "" && MQTT_PASSWORD != "" {
		opts.SetUsername(MQTT_USERNAME)
		opts.SetPassword(MQTT_PASSWORD)
	}

	client := mqtt.NewClient(opts)

	for token := client.Connect(); token.Wait() && token.Error() != nil; {
		log.Error().
			Err(token.Error()).
			Msg("Mqtt connection failed, retrying in 2 seconds")
		time.Sleep(2 * time.Second)
	}

	return client
}

func setupLog() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
}

func main() {
	setupLog()

	nodes = make(map[string]*node_state)
	aliases = make(map[string]string)
	ctx, cancel := dbCtx()
	defer cancel()
	var err error
	pool, err = pgxpool.ConnectConfig(ctx, makeDbConfig())
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	client = connect()
	go recoveryWorker()

	exit := make(chan string)
	<-exit
}
