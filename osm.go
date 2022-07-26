package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// OSMGeoJSON represents structure expected from https://osm-boundaries.com/
type OSMGeoJSON struct {
	Type     string `json:"type"`
	Features []struct {
		Type     string `json:"type"`
		Geometry struct {
			Type        string      `json:"type"`
			Coordinates interface{} `json:"coordinates"`
		} `json:"geometry"`
		Properties struct {
			OSMID      int    `json:"osm_id"`
			Boundary   string `json:"boundary"`
			AdminLevel int    `json:"admin_level"`
			Parents    string `json:"parents"`
			Name       string `json:"name"`
			LocalName  string `json:"local_name"`
			NameEn     string `json:"name_en"`
		} `json:"properties"`
	} `json:"features"`
}

// RapidProGeoJSON represents structure expected to be parsed into RapidPro
type RapidProGeoJSON struct {
	Type     string                   `json:"type"`
	Features []RapidProGeoJSONFeature `json:"features"`
}

type RapidProGeoJSONFeature struct {
	Type       string                    `json:"type"`
	Properties RapidProGeoJSONProperties `json:"properties"`
	Geometry   struct {
		Type        string      `json:"type"`
		Coordinates interface{} `json:"coordinates"`
	} `json:"geometry"`
}

type RapidProGeoJSONProperties struct {
	OSMID       string `json:"osm_id"`
	Name        string `json:"name"`
	NameEn      string `json:"name_en"`
	IsInCountry string `json:"is_in_country"`
	IsInState   string `json:"is_in_state"`
}

type AdminMapping struct {
	Default struct {
		AdminLevel0 int `yaml:"admin_level_0"`
		AdminLevel1 int `yaml:"admin_level_1"`
		AdminLevel2 int `yaml:"admin_level_2"`
	} `yaml:"default"`
	PerCountry map[string]struct {
		AdminLevel1 int `yaml:"admin_level_1"`
		AdminLevel2 int `yaml:"admin_level_2"`
		Meta        struct {
			Name string `yaml:"name"`
		} `yaml:"meta"`
	} `yaml:"per_country"`
}

type OSMAdminLevel struct {
	AdminLevel0 int
	AdminLevel1 int
	AdminLevel2 int
}

func getOSMAdminLevels(osmID string, mapFilePath string) (*OSMAdminLevel, error) {
	var adminMapping AdminMapping

	mapFile, readFileErr := ioutil.ReadFile(mapFilePath)
	if readFileErr != nil {
		return nil, fmt.Errorf("could not read mapping file : %v", readFileErr)
	}

	unmarshalErr := yaml.Unmarshal(mapFile, &adminMapping)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("could not unmarshal mapping file : %v", unmarshalErr)
	}

	osmAdminLevel := &OSMAdminLevel{
		AdminLevel0: adminMapping.Default.AdminLevel0,
		AdminLevel1: adminMapping.Default.AdminLevel1,
		AdminLevel2: adminMapping.Default.AdminLevel2,
	}

	// check if OSM ID is present in mapping
	if mapVal, exists := adminMapping.PerCountry[osmID]; exists {
		osmAdminLevel.AdminLevel1 = mapVal.AdminLevel1
		osmAdminLevel.AdminLevel2 = mapVal.AdminLevel2
	}

	return osmAdminLevel, nil
}

func download(client *http.Client, osmOpts *OSMOpts, adminLevel int) (OSMGeoJSON, error) {
	var osmGeoJSON OSMGeoJSON

	req, err := http.NewRequest("GET", "https://osm-boundaries.com/Download/Submit", nil)
	if err != nil {
		return osmGeoJSON, fmt.Errorf("Error creating new request : %v", err)
	}

	query := req.URL.Query()
	query.Add("apiKey", osmOpts.ApiKey)
	query.Add("db", osmOpts.OSMDatabase)
	query.Add("osmIds", osmOpts.getOSMBoundaryID())
	query.Add("minAdminLevel", strconv.Itoa(adminLevel))
	query.Add("maxAdminLevel", strconv.Itoa(adminLevel))
	query.Add("format", "GeoJSON")
	query.Add("srid", osmOpts.SRID)
	query.Add("simplify", osmOpts.Simplify)
	query.Add("recursive", "")
	req.URL.RawQuery = query.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return osmGeoJSON, fmt.Errorf("Error sending request : %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return osmGeoJSON, fmt.Errorf("Error reading body : %v", err)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return osmGeoJSON, fmt.Errorf("Error creating gzip reader : %v", err)
	}
	defer gzipReader.Close()

	contents, err := ioutil.ReadAll(gzipReader)
	if err != nil {
		return osmGeoJSON, err
	}

	err = json.Unmarshal(contents, &osmGeoJSON)
	if err != nil {
		return osmGeoJSON, fmt.Errorf("Error decoding OSM GeoJSON : %v", err)
	}

	return osmGeoJSON, nil
}

