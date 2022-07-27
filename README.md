# rapidpro-osm-extractor

Tool to extract geojson data from https://osm-boundaries.com/ to be parsed into https://github.com/rapidpro/rapidpro

## Installation

TODO: Add installation docs

## Usage

Command line arguments description:

| Argument name | Default | Required | Description |
| -- | -- | -- | -- |
| api-key | `None` | `yes` | API Key used to download GeoJSON data from osm-boundaries.com. This argument can be set using an environment variable as well `OSM_BOUNDARY_API_KEY`. |
| osm-id | `None` | `yes` | The OSM ID to download. This can be obtained from [Nominatim](https://nominatim.openstreetmap.org/). |
| admin-mapping-file | `None` | `yes` | Path to file with administrative leve)l mappings. |
| output-dir | `None` | `yes` | Path to directory to output GeoJSON data. |
| database | `osm20211227` | `no` | OSM database to use. |
| srid | `4326` | `no` | Spatial reference identifier. |
| simplify | `0.01` | `no` | Simplification level. |

Example extracting GeoJSON for country with OSM ID R12345.

```sh
rapidpro-osm-extractor --api-key my-secret-api-key --admin-mapping-file /path/to/admin_mapping.yaml --osm-id 12345 --output-dir /path/to/out-dir
```
