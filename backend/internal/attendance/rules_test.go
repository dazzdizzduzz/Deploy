package attendance

import (
	"errors"
	"testing"
	"time"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := LoadTimezone(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// at membuat waktu lokal pada tanggal 11 November 2026 (atau tanggal lain lewat day).
func at(loc *time.Location, day, hour, minute int) time.Time {
	return time.Date(2026, time.November, day, hour, minute, 0, 0, loc)
}

var officeHours = Schedule{
	StartTime:            "08:00",
	EndTime:              "17:00",
	ToleranceMinutes:     15,
	OvertimeAfterMinutes: 60,
	MaxOvertimeMinutes:   180,
}

func TestLoadTimezone(t *testing.T) {
	if loc := mustLoc(t, ""); loc.String() != DefaultTimezone {
		t.Fatalf("default = %s, want %s", loc, DefaultTimezone)
	}
	if _, err := LoadTimezone("Asia/TidakAda"); err == nil {
		t.Fatal("zona waktu tidak dikenal seharusnya error")
	}
}

func TestWorkDate_UsesTenantTimezone(t *testing.T) {
	wita := mustLoc(t, "Asia/Makassar")
	// 23:30 UTC tanggal 10 = 07:30 WITA tanggal 11.
	got, err := WorkDate(time.Date(2026, 11, 10, 23, 30, 0, 0, time.UTC), wita)
	if err != nil {
		t.Fatal(err)
	}
	if got.Day() != 11 || got.Hour() != 0 {
		t.Fatalf("WorkDate = %v, want 2026-11-11 00:00 WITA", got)
	}
	if _, err := WorkDate(time.Now(), nil); !errors.Is(err, ErrNilLocation) {
		t.Fatalf("err = %v, want ErrNilLocation", err)
	}
}

func TestEvaluateClockIn(t *testing.T) {
	wita := mustLoc(t, "Asia/Makassar")
	tests := []struct {
		name     string
		clockIn  time.Time
		status   Status
		lateMins int
	}{
		{"datang lebih awal", at(wita, 11, 7, 45), StatusOnTime, 0},
		{"tepat jam masuk", at(wita, 11, 8, 0), StatusOnTime, 0},
		{"masih dalam toleransi", at(wita, 11, 8, 15), StatusOnTime, 0},
		{"lewat toleransi satu menit", at(wita, 11, 8, 16), StatusLate, 16},
		{"terlambat satu jam", at(wita, 11, 9, 0), StatusLate, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := EvaluateClockIn(tt.clockIn, officeHours, wita)
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != tt.status || res.LateMinutes != tt.lateMins {
				t.Fatalf("got %s/%d, want %s/%d", res.Status, res.LateMinutes, tt.status, tt.lateMins)
			}
			if res.WorkDate.Day() != 11 {
				t.Fatalf("WorkDate = %v, want tanggal 11", res.WorkDate)
			}
		})
	}
}

func TestEvaluateClockIn_TimezoneMatters(t *testing.T) {
	// 00:30 UTC = 08:30 WITA (terlambat) tetapi 07:30 WIB (tepat waktu).
	utc := time.Date(2026, 11, 11, 0, 30, 0, 0, time.UTC)

	wita, err := EvaluateClockIn(utc, officeHours, mustLoc(t, "Asia/Makassar"))
	if err != nil {
		t.Fatal(err)
	}
	wib, err := EvaluateClockIn(utc, officeHours, mustLoc(t, "Asia/Jakarta"))
	if err != nil {
		t.Fatal(err)
	}
	if wita.Status != StatusLate || wib.Status != StatusOnTime {
		t.Fatalf("WITA=%s WIB=%s, want late dan on_time", wita.Status, wib.Status)
	}
}

func TestEvaluateClockIn_AcceptsPostgresTimeFormat(t *testing.T) {
	sch := officeHours
	sch.StartTime = "08:00:00" // kolom TIME PostgreSQL dibaca sebagai HH:MM:SS
	sch.EndTime = "17:00:00"
	res, err := EvaluateClockIn(at(time.UTC, 11, 8, 20), sch, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusLate {
		t.Fatalf("status = %s, want late", res.Status)
	}
}

func TestEvaluateClockIn_InvalidSchedule(t *testing.T) {
	sch := officeHours
	sch.StartTime = "8 pagi"
	if _, err := EvaluateClockIn(at(time.UTC, 11, 8, 0), sch, time.UTC); !errors.Is(err, ErrInvalidClock) {
		t.Fatalf("err = %v, want ErrInvalidClock", err)
	}
}

func TestEvaluateClockOut(t *testing.T) {
	wita := mustLoc(t, "Asia/Makassar")
	clockIn := at(wita, 11, 8, 0)
	tests := []struct {
		name     string
		clockOut time.Time
		early    int
		overtime int
	}{
		{"pulang cepat 30 menit", at(wita, 11, 16, 30), 30, 0},
		{"tepat jam pulang", at(wita, 11, 17, 0), 0, 0},
		{"lewat 59 menit belum lembur", at(wita, 11, 17, 59), 0, 0},
		{"lewat 60 menit jadi lembur", at(wita, 11, 18, 0), 0, 60},
		{"lembur dibatasi maksimum", at(wita, 11, 22, 0), 0, 180},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := EvaluateClockOut(clockIn, tt.clockOut, officeHours, wita)
			if err != nil {
				t.Fatal(err)
			}
			if res.EarlyLeaveMinutes != tt.early || res.OvertimeMinutes != tt.overtime {
				t.Fatalf("got early=%d overtime=%d, want %d/%d",
					res.EarlyLeaveMinutes, res.OvertimeMinutes, tt.early, tt.overtime)
			}
		})
	}
}

func TestEvaluateClockOut_NoOvertimeCap(t *testing.T) {
	sch := officeHours
	sch.MaxOvertimeMinutes = 0 // tanpa batas
	res, err := EvaluateClockOut(at(time.UTC, 11, 8, 0), at(time.UTC, 11, 22, 0), sch, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if res.OvertimeMinutes != 300 {
		t.Fatalf("overtime = %d, want 300", res.OvertimeMinutes)
	}
}

func TestEvaluateClockOut_NightShift(t *testing.T) {
	night := Schedule{StartTime: "22:00", EndTime: "06:00", ToleranceMinutes: 10, OvertimeAfterMinutes: 30}
	clockIn := at(time.UTC, 11, 21, 55)

	// Pulang 05:00 keesokan harinya = pulang cepat 60 menit, bukan lembur.
	res, err := EvaluateClockOut(clockIn, at(time.UTC, 12, 5, 0), night, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if res.EarlyLeaveMinutes != 60 || res.OvertimeMinutes != 0 {
		t.Fatalf("got %+v, want early=60", res)
	}

	// Pulang 07:00 keesokan harinya = lembur 60 menit.
	res, err = EvaluateClockOut(clockIn, at(time.UTC, 12, 7, 0), night, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if res.OvertimeMinutes != 60 {
		t.Fatalf("got %+v, want overtime=60", res)
	}
}

func TestEvaluateClockOut_BeforeClockIn(t *testing.T) {
	_, err := EvaluateClockOut(at(time.UTC, 11, 8, 0), at(time.UTC, 11, 7, 0), officeHours, time.UTC)
	if !errors.Is(err, ErrClockOutBeforeClockIn) {
		t.Fatalf("err = %v, want ErrClockOutBeforeClockIn", err)
	}
}
