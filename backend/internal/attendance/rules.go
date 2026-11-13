package attendance

import (
	"errors"
	"fmt"
	"time"

	// Menanam database zona waktu IANA ke dalam biner. Image alpine dan Windows tidak selalu
	// punya tzdata, sehingga tanpa ini time.LoadLocation("Asia/Makassar") bisa gagal di produksi.
	_ "time/tzdata"
)

// Status kehadiran, nilainya sama dengan CHECK constraint kolom attendances.status.
type Status string

const (
	StatusOnTime Status = "on_time"
	StatusLate   Status = "late"
	StatusAbsent Status = "absent"
)

// DefaultTimezone dipakai bila tenant belum mengatur zona waktu (kontrak API: default Asia/Jakarta).
const DefaultTimezone = "Asia/Jakarta"

var (
	ErrInvalidClock          = errors.New("format jam harus HH:MM atau HH:MM:SS")
	ErrClockOutBeforeClockIn = errors.New("waktu clock-out lebih awal dari clock-in")
	ErrNilLocation           = errors.New("zona waktu tenant wajib diisi")
)

// Schedule adalah bagian dari work_schedules yang dibutuhkan untuk menghitung status.
// StartTime/EndTime memakai format yang sama dengan repository.WorkSchedule ("08:00").
type Schedule struct {
	StartTime            string
	EndTime              string
	ToleranceMinutes     int // toleransi keterlambatan, contoh 15
	OvertimeAfterMinutes int // lembur baru dihitung bila kelebihan kerja >= nilai ini
	MaxOvertimeMinutes   int // batas atas lembur per hari; 0 berarti tanpa batas
}

// LoadTimezone memuat zona waktu tenant; string kosong memakai DefaultTimezone.
func LoadTimezone(name string) (*time.Location, error) {
	if name == "" {
		name = DefaultTimezone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("zona waktu %q tidak dikenal: %w", name, err)
	}
	return loc, nil
}

// WorkDate mengembalikan tanggal kerja (jam 00:00) menurut zona waktu tenant.
// Contoh: 2026-11-10T23:30Z = 2026-11-11 07:30 WITA, jadi tanggal kerjanya 11 November.
func WorkDate(t time.Time, loc *time.Location) (time.Time, error) {
	if loc == nil {
		return time.Time{}, ErrNilLocation
	}
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc), nil
}

// shiftBounds menghitung waktu mulai dan selesai shift pada tanggal kerja tertentu.
// Bila jam selesai <= jam mulai (shift malam, contoh 22:00-06:00), jam selesai jatuh besok.
func shiftBounds(workDate time.Time, sch Schedule, loc *time.Location) (start, end time.Time, err error) {
	sh, sm, err := parseClock(sch.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("start_time: %w", err)
	}
	eh, em, err := parseClock(sch.EndTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("end_time: %w", err)
	}

	y, m, d := workDate.In(loc).Date()
	start = time.Date(y, m, d, sh, sm, 0, 0, loc)
	end = time.Date(y, m, d, eh, em, 0, 0, loc)
	if !end.After(start) {
		end = end.AddDate(0, 0, 1)
	}
	return start, end, nil
}

// ClockInResult adalah hasil evaluasi clock-in.
type ClockInResult struct {
	WorkDate    time.Time
	Status      Status
	LateMinutes int
}

// EvaluateClockIn menentukan status Hadir (on_time) atau Terlambat (late) - FR-08.
// Aturan: terlambat bila clock-in lewat dari jam mulai + toleransi. Bila terlambat,
// menit keterlambatan dihitung dari jam mulai (bukan dari akhir toleransi), dibulatkan ke bawah.
func EvaluateClockIn(clockIn time.Time, sch Schedule, loc *time.Location) (ClockInResult, error) {
	workDate, err := WorkDate(clockIn, loc)
	if err != nil {
		return ClockInResult{}, err
	}
	start, _, err := shiftBounds(workDate, sch, loc)
	if err != nil {
		return ClockInResult{}, err
	}

	tolerance := time.Duration(max(0, sch.ToleranceMinutes)) * time.Minute
	res := ClockInResult{WorkDate: workDate, Status: StatusOnTime}

	if clockIn.After(start.Add(tolerance)) {
		res.Status = StatusLate
		res.LateMinutes = int(clockIn.Sub(start) / time.Minute)
	}
	return res, nil
}

// ClockOutResult adalah hasil evaluasi clock-out.
type ClockOutResult struct {
	EarlyLeaveMinutes int
	OvertimeMinutes   int
}

// EvaluateClockOut menghitung pulang cepat atau lembur - FR-07, ISSUE-64.
//   - Clock-out sebelum jam selesai: EarlyLeaveMinutes = selisihnya.
//   - Clock-out setelah jam selesai: lembur dihitung dari jam selesai, tetapi hanya bila
//     kelebihannya >= OvertimeAfterMinutes, lalu dibatasi MaxOvertimeMinutes (bila > 0).
//
// Jam selesai dihitung dari tanggal kerja clock-in, sehingga shift malam tetap benar.
func EvaluateClockOut(clockIn, clockOut time.Time, sch Schedule, loc *time.Location) (ClockOutResult, error) {
	if clockOut.Before(clockIn) {
		return ClockOutResult{}, ErrClockOutBeforeClockIn
	}
	workDate, err := WorkDate(clockIn, loc)
	if err != nil {
		return ClockOutResult{}, err
	}
	_, end, err := shiftBounds(workDate, sch, loc)
	if err != nil {
		return ClockOutResult{}, err
	}

	var res ClockOutResult
	if clockOut.Before(end) {
		res.EarlyLeaveMinutes = int(end.Sub(clockOut) / time.Minute)
		return res, nil
	}

	extra := int(clockOut.Sub(end) / time.Minute)
	if extra > 0 && extra >= max(0, sch.OvertimeAfterMinutes) {
		res.OvertimeMinutes = extra
		if sch.MaxOvertimeMinutes > 0 && res.OvertimeMinutes > sch.MaxOvertimeMinutes {
			res.OvertimeMinutes = sch.MaxOvertimeMinutes
		}
	}
	return res, nil
}

// parseClock membaca "HH:MM" atau "HH:MM:SS" (format kolom TIME PostgreSQL). Detik diabaikan.
func parseClock(s string) (hour, minute int, err error) {
	for _, layout := range []string{"15:04", "15:04:05"} {
		if t, perr := time.Parse(layout, s); perr == nil {
			return t.Hour(), t.Minute(), nil
		}
	}
	return 0, 0, fmt.Errorf("%w: %q", ErrInvalidClock, s)
}
