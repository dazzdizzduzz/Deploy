// Package attendance berisi aturan bisnis absensi yang bersifat deterministik (rule-based, SAD bab 1.2):
// perhitungan geofence GPS (FR-06), status kehadiran saat clock-in (FR-08), dan
// pulang cepat / lembur saat clock-out (FR-07). Paket ini tidak menyentuh HTTP maupun database
// agar mudah diuji unit.
package attendance

import (
	"errors"
	"math"
)

// earthRadiusMeters adalah jari-jari rata-rata bumi (IUGG) yang umum dipakai rumus Haversine.
const earthRadiusMeters = 6371000.0

// ErrInvalidCoordinate dikembalikan bila latitude/longitude di luar rentang geografis.
var ErrInvalidCoordinate = errors.New("koordinat di luar rentang yang valid")

// Point adalah satu titik koordinat GPS dalam derajat desimal.
type Point struct {
	Lat float64
	Lon float64
}

// Validate memastikan koordinat berada di rentang -90..90 (lat) dan -180..180 (lon)
// serta bukan NaN/Inf (bisa terjadi bila klien mengirim data rusak).
func (p Point) Validate() error {
	if math.IsNaN(p.Lat) || math.IsNaN(p.Lon) || math.IsInf(p.Lat, 0) || math.IsInf(p.Lon, 0) {
		return ErrInvalidCoordinate
	}
	if p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 {
		return ErrInvalidCoordinate
	}
	return nil
}

// DistanceMeters menghitung jarak lingkaran besar (great-circle) antara dua titik
// memakai rumus Haversine. Akurasi cukup untuk radius geofence 25-500 meter (FR-05).
func DistanceMeters(a, b Point) float64 {
	lat1 := toRadians(a.Lat)
	lat2 := toRadians(b.Lat)
	dLat := toRadians(b.Lat - a.Lat)
	dLon := toRadians(b.Lon - a.Lon)

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)

	// Batasi h ke [0,1] agar galat pembulatan float tidak membuat Sqrt/Asin bernilai NaN.
	h = math.Min(1, math.Max(0, h))

	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

// GeofenceResult adalah hasil pemeriksaan geofence.
type GeofenceResult struct {
	Inside         bool
	DistanceMeters int // dibulatkan ke meter terdekat, disimpan ke kolom distance_*_meters
}

// CheckGeofence memeriksa apakah posisi karyawan berada di dalam radius kantor.
// Titik tepat di batas radius dianggap masih di dalam.
func CheckGeofence(employee, office Point, radiusMeters int) (GeofenceResult, error) {
	if err := employee.Validate(); err != nil {
		return GeofenceResult{}, err
	}
	if err := office.Validate(); err != nil {
		return GeofenceResult{}, err
	}
	if radiusMeters <= 0 {
		return GeofenceResult{}, errors.New("radius geofence harus lebih dari 0 meter")
	}

	d := DistanceMeters(employee, office)
	return GeofenceResult{
		Inside:         d <= float64(radiusMeters),
		DistanceMeters: int(math.Round(d)),
	}, nil
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
