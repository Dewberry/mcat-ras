package tools

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/USACE/filestore"
	"github.com/dewberry/gdal"
)

// GeomFileContents keywords  and data container for ras flow file search
type GeomFileContents struct {
	Path           string
	FileExt        string                `json:"File Extension"`
	GeomTitle      string                `json:"Geom Title"`
	ProgramVersion string                `json:"Program Version"`
	Description    string                `json:"Description"`
	Structures     []hydraulicStructures `json:"Hydraulic Structures"`
}

type hydraulicStructures struct {
	River       string     `json:"River Name"`
	Reach       string     `json:"Reach Name"`
	NumXS       int        `json:"Num CrossSections"`
	NumCulverts int        `json:"Num Culverts"`
	BridgeData  bridgeData `json:"Bridges"`
	NumInlines  int        `json:"Num Inlines"`
}

type bridgeData struct {
	NumBridges int       `json:"Num Bridges"`
	Bridges    []bridges `json:"Bridges"`
}

type bridges struct {
	Name          string
	Station       float64
	Description   string
	DeckWidth     float64    `json:"Deck Width"`
	UpHighChord   chordPairs `json:"Upstream High Chord"`
	UpLowChord    chordPairs `json:"Upstream Low Chord"`
	DownHighChord chordPairs `json:"Downstream High Chord"`
	DownLowChord  chordPairs `json:"Downstream Max Chord"`
	NumPiers      int        `json:"Num Piers"`
}

type chordPairs struct {
	Max float64
	Min float64
}

func maxValue(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0.0, errors.New("Cannot detect a maximum value in an empty slice")
	}

	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	return max, nil
}

func minValue(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0.0, errors.New("Cannot detect a minimum value in an empty slice")
	}

	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}

	return min, nil
}

func rightofEquals(line string) string {
	return strings.TrimSpace(strings.Split(line, "=")[1])
}

func getDescription(sc *bufio.Scanner, idx int, endLine string) (string, int, error) {
	description := ""
	nLines := 0
	for sc.Scan() {
		line := sc.Text()
		idx++
		endDescription, err := regexp.MatchString(endLine, line)
		if err != nil {
			return description, idx, err
		}
		if endDescription {
			return description, idx, nil
		}
		if line != "" {
			if nLines > 0 {
				description += "\n"
			}
			description += line
			nLines++
		}
	}
	return description, idx, nil
}
func numberofLines(nValues int, colWidth int, valueWidth int) int {
	nLines := math.Ceil(float64(nValues) / (float64(colWidth) / float64(valueWidth)))
	return int(nLines)
}

func datafromTextBlock(hsSc *bufio.Scanner, colWidth int, valueWidth int, nLines int, nSkipLines int) ([]float64, error) {
	values := []float64{}
	nSkipped := 0
	nProcessed := 0
out:
	for hsSc.Scan() {
		if nSkipped < nSkipLines {
			nSkipped++
			continue
		}
		nProcessed++
		line := hsSc.Text()
		for s := 0; s < colWidth; {
			if len(line) > s {
				sVal := strings.TrimSpace(line[s : s+valueWidth])
				if sVal != "" {
					val, err := strconv.ParseFloat(sVal, 64)
					if err != nil {
						return values, err
					}
					values = append(values, val)
				}
				s += valueWidth
			} else {
				if nLines == nProcessed {
					break out
				}
				break
			}

		}
		if nLines == nProcessed {
			break out
		}
	}
	return values, nil
}

