package parser

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

type Thumbnail struct {
	width      int
	height     int
	dataLength int
	data       []byte
}

type GCode struct {
	images []Thumbnail
	data   map[string]string
}

type SuperSlicerGcode struct {
	images        []Thumbnail
	fillamentUsed float64
	fillamentType string
	printerNotes  string
}

func ParseGCode(f *bufio.Reader) (GCode, error) {
	gcode := GCode{}
	thumbnailInitalized := false
	currentThumbnail := Thumbnail{}
	gcode.data = make(map[string]string)

	for {
		line, err := f.ReadString('\n')
		if err != nil {
			break
		}

		if strings.HasPrefix(line, ";") {
			command := strings.TrimSpace(strings.TrimPrefix(line, ";"))

			if strings.HasPrefix(command, "thumbnail begin") {
				if thumbnailInitalized {
					return gcode, fmt.Errorf("Thumbnail already initialized")
				}
				thumbnailInitalized = true

				data := strings.Split(command, " ")
				size := strings.Split(strings.TrimSpace(data[2]), "x")
				currentThumbnail.width, err = strconv.Atoi(size[0])
				if err != nil {
					return gcode, err
				}
				currentThumbnail.height, err = strconv.Atoi(size[1])
				if err != nil {
					return gcode, err
				}
				currentThumbnail.dataLength, err = strconv.Atoi(data[3])
				if err != nil {
					return gcode, err
				}
			} else if strings.HasPrefix(command, "thumbnail end") {
				if !thumbnailInitalized {
					return gcode, fmt.Errorf("Thumbnail not initialized")
				}

				if len(currentThumbnail.data) != currentThumbnail.dataLength {
					return gcode, fmt.Errorf("Thumbnail data length does not match")
				}

				gcode.images = append(gcode.images, currentThumbnail)
				currentThumbnail = Thumbnail{}
				thumbnailInitalized = false
			} else {
				if thumbnailInitalized {
					currentThumbnail.data = append(currentThumbnail.data, []byte(command)...)
				} else {
					command := strings.Split(command, " = ")
					if len(command) != 2 {
						continue
					}
					gcode.data[command[0]] = command[1]
				}
			}
		}
	}

	return gcode, nil
}

func (gcode GCode) AsSuperSlicerGcode() (SuperSlicerGcode, error) {
	superSlicerGcode := SuperSlicerGcode{}
	superSlicerGcode.images = gcode.images

	var err error
	var ok bool

	superSlicerGcode.fillamentUsed, err = strconv.ParseFloat(gcode.data["total filament used [g]"], 64)
	if err != nil {
		return superSlicerGcode, err
	}

	superSlicerGcode.fillamentType, ok = gcode.data["filament_type"]
	if !ok {
		return superSlicerGcode, fmt.Errorf("fillament type not found, most likely not a SuperSlicer Gcode")
	}

	superSlicerGcode.printerNotes, ok = gcode.data["printer_notes"]
	if !ok {
		return superSlicerGcode, fmt.Errorf("printer notes not found, most likely not a SuperSlicer Gcode")
	}

	return superSlicerGcode, nil
}
