package scheduler

import (
	"testing"
	"time"
)

func TestNewCronParser(t *testing.T) {
	parser := NewCronParser()

	if parser == nil {
		t.Error("Expected parser to be created")
	}

	if parser.location == nil {
		t.Error("Expected location to be initialized")
	}
}

func TestParseSimple(t *testing.T) {
	parser := NewCronParser()

	schedule, err := parser.Parse("0 * * * *")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if schedule == nil {
		t.Error("Expected schedule to be created")
	}

	if len(schedule.minute) != 1 || schedule.minute[0] != 0 {
		t.Error("Expected minute to be 0")
	}
}

func TestParseEveryMinute(t *testing.T) {
	parser := NewCronParser()

	schedule, err := parser.Parse("* * * * *")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(schedule.minute) != 60 {
		t.Errorf("Expected 60 minutes, got %d", len(schedule.minute))
	}
}

func TestParseRange(t *testing.T) {
	parser := NewCronParser()

	schedule, err := parser.Parse("0-5 * * * *")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(schedule.minute) != 6 {
		t.Errorf("Expected 6 minutes, got %d", len(schedule.minute))
	}
}

func TestParseList(t *testing.T) {
	parser := NewCronParser()

	schedule, err := parser.Parse("0,5,10 * * * *")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(schedule.minute) != 3 {
		t.Errorf("Expected 3 minutes, got %d", len(schedule.minute))
	}
}

func TestParseStep(t *testing.T) {
	parser := NewCronParser()

	schedule, err := parser.Parse("*/5 * * * *")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(schedule.minute) != 12 {
		t.Errorf("Expected 12 minutes, got %d", len(schedule.minute))
	}
}

func TestParseWithSeconds(t *testing.T) {
	parser := NewCronParser()

	schedule, err := parser.Parse("0 0 * * * *")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(schedule.second) != 1 || schedule.second[0] != 0 {
		t.Error("Expected second to be 0")
	}
}

func TestParseInvalid(t *testing.T) {
	parser := NewCronParser()

	_, err := parser.Parse("invalid")
	if err == nil {
		t.Error("Expected error for invalid expression")
	}
}

func TestParseInvalidParts(t *testing.T) {
	parser := NewCronParser()

	_, err := parser.Parse("0 * * *")
	if err == nil {
		t.Error("Expected error for invalid number of parts")
	}
}

func TestParseInvalidRange(t *testing.T) {
	parser := NewCronParser()

	_, err := parser.Parse("60 * * * *")
	if err == nil {
		t.Error("Expected error for invalid minute range")
	}
}

func TestSetLocation(t *testing.T) {
	parser := NewCronParser()
	utc := time.UTC

	parser.SetLocation(utc)

	if parser.location != utc {
		t.Error("Expected location to be updated")
	}
}

func TestNext(t *testing.T) {
	parser := NewCronParser()
	parser.SetLocation(time.UTC)
	schedule, err := parser.Parse("0 * * * *")
	if err != nil {
		t.Fatalf("Failed to parse schedule: %v", err)
	}

	now := time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC)
	next := schedule.Next(now)

	if !next.After(now) {
		t.Error("Expected next time to be after now")
	}

	if next.Minute() != 0 {
		t.Errorf("Expected next minute to be 0, got %d", next.Minute())
	}
}

func TestNextEveryMinute(t *testing.T) {
	parser := NewCronParser()
	parser.SetLocation(time.UTC)
	schedule, err := parser.Parse("* * * * *")
	if err != nil {
		t.Fatalf("Failed to parse schedule: %v", err)
	}

	now := time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC)
	next := schedule.Next(now)

	if !next.After(now) {
		t.Error("Expected next time to be after now")
	}

	if next.Second() != 0 {
		t.Errorf("Expected next second to be 0, got %d", next.Second())
	}
}

func TestNextWithSeconds(t *testing.T) {
	parser := NewCronParser()
	parser.SetLocation(time.UTC)
	schedule, err := parser.Parse("0 0 * * * *")
	if err != nil {
		t.Fatalf("Failed to parse schedule: %v", err)
	}

	now := time.Date(2024, 1, 1, 12, 30, 30, 0, time.UTC)
	next := schedule.Next(now)

	if !next.After(now) {
		t.Error("Expected next time to be after now")
	}

	if next.Second() != 0 {
		t.Errorf("Expected next second to be 0, got %d", next.Second())
	}

	if next.Minute() != 0 {
		t.Errorf("Expected next minute to be 0, got %d", next.Minute())
	}
}

func TestNextHourly(t *testing.T) {
	parser := NewCronParser()
	parser.SetLocation(time.UTC)
	schedule, err := parser.Parse("0 * * * *")
	if err != nil {
		t.Fatalf("Failed to parse schedule: %v", err)
	}

	now := time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC)
	next := schedule.Next(now)

	if !next.After(now) {
		t.Error("Expected next time to be after now")
	}

	if next.Hour() != 13 {
		t.Errorf("Expected next hour to be 13, got %d", next.Hour())
	}

	if next.Minute() != 0 {
		t.Errorf("Expected next minute to be 0, got %d", next.Minute())
	}
}

func TestNextDaily(t *testing.T) {
	parser := NewCronParser()
	parser.SetLocation(time.UTC)
	schedule, err := parser.Parse("0 0 * * *")
	if err != nil {
		t.Fatalf("Failed to parse schedule: %v", err)
	}

	now := time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC)
	next := schedule.Next(now)

	if !next.After(now) {
		t.Error("Expected next time to be after now")
	}

	if next.Day() != 2 {
		t.Errorf("Expected next day to be 2, got %d", next.Day())
	}

	if next.Hour() != 0 {
		t.Errorf("Expected next hour to be 0, got %d", next.Hour())
	}
}

func TestNextMonthly(t *testing.T) {
	parser := NewCronParser()
	parser.SetLocation(time.UTC)
	schedule, err := parser.Parse("0 0 1 * *")
	if err != nil {
		t.Fatalf("Failed to parse schedule: %v", err)
	}

	now := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)
	next := schedule.Next(now)

	if !next.After(now) {
		t.Error("Expected next time to be after now")
	}

	if next.Month() != 2 {
		t.Errorf("Expected next month to be 2, got %d", next.Month())
	}

	if next.Day() != 1 {
		t.Errorf("Expected next day to be 1, got %d", next.Day())
	}
}