func getHighLowChord(hsSc *bufio.Scanner, nElevsText string, colWidth int, valueWidth int) ([2]chordPairs, error) {
	highLowPairs := [2]chordPairs{}

	nElevs, err := strconv.Atoi(strings.TrimSpace(nElevsText))
	if err != nil {
		return highLowPairs, err
	}

	nLines := numberofLines(nElevs, colWidth, valueWidth)

	elevHighChord, err := datafromTextBlock(hsSc, colWidth, valueWidth, nLines, nLines)
	if err != nil {
		return highLowPairs, err
	}

	maxHighCord, err := maxValue(elevHighChord)
	if err != nil {
		return highLowPairs, err
	}

	minHighCord, err := minValue(elevHighChord)
	if err != nil {
		return highLowPairs, err
	}
	highLowPairs[0] = chordPairs{Max: maxHighCord, Min: minHighCord}

	elevLowChord, err := datafromTextBlock(hsSc, 80, 8, nLines, 0)
	if err != nil {
		return highLowPairs, err
	}

	maxLowCord, err := maxValue(elevLowChord)
	if err != nil {
		return highLowPairs, err
	}

	minLowCord, err := minValue(elevLowChord)
	if err != nil {
		return highLowPairs, err
	}
	highLowPairs[1] = chordPairs{Max: maxLowCord, Min: minLowCord}
	return highLowPairs, nil
}

func getBridgeData(hsSc *bufio.Scanner, lineData []string) (bridges, error) {
	bridge := bridges{}

	station, err := strconv.ParseFloat(strings.TrimSpace(lineData[1]), 64)
	if err != nil {
		return bridge, err
	}
	bridge.Station = station

	for hsSc.Scan() {
		line := hsSc.Text()
		switch {
		case strings.HasPrefix(line, "BEGIN DESCRIPTION"):
			description, _, err := getDescription(hsSc, 0, "END DESCRIPTION:")
			if err != nil {
				return bridge, err
			}
			bridge.Description += description

		case strings.HasPrefix(line, "Node Name="):
			bridge.Name = rightofEquals(line)

		case strings.HasPrefix(line, "Deck Dist"):
			hsSc.Scan()
			nextLineData := strings.Split(hsSc.Text(), ",")
			deckWidth, err := strconv.ParseFloat(strings.TrimSpace(nextLineData[0]), 64)
			if err != nil {
				return bridge, err
			}
			bridge.DeckWidth = deckWidth
			upHighLowPair, err := getHighLowChord(hsSc, nextLineData[4], 80, 8)
			if err != nil {
				return bridge, err
			}
			bridge.UpHighChord = upHighLowPair[0]
			bridge.UpLowChord = upHighLowPair[1]

			downHighLowPair, err := getHighLowChord(hsSc, nextLineData[5], 80, 8)
			if err != nil {
				return bridge, err
			}
			bridge.DownHighChord = downHighLowPair[0]
			bridge.DownLowChord = downHighLowPair[1]

		case strings.HasPrefix(line, "Pier Skew"):
			bridge.NumPiers++

		case strings.HasPrefix(line, "BR Coef"):
			return bridge, err
		}
	}
	return bridge, nil
}

func getHydraulicStructureData(rm *RasModel, fn string, idx int) (hydraulicStructures, error) {
	structures := hydraulicStructures{}
	bData := bridgeData{}

	newf, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return structures, nil
	}
	defer newf.Close()

	hsSc := bufio.NewScanner(newf)

	i := 0
	for hsSc.Scan() {
		if i == idx {
			riverReach := strings.Split(rightofEquals(hsSc.Text()), ",")
			structures.River = strings.TrimSpace(riverReach[0])
			structures.Reach = strings.TrimSpace(riverReach[1])
		} else if i > idx {
			line := hsSc.Text()
			if strings.HasPrefix(line, "River Reach=") {
				structures.BridgeData = bData
				return structures, nil
			}
			if strings.HasPrefix(line, "Type RM Length L Ch R =") {
				data := strings.Split(rightofEquals(line), ",")
				structureType, err := strconv.Atoi(strings.TrimSpace(data[0]))
				if err != nil {
					return structures, err
				}
				switch structureType {
				case 1:
					structures.NumXS++

				case 2:
					structures.NumCulverts++

				case 3:
					bridge, err := getBridgeData(hsSc, data)
					if err != nil {
						return structures, err
					}
					bData.Bridges = append(bData.Bridges, bridge)
					bData.NumBridges++

				case 5:
					structures.NumInlines++

				}
			}
		}
		i++
	}
	structures.BridgeData = bData

	return structures, nil
}

