/*
This script aims to create a GPX route from multiple Gopro video files. 

You will need to make sure your Gopro has location enabled for this script to work. If this feature is not enabled we will not be able to create a route.

script description : 
	- define a folder where your Gopro videos are saved (variable: sourceDir)
	- define a folder in which will be saved the different telemetry data in .csv format (variable : telemetryDestDir (by default : sourceDir + "/telemetry/")
	- define a folder in which will be saved the different gpx files in .gpx format (variable : gpxFileDestDir (by default : sourceDir + "/gpxFile/")
	- browse the file and for each Gopro video found we extract the data in a telemetry folder and the gpx files in the gpxFiles folder
	- then we go through the gpxFiles folder and for each gpx file we will get the first and the last coordinate. 
	- once the coordinates are retrieved we use the API of openRouteService to create a route between the last coordinates of a file and the first coordinates of the next file.
	- we store the GPX files as iToi+1.gpx
	- once all the files are created we merge all the Gpx files to have the route taken via the videos of the folder we have entered.
*/
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"os"
	"strings"
	"errors"
	"sort"
	"strconv"
	"encoding/xml"
	"net/http"
	"bytes"
	Parser "GoproVideoGpxExtractor/parser"
)

type Trkpt struct {
	Lat string `xml:"lat,attr"`
	Lon string `xml:"lon,attr"`
	Ele float64 `xml:"ele"`
}

type Trkseg struct {
	Trkpt []Trkpt `xml:"trkpt"`
}

type Trk struct {
	Name string `xml:"name"`
	Trkseg []Trkseg `xml:"trkseg"`
}

type Result struct {
	XMLName xml.Name `xml:"gpx"`
	Trk Trk `xml:"trk"`
}

