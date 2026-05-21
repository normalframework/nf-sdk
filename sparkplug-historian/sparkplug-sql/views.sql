CREATE OR REPLACE VIEW point_metadata AS
(SELECT metadata.id         AS metric_id,
       metadata.node_name,
       ((metadata.properties -> 'uuid'::text) -> 'Value'::text) ->>
       'StringValue'::text AS uuid,
       ((metadata.properties -> 'name'::text) -> 'Value'::text) ->>
       'StringValue'::text AS name,
       ((metadata.properties -> 'parent_name'::text) -> 'Value'::text) ->>
       'StringValue'::text AS parent_name,
       ((metadata.properties -> 'parent_uuid'::text) -> 'Value'::text) ->>
       'StringValue'::text AS parent_uuid,
       ((metadata.properties -> 'protocol_id'::text) -> 'Value'::text) ->>
       'StringValue'::text AS protocol_id,
       ((metadata.properties -> 'display_units'::text) -> 'Value'::text) ->>
       'StringValue'::text AS display_units
FROM metadata
);

CREATE OR REPLACE VIEW bacnet_metadata AS
(
SELECT id                                                                                     metric_id,
       node_name,
       (properties -> 'hpl:bacnet:1.device_id' -> 'Value' ->> 'StringValue')::int             device_id,
       (properties -> 'hpl:bacnet:1.device_uuid' -> 'Value' ->> 'StringValue')::text          device_uuid,
       (properties -> 'hpl:bacnet:1.scanned_at' -> 'Value' ->> 'StringValue')::timestamp      scanned_at,
       (properties -> 'hpl:bacnet:1.prop_object_name' -> 'Value' ->> 'StringValue')           object_name,
       (properties -> 'hpl:bacnet:1.prop_description' -> 'Value' ->> 'StringValue')           object_description,
       (properties -> 'hpl:bacnet:1.type' -> 'Value' ->> 'StringValue')                       object_type,
       (properties -> 'hpl:bacnet:1.instance' -> 'Value' ->> 'StringValue')                   object_instance,
       (properties -> 'hpl:bacnet:1.device_prop_object_name' -> 'Value' ->> 'StringValue')    device_name,
       (properties -> 'hpl:bacnet:1.device_prop_description' -> 'Value' ->> 'StringValue')    device_description,
       (properties -> 'hpl:bacnet:1.device_prop_model_name' -> 'Value' ->> 'StringValue')     device_model_name,
       (properties -> 'hpl:bacnet:1.device_prop_vendor_name' -> 'Value' ->> 'StringValue')    device_vendor_name,
       (properties -> 'hpl:bacnet:1.device_prop_application_software_version' -> 'Value' ->>
        'StringValue')                                                                        device_application_software_version,
       (properties -> 'hpl:bacnet:1.device_prop_location' -> 'Value' ->> 'StringValue')       device_location,
       (properties -> 'hpl:bacnet:1.prop_low_limit' -> 'Value' ->> 'StringValue')::float      low_limit,
       (properties -> 'hpl:bacnet:1.prop_high_limit' -> 'Value' ->> 'StringValue')::float     high_limit,
       (properties -> 'hpl:bacnet:1.prop_min_pres_value' -> 'Value' ->> 'StringValue')::float min_pres_value,
       (properties -> 'hpl:bacnet:1.prop_max_pres_value' -> 'Value' ->> 'StringValue')::float max_pres_value

FROM metadata
    );

CREATE OR REPLACE VIEW model_metadata AS
(
SELECT metadata.id                                                                                   AS metric_id,
       metadata.node_name,
       ((metadata.properties -> 'model.equipTypeId'::text) -> 'Value'::text) ->> 'StringValue'::text AS equip_type,
       ((metadata.properties -> 'model.markers'::text) -> 'Value'::text) ->> 'StringValue'::text     AS markers,
       ((metadata.properties -> 'model.equipRef'::text) -> 'Value'::text) ->> 'StringValue'::text    AS equipref,
       ((metadata.properties -> 'model.class'::text) -> 'Value'::text) ->> 'StringValue'::text       as class
FROM metadata
);


CREATE OR REPLACE VIEW public.data_replication_status AS
SELECT nodes.node_name,
       ('1970-01-01 00:00:00'::timestamp without time zone +
        ((((nodes.lgsn >> 10) / 1000))::double precision * '00:00:01'::interval)) AS lgsm_time
FROM public.nodes;

CREATE
OR REPLACE FUNCTION array_distinct(anyarray) RETURNS anyarray AS
$f$
SELECT array_agg(DISTINCT x)
FROM unnest($1) t(x);
$f$
LANGUAGE SQL IMMUTABLE;


CREATE
OR REPLACE FUNCTION get_element_list(p_value json, p_keyname text) RETURNS text AS
$f$
select string_agg(x.val ->> p_keyname, ',')
from json_array_elements(p_value) as x(val);
$f$
LANGUAGE SQL IMMUTABLE;


-- with select
--     device_vendor_name
--    , device_model_name
--    , array_distinct(array_agg(node_name)) as sites
--    , count (*) as object_count
--    , count (distinct (node_name
--    , device_id)) as device_count
-- from
--     bacnet_metadata
-- group by
--     device_vendor_name,
--     device_model_name
-- ;


with object_counts as (SELECT device_vendor_name,
                              device_model_name,
                              node_name,
                              device_id,
                              object_type,
                              count(*)                             as cnt,
                              array_distinct(array_agg(node_name)) as sites
                       FROM bacnet_metadata
                       GROUP BY node_name,
                                device_vendor_name,
                                device_model_name,
                                device_id,
                                object_type),
     device_vectors as (SELECT device_vendor_name,
                               device_model_name,
                               node_name,
                               device_id,
                               array_agg(cnt)         as object_counts,
                               array_agg(object_type) as object_types
                        FROM object_counts
                        GROUP BY device_vendor_name,
                                 device_model_name,
                                 node_name, device_id)

select device_vendor_name,
       device_model_name,
       array_distinct(array_agg(node_name)),
       count(distinct (node_name, device_id)) as device_count,
       object_counts
from device_vectors
group by device_vendor_name,
         device_model_name,
         object_counts;


CREATE OR REPLACE VIEW equipment_types AS
(

SELECT id,
       node_name,
       (value ->> 'name')::text                     name,
       (value ->> 'class_name')::text               class,
       (value ->> 'description')::text              description,
       get_element_list(value -> 'markers', 'name') markers
FROM ontology
    );



CREATE OR REPLACE VIEW equips AS
(

SELECT name                                                                 uuid,
       node_name,
       (metadata -> 'model.id' -> 'Value' ->> 'StringValue')::text          id,
       (metadata -> 'model.class' -> 'Value' ->> 'StringValue')::text       class,
       (metadata -> 'model.equipTypeId' -> 'Value' ->> 'StringValue')::text equipTypeId,
       (metadata -> 'model.markers' -> 'Value' ->> 'StringValue')::text     markers
FROM node_metadata
    );



CREATE OR REPLACE VIEW equips_points_stat AS
(

SELECT e.id, e.equiptypeid, COUNT(mm.class)
FROM equips e
         LEFT JOIN model_metadata mm ON e.id = mm.equipref
GROUP BY e.id, e.equiptypeid
    );