// getGeomData Reads a geometry file. returns none to allow concurrency
func getGeomData(rm *RasModel, fn string, wg *sync.WaitGroup, errChan chan error) {

	defer wg.Done()

	meta := GeomFileContents{Path: fn, FileExt: filepath.Ext(fn)}

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		errChan <- err
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)

	var description string

	header := true
	idx := 0
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "Geom Title="):
			meta.GeomTitle = rightofEquals(line)

		case strings.HasPrefix(line, "Program Version="):
			meta.ProgramVersion = rightofEquals(line)

		case strings.HasPrefix(line, "BEGIN GEOM DESCRIPTION:"):
			if header {
				description, idx, err = getDescription(sc, idx, "END GEOM DESCRIPTION:")
				if err != nil {
					errChan <- err
					return
				}
				meta.Description += description
			}

		case strings.HasPrefix(line, "River Reach="):
			structures, err := getHydraulicStructureData(rm, fn, idx)
			if err != nil {
				errChan <- err
				return
			}
			meta.Structures = append(meta.Structures, structures)
			header = false

		case strings.HasPrefix(line, "Storage Area="):
			header = false
		}
		idx++
	}
	rm.Metadata.GeomFiles = append(rm.Metadata.GeomFiles, meta)
	return
}

// sourceCRS
var sourceCRS string = `PROJCS["NAD_1983_StatePlane_Maryland_FIPS_1900_Feet",GEOGCS["GCS_North_American_1983",DATUM["D_North_American_1983",SPHEROID["GRS_1980",6378137.0,298.257222101]],PRIMEM["Greenwich",0.0],UNIT["Degree",0.0174532925199433]],PROJECTION["Lambert_Conformal_Conic"],PARAMETER["False_Easting",1312333.333333333],PARAMETER["False_Northing",0.0],PARAMETER["Central_Meridian",-77.0],PARAMETER["Standard_Parallel_1",38.3],PARAMETER["Standard_Parallel_2",39.45],PARAMETER["Latitude_Of_Origin",37.66666666666666],UNIT["Foot_US",0.3048006096012192]]`

// DestinationCRS ...
var DestinationCRS int = 4326

// GeoData ...
type GeoData struct {
	Features     map[string]Features
	Georeference int
}

// Features ...
type Features struct {
	Rivers              []vectorLayer
	XS                  []vectorLayer
	Banks               []vectorLayer
	StorageAreas        []vectorLayer
	TwoDAreas           []vectorLayer
	HydraulicStructures []vectorLayer
}

type vectorLayer struct {
	FeatureName string                 `json:"feature_name"`
	Fields      map[string]interface{} `json:"fields"`
	Geometry    []uint8                `json:"geometry"`
}

type xyzPoint struct {
	x float64
	y float64
	z float64
}

func dataPairsfromTextBlock(sc *bufio.Scanner, nPairs int, colWidth int, valueWidth int) ([][2]float64, error) {
	var stride int = valueWidth * 2
	pairs := [][2]float64{}
out:
	for sc.Scan() {
		line := sc.Text()
		for s := 0; s < colWidth; {
			if len(line) > s {
				val1, err := strconv.ParseFloat(strings.TrimSpace(line[s:s+valueWidth]), 64)
				if err != nil {
					return pairs, err
				}
				val2, err := strconv.ParseFloat(strings.TrimSpace(line[s+valueWidth:s+stride]), 64)
				if err != nil {
					return pairs, err
				}
				pairs = append(pairs, [2]float64{val1, val2})
				if len(pairs) == nPairs {
					break out
				}
			} else {
				break
			}
			s += stride
		}
	}
	return pairs, nil
}

