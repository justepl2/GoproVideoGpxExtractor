package parser

import (
	"encoding/xml"
	"io/ioutil"
	"bytes"
	"log"
	"fmt"
)

// Stucts Unmashal OpenRouteSevice GPX
type Extensions struct {
	Duration float64 		`xml:"duration"`
	Distance float64 		`xml:"distance"`
	Type int 				`xml:"type"`
	Step int 				`xml:"step"`
}

type Rtept struct {
	Lat string 				`xml:"lat,attr"`
	Lon string 				`xml:"lon,attr"`
	Extensions []Extensions `xml:"extensions"`
}

type Rte struct {
	Rtept []Rtept 			`xml:"rtept"`
}

type OpenRouteServiceGpx struct {
	XMLName xml.Name 		`xml:"gpx"`
	Rte Rte 				`xml:"rte"`
}


// Structs for Marshall OpenRROuteServicee GPX into a TRK GPX can be read for make videos
type Trkpt struct {
	Lat string 				`xml:"lat,attr"`
	Lon string 				`xml:"lon,attr"`
}

type Trkseg struct {
	Trkpt []Trkpt 			`xml:"trkpt"`
}

type Trk struct {
	Name string 			`xml:"name"`
	Trkseg Trkseg 			`xml:"trkseg"`
}

type TrkGpxStruct struct {
	XMLName xml.Name 		`xml:"gpx"`
	Trk Trk					`xml:"trk"`
}

func ParseOpenRouteServiceGPX(gpxData []byte, i int, destDir string, files []string)  {
	openRouteServiceGpxStruct := OpenRouteServiceGpx{}

	content := xml.NewDecoder(bytes.NewBuffer(gpxData)) 
	fmt.Println("link : " + files[0] + " -> " + files[1])
	err := content.Decode(&openRouteServiceGpxStruct)
	if err != nil {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Fatal(err)
	}
	
	convertOpenRouteServiceGpxIntoTrkGpx(openRouteServiceGpxStruct, i, destDir, files)
}

func convertOpenRouteServiceGpxIntoTrkGpx(openRouteServiceGpxStruct OpenRouteServiceGpx, i int, destDir string, files []string) {
	trkGpx := TrkGpxStruct{}

	trkGpx.Trk.Name = "openRouteService"


	for _, rtept := range openRouteServiceGpxStruct.Rte.Rtept {
		trkpt := Trkpt{Lat: rtept.Lat, Lon: rtept.Lon}
		trkGpx.Trk.Trkseg.Trkpt = append(trkGpx.Trk.Trkseg.Trkpt, trkpt)
	}
	writeGpxFileWithTrkInformation(trkGpx, i, destDir, files)
}

func writeGpxFileWithTrkInformation(trkGpxStruct TrkGpxStruct, i int, destDir string, files []string) {

	file, _ := xml.MarshalIndent(trkGpxStruct, "", " ")
 
	_ = ioutil.WriteFile(destDir + "/gpxFiles/" + files[0] + "To" + files[1] + ".gpx", file, 0644)

}