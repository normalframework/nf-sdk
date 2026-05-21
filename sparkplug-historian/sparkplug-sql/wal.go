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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
			log.Debug().
				Str("name", name).
				Msg("Can't log historical metric due to invalid name")
			continue
		}

		if only_historical && (m.IsHistorical == nil || !*m.IsHistorical) {
			continue
		}

		var data = struct {
			name [36]byte
			n1   uint64
			n2   float64
		}{}
		copy(data.name[:], name[:36])
		data.n1 = *m.Timestamp
		data.n2 = makeScalar(m)

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
		os.Remove(path)
	} else {
		log.Error().Err(err).
			Str("path", path).
			Msg("Recovery failed")

		if strings.Contains(err.Error(), "not permitted on chunk") {
			os.Rename(path, path+".tooold")
		}
	}

	return nil
}

func insertRecoveryData(node_id string, device_id string, data []record) error {
	log.Debug().
		Str("node_id", node_id).
		Str("device_id", device_id).
		Msg("Inserting values")

	const insertStatement = `
INSERT INTO metrics (metric_id, time, value) VALUES (
 (SELECT id FROM metadata WHERE group_name = $1 AND
   node_name = $2 AND device_name = $3 AND metric_name = $4), to_timestamp($5), $6)
ON CONFLICT (metric_id, time) DO NOTHING`
	ctx, cancel := dbCtx()
	defer cancel()

	err := pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		inserts := 0

		for _, val := range data {
			b.Queue(insertStatement, SPARKPLUG_GROUP_ID,
				node_id, device_id,
				string(val.Name[:]), val.Ts/1000, val.Val)
			inserts++
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
			log.Warn().Err(err).Msg("Recovery commit error")
			return err
		}
		log.Info().
			Str("node_id", node_id).
			Str("device_id", device_id).
			Int("count", inserts).
			Msg("Recovery complete")

		return nil
	})
	return err
}

func recoveryWorker() {
	for {
		filepath.Walk(DATA_DIR, recoverFile)
		time.Sleep(5 * time.Second)
	}
}