func getDataPairsfromTextBlock(nDataPairsLine string, sc *bufio.Scanner, colWidth int, valueWidth int) ([][2]float64, error) {
	pairs := [][2]float64{}
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, nDataPairsLine) {
			nPairs, err := strconv.Atoi(rightofEquals(line))
			if err != nil {
				return pairs, err
			}
			pairs, err = dataPairsfromTextBlock(sc, nPairs, colWidth, valueWidth)
			if err != nil {
				return pairs, err
			}
			break
		}
	}
	return pairs, nil
}

// distance returns the distance along a straight line in euclidean space
func distance(p0, p1 [2]float64) float64 {
	result := math.Sqrt(math.Pow((p1[0]-p0[0]), 2) + math.Pow((p1[1]-p0[1]), 2))
	return result
}

// pointAtDistance returns a new point along a straight line in euclidean space
// at a specified distance
func pointAtDistance(p0, p1 [2]float64, delta float64) [2]float64 {
	distanceRatio := delta / distance(p0, p1)
	newX := (1-distanceRatio)*p0[0] + distanceRatio*p1[0]
	newY := (1-distanceRatio)*p0[1] + distanceRatio*p1[1]
	return [2]float64{newX, newY}
}

// interpZ creates a new point a given distance along a line composed
// of many segments.
func interpXY(xyPairs [][2]float64, d float64) [2]float64 {
	// newPoint is an x, y pair
	var newPoint [2]float64
	lineSegments := len(xyPairs) - 1
	lineLength := 0.0

findLineSegment:
	for i := 0; i < lineSegments; i++ {
		p0, p1 := xyPairs[i], xyPairs[i+1]
		lineLength += distance(p0, p1)

		switch {
		case lineLength > d:
			delta := distance(p0, p1) - (lineLength - d)
			newPoint = pointAtDistance(p0, p1, delta)
			break findLineSegment

		default:
			continue
		}

	}

	return newPoint
}

// attributeZ using station from cross-section line and gis coordinates
// Minor bug here caused (I think) where interpolating
// a station beyond the length of the cutline occurs resulting
// in a point (0,0). Need to add improved error handling for this.
// For now don't add stations/elevations longer than the GIS line
func attributeZ(xyPairs [][2]float64, mzPairs [][2]float64) []xyzPoint {
	points := []xyzPoint{}
	startingStation := mzPairs[0][0]
	for _, mzPair := range mzPairs {
		newPoint := interpXY(xyPairs, mzPair[0]-startingStation)
		if newPoint[0] != 0 && newPoint[1] != 0 {
			points = append(points, xyzPoint{newPoint[0], newPoint[1], mzPair[1]})
		}
	}
	return points
}

func getTransform(sourceProjection string, destinationEPSG int) (gdal.CoordinateTransform, error) {
	transform := gdal.CoordinateTransform{}
	sourceSpRef := gdal.CreateSpatialReference(sourceProjection)
	sourceSpRef.MorphFromESRI()
	if err := sourceSpRef.Validate(); err != nil {
		return transform, errors.New("Unable to extract geospatial data. Invalid source Projection")
	}

	destinationSpRef := gdal.CreateSpatialReference("")
	if err := destinationSpRef.FromEPSG(destinationEPSG); err != nil {
		return transform, err
	}
	transform = gdal.CreateCoordinateTransform(sourceSpRef, destinationSpRef)
	return transform, nil
}

func flipXYLineString(xyLineString gdal.Geometry) gdal.Geometry {
	yxLineString := gdal.Create(gdal.GT_LineString)
	nPoints := xyLineString.PointCount()
	for i := 0; i < nPoints; i++ {
		x, y, _ := xyLineString.Point(i)
		yxLineString.AddPoint2D(y, x)
	}
	xyLineString.Destroy()
	return yxLineString
}

