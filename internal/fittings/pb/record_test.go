package pb

import (
	"testing"
)

func TestFittingRecord_Validate(t *testing.T) {
	tests := []struct {
		name    string
		r       *FittingRecord
		wantErr bool
	}{
		{"firsttest", &FittingRecord{Source: "test", EFT: `[Nergal, FFA]
Corpii A-Type Small Armor Repairer
Corpum C-Type Multispectrum Energized Membrane
Assault Damage Control II
Corpii A-Type Small Armor Repairer

Small Capacitor Booster II
Coreli A-Type 1MN Afterburner
Republic Fleet Small Cap Battery

Veles Light Entropic Disintegrator
Coreli A-Type Small Remote Armor Repairer

Small Thermal Armor Reinforcer II
Small Capacitor Control Circuit II




Occult S x1789
Nanite Repair Paste x29
Meson Exotic Plasma S x1500
Navy Cap Booster 400 x10
Mystic S x1697
Agency 'Hardshell' TB5 Dose II x1
Agency 'Pyrolancea' DB3 Dose I x1

`}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("FittingRecord.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.r.ShipName != "Nergal" {
				t.Errorf("FittingRecord.Validate() ship name incorrect, received: %q", tt.r.ShipName)
			}
		})
	}
}
