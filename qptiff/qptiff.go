package qptiff

import (
	"encoding/xml"
	"strings"

	"gitea.alanrace.com/alan/go-tiff"
)

type ImageType int

const (
	FullResolution ImageType = iota
	Thumbnail
	ReducedResolution
	Overview
	Label
)

func (imageType ImageType) String() string {
	return [...]string{"FullResolution", "Thumbnail", "ReducedResolution", "Overview", "Label"}[imageType]
}

var imageTypeMap = map[string]ImageType{
	"FullResolution":    FullResolution,
	"Thumbnail":         Thumbnail,
	"ReducedResolution": ReducedResolution,
	"Overview":          Overview,
	"Label":             Label,
}

type QptiffFile struct {
	tiff.TiffFile

	FilterList []string
	FilterMap  map[string]*Filter

	Thumbnail *tiff.ImageFileDirectory
	Overview *tiff.ImageFileDirectory
	Label *tiff.ImageFileDirectory
}

type QptiffImageFileDirectory struct {
	tiff.ImageFileDirectory
}

type ImageDescription struct {
	DescriptionVersion  string `xml:"DescriptionVersion"`
	AcquisitionSoftware string
	ImageType           string
	Identifier          string
	SlideID             string
	Barcode             string
	ComputerName        string
	IsUnmixedComponent  bool
	ExposureTime        int64
	SignalUnits         int64
	Objective           string
	CameraName          string
	ValidationCode      string
}

type Filter struct {
	Name         string  `xml:"Name"`
	ColourString string  `xml:"Color"`
	Response     float64 `xml:"Responsivity>Filter>Response"`
	Date         string  `xml:"Responsivity>Filter>Date"`
	FilterID     string  `xml:"Responsivity>Filter>FilterID"`

	IFDList []*tiff.ImageFileDirectory
}

// TODO: Collect all of the data out of the <ScanProfile> - this only appears in the first ImageFileDirectory.ImageDescription
type ScanProfile struct {
}

type FormatError struct {
	msg string // description of error
	//Offset int64  // error occurred after reading Offset bytes
}

func (e *FormatError) Error() string { return e.msg }

func Open(path string) (*QptiffFile, error) {
	var qptiffFile QptiffFile
	tiffFile, err := tiff.Open(path)

	qptiffFile.TiffFile = *tiffFile
	qptiffFile.FilterMap = make(map[string]*Filter)

	for _, ifd := range tiffFile.IFDList {
		//firstImage := tiffFile.IFDList[0]
		// Process image details from first image to determine which type of data, either brightfield or fluorescence

		imageDetailsTag, ok := ifd.Tags[tiff.ImageDescription].(*tiff.ASCIITiffTag)

		if !ok {
			return nil, &FormatError{msg: "ImageDescription is not ASCII type"}
		}

		imageDetailsString := imageDetailsTag.GetValueAsString()
		// Remove first line, as the string is not actually encoded as UTF-16
		imageDetailsString = strings.Replace(imageDetailsString, "<?xml version=\"1.0\" encoding=\"utf-16\"?>", "", 1)
		var imageDetails ImageDescription

		xml.Unmarshal([]byte(imageDetailsString), &imageDetails)

		imageType := imageTypeMap[imageDetails.ImageType]

		switch imageType {
		case FullResolution:
			var filter Filter
			xml.Unmarshal([]byte(imageDetailsString), &filter)

			qptiffFile.FilterList = append(qptiffFile.FilterList, filter.Name)
			qptiffFile.FilterMap[filter.Name] = &filter

			filter.IFDList = append(filter.IFDList, ifd)
		case Thumbnail:
			qptiffFile.Thumbnail = ifd
		case ReducedResolution:
			var filter Filter
			xml.Unmarshal([]byte(imageDetailsString), &filter)

			fullFilter := qptiffFile.FilterMap[filter.Name]
			fullFilter.IFDList = append(fullFilter.IFDList, ifd)
		case Overview:
			qptiffFile.Overview = ifd
		case Label:
			qptiffFile.Label = ifd
		default:
			return nil, &FormatError{msg: "Unknown ImageType " + imageDetails.ImageType}
		}

		/*fo, err := os.Create(strconv.Itoa(index) + ".xml")
		if err != nil {
			panic(err)
		}

		w := bufio.NewWriter(fo)

		if _, err := w.Write([]byte(imageDetailsString)); err != nil {
			panic(err)
		}

		w.Flush()
		fo.Close()*/
	}

	return &qptiffFile, err
}

/** https://github.com/openmicroscopy/bioformats/blob/develop/components/formats-gpl/src/loci/formats/in/VectraReader.java
 *
 * Returns the index of the IFD to be used for the given
 * core index and image number.
 *
 * The IFD order in general is:
 *
 *  - IFD #0 to n-1: full resolution images (1 RGB for BF data, n grayscale for FL)
 *  - IFD #n: RGB thumbnail
 *  - IFD #n+1 to (n*2)-1: 50% resolution images (optional)
 *  - IFD #n*2 to (n*3)-1: 25% resolution images (optional)
 *  ...
 *  - macro/overview image (optional)
 *  - label image (optional)
 */
