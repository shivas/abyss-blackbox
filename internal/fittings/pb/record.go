package pb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"net/http"
	"strings"
)

func emptyShipImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 40, G: 40, B: 40, A: 255})
		}
	}

	return img
}

func (r *FittingRecord) ShipPicture() image.Image {
	if r == nil || r.Icon == nil {
		return emptyShipImage()
	}

	img, _, err := image.Decode(bytes.NewReader(r.Icon))
	if err != nil {
		return emptyShipImage()
	}

	return img
}

func (r *FittingRecord) Validate() error {
	response, err := http.Post("https://abyssal.space/api/fitting/validate", "text/plain", strings.NewReader(r.EFT))
	if err != nil {
		return err
	}

	defer response.Body.Close()

	var validateResponse struct {
		Valid       bool    `json:"valid"`
		Error       *string `json:"error,omitempty"`
		Ship        string  `json:"ship"`
		ShipTypeID  int     `json:"shipTypeID"`
		FittingName string  `json:"fittingName"`
		FFH         string  `json:"ffh"`
	}

	err = json.NewDecoder(response.Body).Decode(&validateResponse)
	if err != nil {
		return err
	}

	if !validateResponse.Valid && validateResponse.Error != nil {
		return fmt.Errorf("error validating fit: %s", *validateResponse.Error)
	}

	r.ShipTypeID = int32(validateResponse.ShipTypeID)
	r.ShipName = validateResponse.Ship
	r.FittingName = validateResponse.FittingName
	r.Price = 0.0
	r.FFH = validateResponse.FFH

	return nil
}