func flipXYLineString25D(xyzLineString gdal.Geometry) gdal.Geometry {
	yxzLineString := gdal.Create(gdal.GT_LineString25D)
	nPoints := xyzLineString.PointCount()
	for i := 0; i < nPoints; i++ {
		x, y, z := xyzLineString.Point(i)
		yxzLineString.AddPoint(y, x, z)
	}
	xyzLineString.Destroy()
	return yxzLineString
}

func flipXYLinearRing(xyLinearRing gdal.Geometry) gdal.Geometry {
	yxLinearRing := gdal.Create(gdal.GT_LinearRing)
	nPoints := xyLinearRing.PointCount()
	for i := 0; i < nPoints; i++ {
		x, y, _ := xyLinearRing.Point(i)
		yxLinearRing.AddPoint2D(y, x)
	}
	xyLinearRing.Destroy()
	return yxLinearRing
}

func flipXYPoint(xyPoint gdal.Geometry) gdal.Geometry {
	yxPoint := gdal.Create(gdal.GT_Point)
	nPoints := xyPoint.PointCount()
	for i := 0; i < nPoints; i++ {
		x, y, _ := xyPoint.Point(i)
		yxPoint.AddPoint2D(y, x)
	}
	xyPoint.Destroy()
	return yxPoint
}

func getRiverCenterline(sc *bufio.Scanner, transform gdal.CoordinateTransform) (vectorLayer, error) {
	riverReach := strings.Split(rightofEquals(sc.Text()), ",")
	layer := vectorLayer{FeatureName: fmt.Sprintf("%s, %s", strings.TrimSpace(riverReach[0]), strings.TrimSpace(riverReach[1]))}

	xyPairs, err := getDataPairsfromTextBlock("Reach XY=", sc, 64, 16)
	if err != nil {
		return layer, err
	}

	xyLineString := gdal.Create(gdal.GT_LineString)
	for _, pair := range xyPairs {
		xyLineString.AddPoint2D(pair[0], pair[1])
	}

	xyLineString.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped:
	yxLineString := flipXYLineString(xyLineString)

	multiLineString := yxLineString.ForceToMultiLineString()
	wkb, err := multiLineString.ToWKB()
	if err != nil {
		return layer, err
	}
	layer.Geometry = wkb
	return layer, err
}

func getXSBanks(sc *bufio.Scanner, transform gdal.CoordinateTransform, riverReachName string) (vectorLayer, []vectorLayer, error) {
	bankLayers := []vectorLayer{}

	xsLayer, xyPairs, startingStation, err := getXS(sc, transform, riverReachName)
	if err != nil {
		return xsLayer, bankLayers, err
	}
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "Bank Sta=") {
			bankLayers, err = getBanks(line, transform, xsLayer, xyPairs, startingStation)
			if err != nil {
				return xsLayer, bankLayers, err
			}
			break
		}
	}
	return xsLayer, bankLayers, err
}

func getXS(sc *bufio.Scanner, transform gdal.CoordinateTransform, riverReachName string) (vectorLayer, [][2]float64, float64, error) {
	compData := strings.Split(rightofEquals(sc.Text()), ",")
	layer := vectorLayer{FeatureName: strings.TrimSpace(compData[1]), Fields: map[string]interface{}{}}
	layer.Fields["RiverReachName"] = riverReachName

	xyPairs, err := getDataPairsfromTextBlock("XS GIS Cut Line", sc, 64, 16)
	if err != nil {
		return layer, xyPairs, 0.0, err
	}

	mzPairs, err := getDataPairsfromTextBlock("#Sta/Elev", sc, 80, 8)
	if err != nil {
		return layer, xyPairs, mzPairs[0][0], err
	}

	xyzPoints := attributeZ(xyPairs, mzPairs)

	xyzLineString := gdal.Create(gdal.GT_LineString25D)
	for _, point := range xyzPoints {
		xyzLineString.AddPoint(point.x, point.y, point.z)
	}

	xyzLineString.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped
	yxzLineString := flipXYLineString25D(xyzLineString)

	multiLineString := yxzLineString.ForceToMultiLineString()
	wkb, err := multiLineString.ToWKB()
	if err != nil {
		return layer, xyPairs, mzPairs[0][0], err
	}
	layer.Geometry = wkb
	return layer, xyPairs, mzPairs[0][0], err
}

