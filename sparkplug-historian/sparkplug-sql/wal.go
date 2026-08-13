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
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	sparkplug "github.com/normalframework/sparkplug-historian/sparkplug_b"
	"github.com/rs/zerolog/log"
)

var DATA_DIR = getEnv("DATA_DIR", "data")

var RECOVERY_STRIDE = getEnvInt("RECOVERY_STRIDE", 1000)

type record struct {
	Name [36]byte
	Ts   uint64
	Val  float64
}

// aliasRecordPrefix marks a WAL record whose metric name could not be
// resolved when it was spooled (alias map wiped / device not yet birthed).
// The number after the prefix is the Sparkplug metric alias.
const aliasRecordPrefix = "!alias:"

// errUnresolved is returned by insertRecoveryData when some records could
// not be matched to metadata rows yet. It is transient: the segment is
// kept and retried (with backoff) so data lands once the device re-births.
var errUnresolved = errors.New("some records unresolved against metadata")

// recordName decodes a WAL record name, stripping trailing NUL padding.
func recordName(r record) string {
	return strings.TrimRight(string(r.Name[:]), "\x00")
}

// recoveryBackoff tracks per-segment retry state for transient replay
// failures (DB timeouts, unresolved metadata). Files are skipped until
// their backoff expires instead of being retried every walk.
type retryState struct {
	failures  int
	nextRetry time.Time
}

var recoveryBackoff = map[string]*retryState{}

func backoffFor(failures int) time.Duration {
	d := 30 * time.Second
	for i := 1; i < failures; i++ {
		d *= 2
		if d >= time.Hour {
			return time.Hour
		}
	}
	return d
}

func logAllMetrics(node_id string, device_id string,
	metrics []*sparkplug.Payload_Metric, only_historical bool) {
	now := time.Now()

	time_bucket := fmt.Sprintf("%02d-%02d-%02dT%02d:00",
		now.Year(), now.Month(), now.Day(), now.Hour())
	os.MkdirAll(DATA_DIR+"/"+SPARKPLUG_GROUP_ID+"/"+
		node_id+"/"+
		device_id, 0750)
	f, err := os.OpenFile(filepath.Clean(DATA_DIR+"/"+SPARKPLUG_GROUP_ID+"/"+
		node_id+"/"+device_id+"/"+time_bucket),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot write historical data")
	}
	defer func() {
		f.Close()
	}()

	alias_mu.Lock()
	defer alias_mu.Unlock()
	for _, m := range metrics {
		var name string
		if m.Name != nil {
			name = *m.Name
		} else if m.Alias != nil {
			name = aliases[fmt.Sprintf("%s\x00%s\x00%d", node_id, device_id, *m.Alias)]
		} else {
			continue
		}
		if name == "" {
			// Name unresolvable (alias map wiped / device not yet
			// birthed). Spool an alias-encoded record instead of
			// dropping the data point: replay resolves the alias via
			// metadata.metric_alias once the device re-births.
			if m.Alias == nil {
				log.Debug().Msg("Can't log historical metric: no name and no alias")
				continue
			}
			name = fmt.Sprintf("%s%d", aliasRecordPrefix, *m.Alias)
		}

		if only_historical && (m.IsHistorical == nil || !*m.IsHistorical) {
			continue
		}

		if m.Timestamp == nil {
			continue
		}

		// copy truncates at 36 bytes naturally; never slice the
		// string (name[:36] panics on names shorter than 36 bytes).
		var data record
		copy(data.Name[:], name)
		data.Ts = *m.Timestamp
		data.Val = makeScalar(m)

		err = binary.Write(f, binary.LittleEndian, data)
		if err != nil {
			log.Fatal().Err(err).Msg("Error writing data")
		}
	}
}