func osmToRapidPro(osmGeoJSON *OSMGeoJSON, countryOSMId string, stateOSMIds []string) RapidProGeoJSON {
	rpGeoJSON := RapidProGeoJSON{}
	rpGeoJSON.Type = osmGeoJSON.Type

	rpFeatures := []RapidProGeoJSONFeature{}
	for _, feature := range osmGeoJSON.Features {
		// Converts OSM ID e.g -3247585 to RapidPro OSM ID e.g R3247585
		rpOSMId := rpOSMIdConv(strconv.Itoa(feature.Properties.OSMID))

		// Determine which state the location belongs in. Defaults to "None" if it doesn't belong
		// to any states
		stateOSMId := "None"
		parents := strings.Split(feature.Properties.Parents, ",")
		intersect := intersection(stateOSMIds, parents)
		if len(intersect) != 0 {
			stateOSMId = rpOSMIdConv(intersect[0])
		}

		rpFeatures = append(rpFeatures, RapidProGeoJSONFeature{
			Type:     feature.Type,
			Geometry: feature.Geometry,
			Properties: RapidProGeoJSONProperties{
				OSMID:       rpOSMId,
				Name:        feature.Properties.LocalName,
				NameEn:      feature.Properties.Name,
				IsInCountry: rpOSMIdConv(countryOSMId),
				IsInState:   stateOSMId,
			},
		})
	}

	rpGeoJSON.Features = rpFeatures
	return rpGeoJSON
}

func writeToOutDir(rpGeoJSON *RapidProGeoJSON, outDir, filename string) error {
	geoJSONFile, err := json.Marshal(rpGeoJSON)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(outDir, filename), geoJSONFile, 0644)
}

func downloadAndParseOSM(osmOpts *OSMOpts) error {
	osmAdminLevels, err := getOSMAdminLevels(osmOpts.getRapidProOSMID(), osmOpts.AdminMapFile)
	if err != nil {
		return err
	}

	client := &http.Client{}

	adminLevel0GeoJSON, err := download(client, osmOpts, osmAdminLevels.AdminLevel0)
	if err != nil {
		return err
	}
	rpAdminLevel0GeoJSON := osmToRapidPro(&adminLevel0GeoJSON, "None", []string{})

	adminLevel1GeoJSON, err := download(client, osmOpts, osmAdminLevels.AdminLevel1)
	if err != nil {
		return err
	}
	countryOSMId := strconv.Itoa(adminLevel0GeoJSON.Features[0].Properties.OSMID)
	rpAdminLevel1GeoJSON := osmToRapidPro(&adminLevel1GeoJSON, countryOSMId, []string{})

	adminLevel2GeoJSON, err := download(client, osmOpts, osmAdminLevels.AdminLevel2)
	if err != nil {
		return err
	}
	stateOSMIds := []string{}
	for _, feature := range adminLevel1GeoJSON.Features {
		stateOSMIds = append(stateOSMIds, strconv.Itoa(feature.Properties.OSMID))
	}
	rpAdminLevel2GeoJSON := osmToRapidPro(&adminLevel2GeoJSON, countryOSMId, stateOSMIds)

	// write to output directory
	filenamePrefix := rpOSMIdConv(countryOSMId)

	adminLevel0FileName := fmt.Sprintf("%sadmin0_simplified.json", filenamePrefix)
	err = writeToOutDir(&rpAdminLevel0GeoJSON, osmOpts.OutDir, adminLevel0FileName)
	if err != nil {
		return err
	}
	adminLevel1FileName := fmt.Sprintf("%sadmin1_simplified.json", filenamePrefix)
	err = writeToOutDir(&rpAdminLevel1GeoJSON, osmOpts.OutDir, adminLevel1FileName)
	if err != nil {
		return err
	}
	adminLevel2FileName := fmt.Sprintf("%sadmin2_simplified.json", filenamePrefix)
	err = writeToOutDir(&rpAdminLevel2GeoJSON, osmOpts.OutDir, adminLevel2FileName)
	if err != nil {
		return err
	}

	return nil
}