func getBanks(line string, transform gdal.CoordinateTransform, xsLayer vectorLayer, xyPairs [][2]float64, startingStation float64) ([]vectorLayer, error) {
	layers := []vectorLayer{}

	bankStations := strings.Split(rightofEquals(line), ",")
	for _, s := range bankStations {
		layer := vectorLayer{FeatureName: strings.TrimSpace(s), Fields: map[string]interface{}{}}
		layer.Fields["RiverReachName"] = xsLayer.Fields["RiverReachName"]
		layer.Fields["xsName"] = xsLayer.FeatureName
		bankStation, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return layers, err
		}
		bankXY := interpXY(xyPairs, bankStation-startingStation)
		xyPoint := gdal.Create(gdal.GT_Point)
		xyPoint.AddPoint2D(bankXY[0], bankXY[1])
		xyPoint.Transform(transform)
		// This is a temporary fix since the x and y values need to be flipped
		yxPoint := flipXYPoint(xyPoint)
		multiPoint := yxPoint.ForceToMultiPoint()
		wkb, err := multiPoint.ToWKB()
		if err != nil {
			return layers, err
		}
		layer.Geometry = wkb
		layers = append(layers, layer)
	}
	return layers, nil
}

func getStorageArea(sc *bufio.Scanner, transform gdal.CoordinateTransform) (vectorLayer, error) {
	layer := vectorLayer{FeatureName: strings.TrimSpace(strings.Split(rightofEquals(sc.Text()), ",")[0])}

	xyPairs, err := getDataPairsfromTextBlock("Storage Area Surface Line=", sc, 32, 16)
	if err != nil {
		return layer, err
	}

	xyLinearRing := gdal.Create(gdal.GT_LinearRing)
	for _, pair := range xyPairs {
		xyLinearRing.AddPoint2D(pair[0], pair[1])
	}

	xyLinearRing.Transform(transform)
	// This is a temporary fix since the x and y values need to be flipped:
	yxLinearRing := flipXYLinearRing(xyLinearRing)

	yxPolygon := gdal.Create(gdal.GT_Polygon)
	yxPolygon.AddGeometry(yxLinearRing)
	yxMultiPolygon := yxPolygon.ForceToMultiPolygon()
	wkb, err := yxMultiPolygon.ToWKB()
	if err != nil {
		return layer, err
	}
	layer.Geometry = wkb
	return layer, err
}

// GetGeospatialData ...
func GetGeospatialData(gd *GeoData, fs filestore.FileStore, geomFilePath string) error {
	geomFileName := filepath.Base(geomFilePath)
	f := Features{}
	riverReachName := ""

	file, err := fs.GetObject(geomFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	sc := bufio.NewScanner(file)

	transform, err := getTransform(sourceCRS, DestinationCRS)
	if err != nil {
		return err
	}

	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "River Reach="):
			riverLayer, err := getRiverCenterline(sc, transform)
			if err != nil {
				return err
			}
			f.Rivers = append(f.Rivers, riverLayer)
			riverReachName = riverLayer.FeatureName

		case strings.HasPrefix(line, "Storage Area="):
			storageAreaLayer, err := getStorageArea(sc, transform)
			if err != nil {
				return err
			}
			f.StorageAreas = append(f.StorageAreas, storageAreaLayer)

		case strings.HasPrefix(line, "Type RM Length L Ch R = 1"):
			xsLayer, bankLayers, err := getXSBanks(sc, transform, riverReachName)
			if err != nil {
				return err
			}
			f.XS = append(f.XS, xsLayer)
			f.Banks = append(f.Banks, bankLayers...)
		}
	}

	gd.Features[geomFileName] = f
	return nil
}
