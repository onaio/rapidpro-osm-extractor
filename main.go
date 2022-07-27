package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

type OSMOpts struct {
	ApiKey       string
	OSMDatabase  string
	OSMId        string
	SRID         string
	Simplify     string
	AdminMapFile string
	OutDir       string
}

func (o *OSMOpts) getOSMBoundaryID() string {
	return fmt.Sprintf("-%s", o.OSMId)
}

func (o *OSMOpts) getRapidProOSMID() string {
	return fmt.Sprintf("R%s", o.OSMId)
}

func main() {
	var osmOpts OSMOpts

	app := &cli.App{
		Name:  "rapidpro-osm-extractor",
		Usage: "Extract OSM geojson data and parse to RapidPro",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "api-key",
				Usage:       "API key for osm-boundaries.com",
				EnvVars:     []string{"OSM_BOUNDARY_API_KEY"},
				Required:    true,
				Destination: &osmOpts.ApiKey,
			},
			&cli.StringFlag{
				Name:        "database",
				Aliases:     []string{"db"},
				Value:       "osm20211227",
				Usage:       "OSM database",
				Destination: &osmOpts.OSMDatabase,
			},
			&cli.StringFlag{
				Name:        "osm-id",
				Aliases:     []string{"id"},
				Usage:       "OSM ID to extract",
				Required:    true,
				Destination: &osmOpts.OSMId,
			},
			&cli.StringFlag{
				Name:        "srid",
				Value:       "4326",
				Usage:       "Spatial reference identifier",
				Destination: &osmOpts.SRID,
			},
			&cli.StringFlag{
				Name:        "simplify",
				Value:       "0.01",
				Usage:       "Simplification level",
				Destination: &osmOpts.Simplify,
			},
			&cli.StringFlag{
				Name:        "admin-mapping-file",
				Usage:       "Admin mapping file path",
				Required:    true,
				Destination: &osmOpts.AdminMapFile,
			},
			&cli.StringFlag{
				Name:        "output-dir",
				Aliases:     []string{"o"},
				Usage:       "Directory to save geojson",
				Required:    true,
				Destination: &osmOpts.OutDir,
			},
		},
		Action: func(cCtx *cli.Context) error {
			err := downloadAndParseOSM(&osmOpts)
			if err != nil {
				cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
