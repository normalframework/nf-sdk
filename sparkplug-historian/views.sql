CREATE VIEW bacnet_metadata AS (
SELECT
  id metric_id,
  node_name,
  (properties->'device_id'->'Value'->>'StringValue')::int device_id,
  (properties->'device_uuid'->'Value'->>'StringValue')::int device_uuid,
  (properties->'scanned_at'->'Value'->>'StringValue')::timestamp scanned_at,
  (properties->'prop_object_name'->'Value'->>'StringValue') object_name,
  (properties->'prop_description'->'Value'->>'StringValue') object_description,
  (properties->'type'->'Value'->>'StringValue') object_type,
  (properties->'instance'->'Value'->>'StringValue') object_instance,
  (properties->'device_prop_object_name'->'Value'->>'StringValue') device_name,
  (properties->'device_prop_description'->'Value'->>'StringValue') device_description,
  (properties->'device_prop_model_name'->'Value'->>'StringValue') device_model_name,
  (properties->'device_prop_vendor_name'->'Value'->>'StringValue') device_vendor_name,
  (properties->'device_prop_application_software_version'->'Value'->>'StringValue') device_application_software_version,
  (properties->'device_prop_location'->'Value'->>'StringValue') device_location,  
  (properties->'prop_low_limit'->'Value'->>'StringValue')::float low_limit,
  (properties->'prop_high_limit'->'Value'->>'StringValue')::float high_limit,
  (properties->'prop_min_pres_value'->'Value'->>'StringValue')::float min_pres_value,
  (properties->'prop_max_pres_value'->'Value'->>'StringValue')::float max_pres_value
  
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