func main() {

	fmt.Println("begin")
	// sourceDir := "/Volumes/VERBATIM_HD/algo_patrick/bat/"
	sourceDir := "/volumes/MEKANULL/Maroc/J4Imilchil-Ouarzazate/Bathou"
	destBinDir := "./rawVideos"
	telemetryDestDir := sourceDir + "/telemetry/"
	gpxFileDestDir := sourceDir + "/gpxFiles/"


	files, err := ioutil.ReadDir(sourceDir)
	if err != nil {
        log.Fatal(err)
    }

	sort.Slice(files, func (i,j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})

	
	_, err = os.Stat(gpxFileDestDir)
    if !os.IsNotExist(err) {
		e := os.RemoveAll(gpxFileDestDir)
		if e != nil {
			log.Fatal(e)
		}
	}

    for i, f := range files {
		if !strings.HasPrefix(f.Name(), ".") {

			if (strings.HasSuffix(f.Name(), ".mp4") || strings.HasSuffix(f.Name(), ".MP4")) {
				fmt.Println("exctract data from : " + f.Name())

				// define Var
				sourceFileName := sourceDir + "/" + f.Name()
				destFileName := destBinDir + "/" + strings.TrimSuffix(f.Name(), ".MP4")
				gpmd2csvPath := "/Users/patricklamatiere/go/src/github.com/JuanIrache/gopro-utils/bin/gpmd2csv/gpmd2csv"
				gopro2gpxPath := "/Users/patricklamatiere/go/src/github.com/JuanIrache/gopro-utils/bin/gopro2gpx/gopro2gpx"
				
				createFolder(destBinDir)
				
				// change .mp4 file into .bin file (raw video)
				transformMp4FileToCsvFile(sourceFileName, destFileName)
				
				// convert bin to csv with all datas
				parseCsvForExtractDatas(gpmd2csvPath, destFileName + ".bin", telemetryDestDir, strings.TrimSuffix(f.Name(), ".MP4") + ".csv")
				
				// create a Gpx file with gopro datas
				parseCsvForCreateGpxFile(gopro2gpxPath, destFileName + ".bin", gpxFileDestDir, strconv.Itoa(i) + ".gpx")
				
				// delete temp files
				deleteTempFiles(destFileName)
				
			}
		}
	}
	
	// get for each gpx files the first and the last coord and store it in an bidimensionnal array
	createFolder(gpxFileDestDir)

	gpxFiles, err := ioutil.ReadDir(gpxFileDestDir)
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(gpxFiles, func (i,j int) bool {
		gpxFileNameA, err := strconv.Atoi(strings.TrimSuffix(gpxFiles[i].Name(), ".gpx"))
		checkErr(err)
		gpxFileNameB, err := strconv.Atoi(strings.TrimSuffix(gpxFiles[j].Name(), ".gpx"))
		checkErr(err)
		return gpxFileNameA < gpxFileNameB
		// return gpxFiles[i].ModTime().Before(gpxFiles[j].ModTime())
	})
		
	fmt.Println("create link between Gopro's GPX =============================================================================================")
	
	gpxBufferLastCoord := Trkpt{Lat: "default", Lon: "default", Ele: 0}
	gpxBufferFirstCoord := gpxBufferLastCoord
	var fileBuffer string
	gpxStruct := Result{}
	var gpxFilesAlreadyUse []string

	for i, gpxFile := range gpxFiles {
	/* browse GPX files to get the last GPS coordinate of the first file 
	* and the first coordinate of the second file. 
	* once the data is retrieved we call the OpenRouteService API 
	* to calculate a route between these two points
	*/

	if i != 0 && (
		gpxBufferLastCoord != gpxBufferFirstCoord ||
		gpxBufferLastCoord.Lat != "default" ||
		gpxBufferLastCoord.Lon != "default" ){

		gpxFileContent, err := ioutil.ReadFile(gpxFileDestDir +  gpxFile.Name())
		
		checkErr(err)
		
		err = xml.Unmarshal([]byte(gpxFileContent), &gpxStruct)
		checkErr(err)

		if 
			len(gpxStruct.Trk.Trkseg)>0 && 
			len(gpxStruct.Trk.Trkseg[0].Trkpt)>0 && 
			!strings.Contains(gpxFile.Name(), "To") && 
			!contains(gpxFilesAlreadyUse, gpxFile.Name()) && 
			!strings.Contains(gpxFile.Name(), "final") {

			firstCoordFromSecondGpxFile := gpxStruct.Trk.Trkseg[0].Trkpt[0]
			// call OpenRouteService API for createt a direction path
			callOpenRouteServiceApi(gpxBufferLastCoord, firstCoordFromSecondGpxFile, i, sourceDir, []string{strings.TrimSuffix(fileBuffer, ".gpx"), strings.TrimSuffix(gpxFile.Name(), ".gpx")})
			
			gpxBufferLastCoord = gpxStruct.Trk.Trkseg[0].Trkpt[len(gpxStruct.Trk.Trkseg[0].Trkpt)-1]
			fileBuffer = gpxFile.Name()
			gpxFilesAlreadyUse = append(gpxFilesAlreadyUse, gpxFile.Name())
		}
		gpxStruct = Result{}
	} else {
		gpxStruct := Result{}
		
		gpxFileContent, err := ioutil.ReadFile(gpxFileDestDir +  gpxFile.Name())
		
		if err != nil {
			log.SetFlags(log.LstdFlags | log.Lshortfile)
			log.Fatal(err)
		}
		
		err = xml.Unmarshal([]byte(gpxFileContent), &gpxStruct)
		
		if err != nil {
			log.SetFlags(log.LstdFlags | log.Lshortfile)
			log.Fatal(err)
		}
		
		if 
			len(gpxStruct.Trk.Trkseg)>0 && 
			len(gpxStruct.Trk.Trkseg[0].Trkpt)>0 {

				if gpxStruct.Trk.Trkseg[0].Trkpt[len(gpxStruct.Trk.Trkseg[0].Trkpt)-1].Lat != "0" ||
				gpxStruct.Trk.Trkseg[0].Trkpt[len(gpxStruct.Trk.Trkseg[0].Trkpt)-1].Lon != "0" {

					gpxBufferLastCoord = gpxStruct.Trk.Trkseg[0].Trkpt[len(gpxStruct.Trk.Trkseg[0].Trkpt)-1]
					fileBuffer = gpxFile.Name()
					gpxStruct = Result{}
				}
		} else {
			fileBuffer = gpxFile.Name()
		}
	}
	}
		fmt.Println("Merge GPX and GPXLink ===================================================================================================")
	defer gpxMerge(gpxFileDestDir)

}

func checkErr(err error) {
	if err != nil {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Fatal(err)
   }
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func createFolder(dir string) {
	// create folder for temp files (if not exist already)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		os.Mkdir(dir, 0755)
	}
}

