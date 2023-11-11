DROP VIEW IF EXISTS bacnet_metadata;
CREATE VIEW bacnet_metadata AS (
SELECT
  id metric_id,
  node_name,
  (properties->'device_id'->>'stringValue')::int device_id,
  (properties->'device_uuid'->>'stringValue') device_uuid,
  (properties->'scanned_at'->>'stringValue')::timestamp scanned_at,
  (properties->'prop_object_name'->>'stringValue') object_name,
  (properties->'prop_description'->>'stringValue') object_description,
  (properties->'type'->>'stringValue') object_type,
  (properties->'instance'->>'stringValue') object_instance,
  (properties->'device_prop_object_name'->>'stringValue') device_name,
  (properties->'device_prop_description'->>'stringValue') device_description,
  (properties->'device_prop_model_name'->>'stringValue') device_model_name,
  (properties->'device_prop_vendor_name'->>'stringValue') device_vendor_name,
  (properties->'device_prop_application_software_version'->>'stringValue') device_application_software_version,
  (properties->'device_prop_location'->>'stringValue') device_location,  
  (properties->'prop_low_limit'->>'stringValue')::float low_limit,
  (properties->'prop_high_limit'->>'stringValue')::float high_limit,
  (properties->'prop_min_pres_value'->>'stringValue')::float min_pres_value,
  (properties->'prop_max_pres_value'->>'stringValue')::float max_pres_value,
  (properties->'class'->>'stringValue') className,
  (properties->'equipRef'->>'stringValue') equipRef,
  string_to_array(properties->'markers'->>'stringValue', ',') markers
FROM
  metadata 
)
;

CREATE VIEW public.data_replication_status AS
 SELECT nodes.node_name,
    ('1970-01-01 00:00:00'::timestamp without time zone + ((((nodes.lgsn >> 10) / 1000))::double precision * '00:00:01'::interval)) AS lgsm_time
   FROM public.nodes;


CREATE FUNCTION array_distinct(anyarray) RETURNS anyarray AS $f$
  SELECT array_agg(DISTINCT x) FROM unnest($1) t(x);
$f$ LANGUAGE SQL IMMUTABLE;


select
	device_vendor_name,
	device_model_name,
	array_distinct(array_agg(node_name)) as sites,
	count(*) as object_count,
	count(distinct (node_name, device_id)) as device_count
from
	bacnet_metadata
group by
      device_vendor_name,
      device_model_name;


with object_counts as (
SELECT
	device_vendor_name,
	device_model_name,
	node_name,
	device_id,
	object_type,
	count(*) as cnt,
	array_distinct(array_agg(node_name)) as sites
FROM
	bacnet_metadata
GROUP BY
      node_name,
      device_vendor_name,
      device_model_name,
      device_id,
      object_type),
device_vectors as (
SELECT
	device_vendor_name,
	device_model_name,
	node_name,
	device_id,
	array_agg(cnt) as object_counts,
	array_agg(object_type) as object_types
FROM object_counts
GROUP BY 	device_vendor_name,
	device_model_name,
node_name, device_id)

select
	device_vendor_name,
	device_model_name,
	array_distinct(array_agg(node_name)),
	count(distinct (node_name, device_id)) as device_count,
	object_counts
from device_vectors
group by
      device_vendor_name,
      device_model_name,
      object_counts
;