func recoverFile(path string, info os.FileInfo, err error) error {
	if info == nil {
		log.Warn().Str("path", path).Err(err).Msg("Error during walk")
		return nil
	}
	if info.Mode()&fs.ModeDir > 0 {
		return nil
	}
	if info.Size() == 0 {
		os.Remove(path)
		return nil
	}
	dir, chunk := filepath.Split(path)
	dir, device_id := filepath.Split(filepath.Clean(dir))
	_, node_id := filepath.Split(filepath.Clean(dir))
	var year, month, day, hour int
	fmt.Sscanf(chunk, "%d-%02d-%02dT%02d:00", &year, &month, &day, &hour)
	now := time.Now()
	if strings.HasSuffix(path, ".tooold") {
		return nil
	}
	// Transient-failure backoff: skip segments that aren't due for
	// retry yet (DB timeouts, unresolved metadata) instead of
	// hammering them on every 5-second walk.
	if rs, ok := recoveryBackoff[path]; ok && now.Before(rs.nextRetry) {
		return nil
	}
	if false && !strings.HasSuffix(path, ".recovery") && year == now.Year() &&
		month == int(now.Month()) &&
		day == now.Day() && hour == now.Hour() {
		log.Debug().
			Str("path", path).
			Msg("Skipping recovery segment because it is current")
		return nil
	}
	log.Info().
		Str("path", path).
		Str("chunk", chunk).
		Msg("Recovering segment")

	if !strings.HasSuffix(path, ".recovery") {
		err := os.Rename(path, path+".recovery")
		if err != nil {
			log.Warn().Err(err).
				Str("path", path).
				Msg("Could not rename segment")
			return nil
		}
		path = path + ".recovery"
	}

	var tmp record
	var data = make([]record, info.Size()/int64(binary.Size(tmp)))
	f, err := os.OpenFile(filepath.Clean(path), os.O_RDONLY, 0600)
	if err != nil {
		log.Warn().
			Str("path", path).
			Err(err).Msg("Cannot open historical data")
		return nil
	}
	defer func() {
		f.Close()
	}()
	err = binary.Read(f, binary.LittleEndian, data)
	if err != nil {
		log.Warn().
			Str("path", path).
			Err(err).Msg("Cannot read historical data")
		return nil
	}

	err = insertRecoveryData(node_id, device_id, data)
	if err == nil {
		delete(recoveryBackoff, path)
		os.Remove(path)
	} else {
		log.Error().Err(err).
			Str("path", path).
			Msg("Recovery failed")

		errStr := err.Error()
		// Only "not permitted on chunk" (data outside TimescaleDB's
		// retention window) is unambiguously permanent — mark .tooold
		// so the worker stops retrying. Lookup misses are transient
		// (metadata appears when the device re-births) and DB errors
		// may clear, so everything else retries with capped backoff.
		if strings.Contains(errStr, "not permitted on chunk") {
			log.Warn().Str("path", path).Msg("Marking unrecoverable segment as tooold")
			delete(recoveryBackoff, path)
			os.Rename(path, path+".tooold")
		} else {
			rs := recoveryBackoff[path]
			if rs == nil {
				rs = &retryState{}
				recoveryBackoff[path] = rs
			}
			rs.failures++
			rs.nextRetry = time.Now().Add(backoffFor(rs.failures))
			log.Warn().
				Str("path", path).
				Int("failures", rs.failures).
				Time("next_retry", rs.nextRetry).
				Msg("Recovery retry scheduled with backoff")
		}
	}

	return nil
}

func insertRecoveryData(node_id string, device_id string, data []record) error {
	log.Debug().
		Str("node_id", node_id).
		Str("device_id", device_id).
		Msg("Inserting values")

	// Both lookups use INSERT...SELECT so a lookup miss inserts zero
	// rows instead of aborting the whole batch on the not-null
	// constraint. Misses are counted and reported as errUnresolved
	// (transient) so the segment is retried later with backoff.
	const insertByName = `
INSERT INTO metrics (metric_id, time, value)
SELECT id, to_timestamp($5), $6
FROM metadata
WHERE group_name = $1 AND node_name = $2 AND device_name = $3 AND metric_name = $4
ON CONFLICT (metric_id, time) DO NOTHING`
	const insertByAlias = `
INSERT INTO metrics (metric_id, time, value)
SELECT id, to_timestamp($5), $6
FROM metadata
WHERE group_name = $1 AND node_name = $2 AND device_name = $3 AND metric_alias = $4
ON CONFLICT (metric_id, time) DO NOTHING`
	// Replay in RECOVERY_STRIDE-sized chunks: segments can hold millions
	// of records (240MB+ observed in production), and a single
	// transaction would never fit inside dbCtx's 60s deadline.
	misses := 0
	for start := 0; start < len(data); start += RECOVERY_STRIDE {
		end := start + RECOVERY_STRIDE
		if end > len(data) {
			end = len(data)
		}
		chunkMisses, err := insertRecoveryChunk(node_id, device_id, data[start:end], insertByName, insertByAlias)
		if err != nil {
			return err
		}
		misses += chunkMisses
	}
	log.Info().
		Str("node_id", node_id).
		Str("device_id", device_id).
		Int("count", len(data)).
		Int("unresolved", misses).
		Msg("Recovery complete")
	if misses > 0 {
		return errUnresolved
	}
	return nil
}

func insertRecoveryChunk(node_id, device_id string, data []record, insertByName, insertByAlias string) (int, error) {
	ctx, cancel := dbCtx()
	defer cancel()

	misses := 0
	err := pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		inserts := 0

		for _, val := range data {
			name := recordName(val)
			if aliasStr, ok := strings.CutPrefix(name, aliasRecordPrefix); ok {
				alias, perr := strconv.ParseUint(aliasStr, 10, 64)
				if perr != nil {
					log.Warn().Str("record", name).Msg("Skipping malformed alias WAL record")
					continue
				}
				b.Queue(insertByAlias, SPARKPLUG_GROUP_ID,
					node_id, device_id,
					alias, val.Ts/1000, val.Val)
			} else {
				b.Queue(insertByName, SPARKPLUG_GROUP_ID,
					node_id, device_id,
					name, val.Ts/1000, val.Val)
			}
			inserts++
		}

		batchResults := tx.SendBatch(ctx, b)
		for i := 0; i < inserts; i++ {
			tag, err := batchResults.Exec()
			if err != nil {
				batchResults.Close()
				return err
			}
			// 0 rows: lookup missed (transient — retried) or the
			// row already existed (conflict — replay dedups).
			if tag.RowsAffected() == 0 {
				misses++
			}
		}

		err := batchResults.Close()
		if err != nil {
			log.Warn().Err(err).Msg("Recovery commit error")
			return err
		}
		return nil
	})
	return misses, err
}

func recoveryWorker() {
	for {
		filepath.Walk(DATA_DIR, recoverFile)
		time.Sleep(5 * time.Second)
	}
}
