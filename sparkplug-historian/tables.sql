
DROP TABLE metadata CASCADE;
drop table metrics CASCADE;


CREATE TABLE IF NOT EXISTS nodes (
      group_name        text NOT NULL,
      node_name         text NOT NULL,
      lgsn              bigint,
      PRIMARY KEY (group_name, node_name)
);

CREATE TABLE IF NOT EXISTS metadata (
      id                serial UNIQUE,
      group_name        text NOT NULL, --
      node_name         text NOT NULL,
      device_name       text NOT NULL,
      metric_name       text NOT NULL,
      metric_alias      bigint,
      last_dbirth       TIMESTAMPTZ,
      last_ddeath       TIMESTAMPTZ,
      dbirth_seq        INT,
      -- holds the PropertySet received in the last DBIRTH
      properties        json DEFAULT '{}',
      PRIMARY KEY (group_name, node_name, device_name, metric_name)
);

CREATE INDEX IF NOT EXISTS metadata_index ON metadata (group_name, node_name, device_name, metric_alias);

-- holds the raw metric data, converted to scalar
CREATE TABLE IF NOT EXISTS metrics (
      metric_id         INT REFERENCES metadata(id),
      time              TIMESTAMPTZ,
      insert_time       TIMESTAMPTZ DEFAULT NOW(),
      value             double precision --don't support strings or types for now ...
);

-- unique index prevents duplicate data from being inserted into the
-- table
CREATE UNIQUE INDEX IF NOT EXISTS metrics_index ON metrics (metric_id, time);

