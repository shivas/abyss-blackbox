package pb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"strings"

	"github.com/lxn/walk"
)

var emptyImage *walk.Bitmap

func init() {
	emptyImage, _ = walk.NewBitmapForDPI(walk.Size{Width: 32, Height: 32}, 92)
}

func (r *FittingRecord) ShipIcon() *walk.Icon {
	img, _, err := image.Decode(bytes.NewReader(r.Icon))
	if err != nil {
		fmt.Printf("returning empty icon\n")
		return &walk.Icon{}
	}

	icon, err := walk.NewIconFromImageForDPI(img, 92)
	if err != nil {
		return &walk.Icon{}
	}

	return icon
}

func (r *FittingRecord) ShipImage() *walk.Bitmap {
	if r == nil || r.Icon == nil {
		return emptyImage
	}

	img, _, err := image.Decode(bytes.NewReader(r.Icon))
	if err != nil {
		return emptyImage
	}

	bm, err := walk.NewBitmapFromImageForDPI(img, 92)

	return bm
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
	r.Price = 0.0 // perform price check?
	r.FFH = validateResponse.FFH

	return nil
}
