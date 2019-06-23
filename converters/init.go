package converters

import (
	"bufio"
	proj "github.com/xeonx/proj4"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var EpsgDatabase = make(map[int]*EpsgProjection)
var GH = NewDefaultEarthGravitationalModel()

// Initialization of conversion libraries and data sources
func init() {
	//Executable path
	ex, err := os.Executable()
	if err != nil {
		log.Fatal("cannot retrieve executable directory", err)
	}
	exPath := filepath.Dir(ex)

	// Loading Earth Gravitational Model data
	err = GH.load(path.Join(exPath, "static","egm180.nor"))
	if err != nil {
		log.Fatal("cannot initialize earth gravitational model", err)
	}

	// Set path for retrieving projection static data
	proj.SetFinder([]string{path.Join(exPath, "static\\share")})

	// Initialization of EPSG Proj4 database
	file, err := os.Open(path.Join(exPath, "static\\epsg_projections.txt"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, "\t")
		code, err := strconv.Atoi(strings.Replace(tokens[0], "EPSG:", "", -1))
		if err != nil {
			log.Fatal("error while parsing the epsg projection file", err)
		}
		desc := tokens[1]
		proj4 := tokens[2]

		EpsgDatabase[code] = &EpsgProjection{
			EpsgCode:    code,
			Description: desc,
			Proj4:       proj4,
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
