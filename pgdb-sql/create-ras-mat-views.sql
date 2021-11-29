-- MATERIALIZED VIEWS FOR MODELS

-- Project files
CREATE MATERIALIZED VIEW models.ras_project_metadata AS
SELECT
    models.model_inventory_id,
    c.collection_id AS collection,
    (models.model_metadata -> 'ProjFileContents' ->> 'ProjTitle' ) AS title,
    (models.model_metadata -> 'ProjFileContents' ->> 'Description') AS description,
    (models.model_metadata -> 'ProjFileContents' ->> 'Units') AS units,
    (models.model_metadata -> 'ProjFileContents' ->> 'CurrentPlan') AS current_plan,
    models.s3_key AS s3_key
FROM models.model AS models
LEFT JOIN inventory.collections AS c USING (collection_id)
WITH DATA;

-- Plan files 
CREATE MATERIALIZED VIEW models.ras_plan_metadata AS
with plan_files as (
    SELECT
        model_inventory_id,
        json_array_elements(model_metadata -> 'PlanFiles') as metadata
    FROM models.model
    WHERE (model_metadata ->> 'PlanFiles') IS NOT NULL
)
SELECT
    model_inventory_id,
    (metadata ->> 'PlanTitle') AS plan_title,
    (metadata ->> 'FileExt') AS file_ext,
    (metadata ->> 'ProgramVersion') AS version,
    (metadata ->> 'Description') AS description,
    (metadata ->> 'ShortIdentifier') AS short_id,
    (metadata ->> 'GeomFile') AS geom_file,
    (metadata ->> 'FlowFile') AS flow_file,
    (metadata ->> 'QuasiSteadyFile') AS quasi_steady_file,
    (metadata ->> 'UnsteadyFile') AS unsteady_file,
    (metadata ->> 'FlowRegime') AS flow_regime,
    (metadata ->> 'Path') AS s3_key
FROM plan_files
WITH DATA;
 
-- Flow files 
CREATE MATERIALIZED VIEW models.ras_flow_metadata AS
with flow_files as (
    SELECT
        model_inventory_id,
        json_array_elements(model_metadata -> 'FlowFiles') as metadata
    FROM models.model
    WHERE (model_metadata ->> 'FlowFiles') IS NOT NULL
)
SELECT
    model_inventory_id,
    (metadata ->> 'FlowTitle') AS flow_title,
    (metadata ->> 'FileExt') AS file_ext,
    (metadata ->> 'ProgramVersion') AS version,
    (metadata ->> 'NProfiles') AS num_profiles,
    (metadata ->> 'ProfileNames') AS profile_names,
    (metadata ->> 'Path') AS s3_key
FROM flow_files
WITH DATA;
 
-- Geometry files 
CREATE MATERIALIZED VIEW models.ras_geometry_metadata AS
with geom_files as (
    SELECT
        model_inventory_id,
        json_array_elements(model_metadata -> 'GeomFiles') as metadata
    FROM models.model
    WHERE (model_metadata ->> 'GeomFiles') IS NOT NULL
)
SELECT
    model_inventory_id,
    (metadata ->> 'Geom Title') AS geom_title,
    (metadata ->> 'File Extension') AS file_ext,
    (metadata ->> 'Program Version') AS version,
    (metadata ->> 'Description') AS description,
    json_array_length(CASE WHEN (metadata -> 'Hydraulic Structures')::text = 'null' THEN '[]'::json ELSE (metadata -> 'Hydraulic Structures') END) as num_reaches,
    (metadata ->> 'Path') AS s3_key
FROM geom_files
WITH DATA;
 
 
 -- Rivers
CREATE MATERIALIZED VIEW models.ras_rivers_metadata AS
with geom_files as (
    SELECT
        model_inventory_id,
        json_array_elements(model_metadata -> 'GeomFiles') as metadata
    FROM models.model
    WHERE (model_metadata ->> 'GeomFiles') IS NOT NULL
),
hydraulic_structures as (
    SELECT
        model_inventory_id,
        json_array_elements(metadata -> 'Hydraulic Structures') as metadata
    FROM geom_files
    WHERE  (metadata ->> 'Hydraulic Structures') IS NOT NULL
)
SELECT
    model_inventory_id,
    (metadata ->> 'River Name') AS river_name,
    (metadata ->> 'Reach Name') AS reach_name,
    (metadata ->> 'Num CrossSections') AS num_xs,
    (metadata -> 'Culvert Data' ->> 'Num Culverts') AS num_culverts,
    (metadata-> 'Bridge Data' ->> 'Num Bridges') AS num_bridges,
    (metadata -> 'Inline Weir Data' ->> 'Num Inline Weirs') AS num_weirs
FROM hydraulic_structures
WITH DATA;

-- Convex Hull
CREATE MATERIALIZED VIEW models.ras_convexhull AS
SELECT 
    ras.model_inventory_id,
    ST_ConvexHull(ST_Union(ST_Force2D(xs.geom))) AS GEOM