func transformMp4FileToCsvFile(sourceFileName string, destFileName string) {
	cmd := exec.Command("ffmpeg", "-y", "-i", sourceFileName,  "-codec", "copy", "-map", "0:3", "-f", "rawvideo", destFileName + ".bin")

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func parseCsvForExtractDatas(gpmd2csvPath string, sourceFile string, destDir string, destFilename string) {
	createFolder(destDir)
	convertBinToCsv := exec.Command(gpmd2csvPath, "-i", sourceFile , "-o", destDir + destFilename)
	err := convertBinToCsv.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func parseCsvForCreateGpxFile(gopro2gpxPath string, sourceFile string, destDir string, destFilename string) {
	createFolder(destDir)

	//command : "gopro2gpx -i GOPR0001.bin -a 500 -f 2 -o GOPR0001.gpx"
	convertBinToGpx := exec.Command(gopro2gpxPath, "-i", sourceFile, "-a", "500", "-o", destDir + destFilename)
	err := convertBinToGpx.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func deleteTempFiles(destFileName string) {
	e := os.Remove(destFileName + ".bin")
	
	if e != nil {
		log.Fatal(e)
	}
}

func callOpenRouteServiceApi(lastGpxCoordFromFirstFile Trkpt, firstGpxCoordFromSecondFile Trkpt, i int, destDir string, files []string) {

	bodyReq := []byte("{\"coordinates\":[[" + lastGpxCoordFromFirstFile.Lon + "," + lastGpxCoordFromFirstFile.Lat + "],[" + firstGpxCoordFromSecondFile.Lon + "," + firstGpxCoordFromSecondFile.Lat + "]]}")
	
	req, err := http.NewRequest("POST", "https://api.openrouteservice.org/v2/directions/cycling-mountain/gpx", bytes.NewBuffer(bodyReq))
	checkErr(err)

	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", "5b3ce3597851110001cf6248e55ad4cc6e7949e28b99f4dd7b5831dc")

	client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
	
	if (resp.StatusCode == 200) {
		body, _ := ioutil.ReadAll(resp.Body)
		Parser.ParseOpenRouteServiceGPX(body, i, destDir, files)
	}
}

func gpxMerge(sourceDir string) {
	gpxFiles, err := ioutil.ReadDir(sourceDir)
	
	if err != nil {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Fatal(err)
	}

	sort.Slice(gpxFiles, func (i,j int) bool {
			if (gpxFiles[i].Name() == "final.gpx" || gpxFiles[j].Name() == "final.gpx") {
				e := os.Remove(sourceDir + "final.gpx")
				if e != nil {
					log.Fatal(e)
				}
			}

			numA, err1 := strconv.Atoi(strings.TrimSuffix(gpxFiles[i].Name(), ".gpx"))
			numB, err2 := strconv.Atoi(strings.TrimSuffix(gpxFiles[j].Name(), ".gpx"))
			if err1 != nil {
				numA = checkIfErorIsNil(err1, strings.TrimSuffix(gpxFiles[i].Name(), ".gpx"), 0)
			}

			if err2 != nil {
				numB = checkIfErorIsNil(err2, strings.TrimSuffix(gpxFiles[j].Name(), ".gpx"), 0)
			}
			if (numA == numB && err1 != nil) {
				// A smaller than B
				return false
			}
			if (numA == numB && err2 != nil) {
				// B losmaller than A
				return true
			}
			return numA < numB
	})

	_, err = os.Create(sourceDir + "final.gpx")
	if err != nil {
		log.Fatal(err)
	}

	gpxmerge := "/Users/patricklamatiere/go/src/github.com/fg1/gpxmerge"
	command := gpxmerge + "/gpxmerge -o " + sourceDir + "final.gpx "
	for _, gpxFile := range gpxFiles {
		command += sourceDir + gpxFile.Name() + " "
		fmt.Println(gpxFile.Name())
	}

	cmd := strings.Fields(command)
	
	gpxMerger := exec.Command(cmd[0], cmd[1:]...)
	err = gpxMerger.Run()


	if err != nil {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Fatal(err)
	}
}

func checkIfErorIsNil(err error, fileName string, recurtionFactor int) int {
	num := 0
	if err != nil {
		if (len(fileName) >= recurtionFactor) {
			fileName := fileName[:len(fileName) - recurtionFactor]
			num, err = strconv.Atoi(fileName)
			if err != nil {
				num = checkIfErorIsNil(err, fileName, 1)
			}
		}
	}
	return num
}