package attendance

import (
	"errors"
	"math"
	"testing"
)

// Titik pusat contoh: kantor Studio Tigapagi di Sanur (perkiraan).
var sanurOffice = Point{Lat: -8.6850, Lon: 115.2620}

func TestDistanceMeters_KnownValues(t *testing.T) {
	tests := []struct {
		name      string
		a, b      Point
		want      float64
		tolerance float64
	}{
		{"titik sama", sanurOffice, sanurOffice, 0, 0.001},
		// 1 derajat lintang = R * pi / 180 = 111194.93 m
		{"satu derajat lintang", Point{0, 0}, Point{1, 0}, 111194.93, 0.5},
		// 0.0009 derajat lintang = sekitar 100 m
		{"sekitar 100 meter", sanurOffice, Point{sanurOffice.Lat + 0.0009, sanurOffice.Lon}, 100.08, 0.5},
		// Melintasi garis bujur 180: jarak harus pendek, bukan keliling bumi.
		{"melintasi antimeridian", Point{0, 179.9995}, Point{0, -179.9995}, 111.19, 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DistanceMeters(tt.a, tt.b)
			if math.Abs(got-tt.want) > tt.tolerance {
				t.Fatalf("DistanceMeters = %.3f, want %.3f (+/- %.3f)", got, tt.want, tt.tolerance)
			}
			// Jarak harus simetris.
			if back := DistanceMeters(tt.b, tt.a); math.Abs(back-got) > 1e-9 {
				t.Fatalf("tidak simetris: %.6f vs %.6f", got, back)
			}
		})
	}
}

func TestCheckGeofence(t *testing.T) {
	near := Point{sanurOffice.Lat + 0.0009, sanurOffice.Lon} // sekitar 100 m
	far := Point{sanurOffice.Lat + 0.0045, sanurOffice.Lon}  // sekitar 500 m

	t.Run("di dalam radius", func(t *testing.T) {
		res, err := CheckGeofence(near, sanurOffice, 150)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Inside || res.DistanceMeters != 100 {
			t.Fatalf("got %+v, want inside dengan jarak 100", res)
		}
	})

	t.Run("di luar radius", func(t *testing.T) {
		res, err := CheckGeofence(far, sanurOffice, 150)
		if err != nil {
			t.Fatal(err)
		}
		if res.Inside {
			t.Fatalf("got inside, want outside (jarak %d m)", res.DistanceMeters)
		}
	})

	t.Run("tepat di batas dianggap di dalam", func(t *testing.T) {
		d := DistanceMeters(near, sanurOffice)
		res, err := CheckGeofence(near, sanurOffice, int(math.Ceil(d)))
		if err != nil {
			t.Fatal(err)
		}
		if !res.Inside {
			t.Fatal("titik di batas radius seharusnya dianggap di dalam")
		}
	})

	t.Run("radius tidak valid", func(t *testing.T) {
		if _, err := CheckGeofence(near, sanurOffice, 0); err == nil {
			t.Fatal("radius 0 seharusnya error")
		}
	})
}

func TestCheckGeofence_InvalidCoordinates(t *testing.T) {
	bad := []Point{
		{Lat: 91, Lon: 0},
		{Lat: -91, Lon: 0},
		{Lat: 0, Lon: 181},
		{Lat: math.NaN(), Lon: 0},
		{Lat: 0, Lon: math.Inf(1)},
	}
	for _, p := range bad {
		if _, err := CheckGeofence(p, sanurOffice, 100); !errors.Is(err, ErrInvalidCoordinate) {
			t.Errorf("CheckGeofence(%v) err = %v, want ErrInvalidCoordinate", p, err)
		}
		if _, err := CheckGeofence(sanurOffice, p, 100); !errors.Is(err, ErrInvalidCoordinate) {
			t.Errorf("CheckGeofence(office=%v) err = %v, want ErrInvalidCoordinate", p, err)
		}
	}
}