FROM 
    models.ras_xs AS xs
INNER JOIN models.ras_rivers AS rivers USING (river_id)
INNER JOIN models.ras_geometry_files AS geom_files USING (geometry_file_id)
INNER JOIN models.model AS ras USING (model_inventory_id)
GROUP BY ras.model_inventory_id
WITH DATA;

-- RAS Summary Table
CREATE MATERIALIZED VIEW models.ras_current_plan_summary AS
    SELECT model_inventory_id, title, collection_description, source, model_name,
        reach_name, river_name,
        plan_file,current_plan, plan_flow_file, plan_geom_file,
        plan_model_version, flow_model_version, geometry_model_version,
        plan_title, plan_description
        geom_title, geom_description, 
        flow_title, n_profiles, profile_names,
        n_cross_sections, n_culverts, num_bridges, num_inline_wiers,files_link,map_link

    FROM (
        SELECT c.title as title, m.name as model_name, c.source, c.description as collection_description, m.model_inventory_id,
            json_array_elements(model_metadata -> 'PlanFiles') ->> 'ProgramVersion' as plan_model_version,
            json_array_elements(model_metadata -> 'PlanFiles') ->> 'FileExt' as plan_file,
            '.' ||  (model_metadata -> 'ProjFileContents' ->> 'CurrentPlan') as current_plan,
            json_array_elements(model_metadata -> 'PlanFiles') ->> 'PlanTitle' as plan_title,
            json_array_elements(model_metadata -> 'PlanFiles') ->> 'Description' as plan_description,
            'https://floodplanning.org/file_explorer?s3_prefix='  || (model_metadata ->> 'ProjFilePath')::text as files_link,
            'https://floodplanning.org/map?definition_file=' || (model_metadata ->> 'ProjFilePath')::text as map_link,
            '.' || (json_array_elements(model_metadata -> 'PlanFiles') ->> 'GeomFile')::text as plan_geom_file,
            '.' || (json_array_elements(model_metadata -> 'PlanFiles') ->> 'FlowFile')::text as plan_flow_file

        FROM models.model m           
        JOIN inventory.collections c using(collection_id) 
        WHERE m.type = 'RAS'
        AND m.model_metadata ->> 'PlanFiles' != 'null' 
        AND m.model_metadata ->> 'ProjFileContents'  != 'null' 
        
    )  plan

    JOIN (
        SELECT m.model_inventory_id,
            json_array_elements(model_metadata -> 'FlowFiles') ->> 'ProgramVersion' as flow_model_version,
            json_array_elements(model_metadata -> 'FlowFiles') ->> 'FileExt' as plan_flow_file,
            json_array_elements(model_metadata -> 'FlowFiles') ->> 'FlowTitle' as flow_title,
            json_array_elements(model_metadata -> 'FlowFiles') ->> 'NProfiles' as n_profiles,
            json_array_elements(model_metadata -> 'FlowFiles') ->> 'ProfileNames' as profile_names
        FROM models.model m           
        JOIN inventory.collections c using(collection_id) 
        WHERE m.type = 'RAS'
        AND m.model_metadata ->> 'FlowFiles' != 'null' 

    ) flow USING(model_inventory_id, plan_flow_file)

    JOIN (
        WITH query_1 AS (
                    SELECT 
                        model_inventory_id, 
                        model_metadata ->> 'GeomFiles' as geom_files
                    FROM models.model m
                    ORDER BY model_inventory_id
        ),
            query_2 AS
            (
                SELECT 
                    model_inventory_id, 
                    json_array_elements(geom_files::json) ->> 'Program Version' as geometry_model_version,
                    json_array_elements(geom_files::json) ->>  'File Extension' as plan_geom_file,
                    json_array_elements(geom_files::json) ->> 'Geom Title' as geom_title,
                    json_array_elements(geom_files::json) ->> 'Description' as geom_description,
                    json_array_elements(geom_files::json) ->>'Hydraulic Structures' as structs

                FROM query_1
                WHERE query_1.geom_files is not null
                ORDER BY model_inventory_id
            )
            SELECT
                model_inventory_id,
                geometry_model_version,
                plan_geom_file,
                geom_title,
                geom_description,
                json_array_elements(structs::json) -> 'Inline Weir Data' ->> 'Num Inline Weirs' as num_inline_wiers,
                json_array_elements(structs::json) -> 'Culvert Data' ->> 'Num Culverts' as n_culverts,
                json_array_elements(structs::json) -> 'Bridge Data' ->> 'Num Bridges' as num_bridges,
                json_array_elements(structs::json) ->> 'Num CrossSections' as n_cross_sections,
                json_array_elements(structs::json) ->> 'Reach Name' as reach_name,
                json_array_elements(structs::json) ->> 'River Name' as river_name
            FROM query_2
            WHERE structs is not null
            ORDER BY model_inventory_id, geometry_model_version

    ) geometry 

    USING(model_inventory_id, plan_geom_file)
    WHERE plan_file = current_plan
    ORDER BY model_inventory_id;
