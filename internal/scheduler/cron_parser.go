package scheduler

import (
	"fmt"
	"strings"
	"time"
)

type CronSchedule struct {
	second   []int
	minute   []int
	hour     []int
	day      []int
	month    []int
	weekday  []int
	location *time.Location
}

type CronParser struct {
	location *time.Location
}

func NewCronParser() *CronParser {
	return &CronParser{
		location: time.Local,
	}
}

func (p *CronParser) SetLocation(loc *time.Location) {
	p.location = loc
}

func (p *CronParser) Parse(expr string) (*CronSchedule, error) {
	parts := strings.Fields(expr)

	if len(parts) != 5 && len(parts) != 6 {
		return nil, fmt.Errorf("invalid cron expression: expected 5 or 6 parts, got %d", len(parts))
	}

	schedule := &CronSchedule{
		location: p.location,
	}

	var err error

	if len(parts) == 6 {
		schedule.second, err = p.parseField(parts[0], 0, 59)
		if err != nil {
			return nil, fmt.Errorf("invalid second field: %w", err)
		}
		schedule.minute, err = p.parseField(parts[1], 0, 59)
		if err != nil {
			return nil, fmt.Errorf("invalid minute field: %w", err)
		}
		schedule.hour, err = p.parseField(parts[2], 0, 23)
		if err != nil {
			return nil, fmt.Errorf("invalid hour field: %w", err)
		}
		schedule.day, err = p.parseField(parts[3], 1, 31)
		if err != nil {
			return nil, fmt.Errorf("invalid day field: %w", err)
		}
		schedule.month, err = p.parseField(parts[4], 1, 12)
		if err != nil {
			return nil, fmt.Errorf("invalid month field: %w", err)
		}
		schedule.weekday, err = p.parseField(parts[5], 0, 6)
		if err != nil {
			return nil, fmt.Errorf("invalid weekday field: %w", err)
		}
	} else {
		schedule.second = []int{0}
		schedule.minute, err = p.parseField(parts[0], 0, 59)
		if err != nil {
			return nil, fmt.Errorf("invalid minute field: %w", err)
		}
		schedule.hour, err = p.parseField(parts[1], 0, 23)
		if err != nil {
			return nil, fmt.Errorf("invalid hour field: %w", err)
		}
		schedule.day, err = p.parseField(parts[2], 1, 31)
		if err != nil {
			return nil, fmt.Errorf("invalid day field: %w", err)
		}
		schedule.month, err = p.parseField(parts[3], 1, 12)
		if err != nil {
			return nil, fmt.Errorf("invalid month field: %w", err)
		}
		schedule.weekday, err = p.parseField(parts[4], 0, 6)
		if err != nil {
			return nil, fmt.Errorf("invalid weekday field: %w", err)
		}
	}

	return schedule, nil
}

func (p *CronParser) parseField(field string, min, max int) ([]int, error) {
	if field == "*" {
		return p.generateRange(min, max), nil
	}

	values := make([]int, 0)

	parts := strings.Split(field, ",")
	for _, part := range parts {
		if strings.Contains(part, "/") {
			stepParts := strings.Split(part, "/")
			if len(stepParts) != 2 {
				return nil, fmt.Errorf("invalid step format: %s", part)
			}

			base := "*"
			if stepParts[0] != "*" {
				base = stepParts[0]
			}

			step, err := p.parseInt(stepParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid step value: %w", err)
			}

			if base == "*" {
				baseValues := p.generateRange(min, max)
				for i := 0; i < len(baseValues); i += step {
					values = append(values, baseValues[i])
				}
			} else if strings.Contains(base, "-") {
				rangeValues, err := p.parseRange(base, min, max)
				if err != nil {
					return nil, err
				}
				for i := 0; i < len(rangeValues); i += step {
					values = append(values, rangeValues[i])
				}
			} else {
				start, err := p.parseInt(base)
				if err != nil {
					return nil, fmt.Errorf("invalid base value: %w", err)
				}
				for v := start; v <= max; v += step {
					values = append(values, v)
				}
			}
		} else if strings.Contains(part, "-") {
			rangeValues, err := p.parseRange(part, min, max)
			if err != nil {
				return nil, err
			}
			values = append(values, rangeValues...)
		} else {
			value, err := p.parseInt(part)
			if err != nil {
				return nil, fmt.Errorf("invalid value: %w", err)
			}
			if value < min || value > max {
				return nil, fmt.Errorf("value %d out of range [%d, %d]", value, min, max)
			}
			values = append(values, value)
		}
	}

	return p.uniqueSorted(values), nil
}

func (p *CronParser) parseRange(field string, min, max int) ([]int, error) {
	parts := strings.Split(field, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format: %s", field)
	}

	start, err := p.parseInt(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid range start: %w", err)
	}

	end, err := p.parseInt(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid range end: %w", err)
	}

	if start < min || start > max {
		return nil, fmt.Errorf("range start %d out of range [%d, %d]", start, min, max)
	}

	if end < min || end > max {
		return nil, fmt.Errorf("range end %d out of range [%d, %d]", end, min, max)
	}

	if start > end {
		return nil, fmt.Errorf("range start %d greater than end %d", start, end)
	}

	return p.generateRange(start, end), nil
}

func (p *CronParser) parseInt(s string) (int, error) {
	var value int
	_, err := fmt.Sscanf(s, "%d", &value)
	return value, err
}

func (p *CronParser) generateRange(min, max int) []int {
	values := make([]int, max-min+1)
	for i := range values {
		values[i] = min + i
	}
	return values
}

func (p *CronParser) uniqueSorted(values []int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0)

	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

func (s *CronSchedule) Next(t time.Time) time.Time {
	if s.location == nil {
		s.location = time.Local
	}

	t = t.In(s.location).Add(time.Second).Truncate(time.Second)

	for {
		if s.matches(t) {
			return t
		}

		t = s.nextTime(t)
	}
}

func (s *CronSchedule) matches(t time.Time) bool {
	if !s.matchesField(t.Second(), s.second) ||
		!s.matchesField(t.Minute(), s.minute) ||
		!s.matchesField(t.Hour(), s.hour) ||
		!s.matchesField(int(t.Month()), s.month) {
		return false
	}

	dayMatches := s.matchesField(t.Day(), s.day)
	weekdayMatches := s.matchesField(int(t.Weekday()), s.weekday)

	daySpecified := len(s.day) > 0 && !s.isAllValues(s.day, 1, 31)
	weekdaySpecified := len(s.weekday) > 0 && !s.isAllValues(s.weekday, 0, 6)

	if daySpecified && weekdaySpecified {
		return dayMatches && weekdayMatches
	}
	if daySpecified {
		return dayMatches
	}
	if weekdaySpecified {
		return weekdayMatches
	}

	return true
}

func (s *CronSchedule) isAllValues(values []int, min, max int) bool {
	if len(values) != max-min+1 {
		return false
	}
	for i := min; i <= max; i++ {
		if !s.matchesField(i, values) {
			return false
		}
	}
	return true
}

func (s *CronSchedule) matchesField(value int, field []int) bool {
	for _, v := range field {
		if v == value {
			return true
		}
	}
	return false
}

func (s *CronSchedule) nextTime(t time.Time) time.Time {
	for {
		t = t.Add(time.Second)
		if s.matches(t) {
			return t
		}
	}
}

func (s *CronSchedule) nextValue(current int, values []int, min, max int) int {
	for _, v := range values {
		if v > current {
			return v
		}
	}
	return min
}

func (s *CronSchedule) Prev(t time.Time) time.Time {
	if s.location == nil {
		s.location = time.Local
	}

	t = t.In(s.location).Add(-time.Second).Truncate(time.Second)

	for {
		if s.matches(t) {
			return t
		}

		t = s.prevTime(t)
	}
}

func (s *CronSchedule) prevTime(t time.Time) time.Time {
	second := s.prevValue(t.Second(), s.second, 0, 59)
	if second != t.Second() {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), second, 0, s.location)
	}

	minute := s.prevValue(t.Minute(), s.minute, 0, 59)
	if minute != t.Minute() {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 59, 0, s.location)
	}

	hour := s.prevValue(t.Hour(), s.hour, 0, 23)
	if hour != t.Hour() {
		return time.Date(t.Year(), t.Month(), t.Day(), hour, 59, 59, 0, s.location)
	}

	day := s.prevValue(t.Day(), s.day, 1, 31)
	month := s.prevValue(int(t.Month()), s.month, 1, 12)
	year := t.Year()

	if day != t.Day() {
		if month == int(t.Month()) {
			return time.Date(year, time.Month(month), day, 23, 59, 59, 0, s.location)
		}
		return time.Date(year, time.Month(month), 31, 23, 59, 59, 0, s.location)
	}

	if month != int(t.Month()) {
		return time.Date(year, time.Month(month), 31, 23, 59, 59, 0, s.location)
	}

	return time.Date(year-1, 12, 31, 23, 59, 59, 0, s.location)
}

func (s *CronSchedule) prevValue(current int, values []int, min, max int) int {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] < current {
			return values[i]
		}
	}
	return max
}

func (s *CronSchedule) String() string {
	return fmt.Sprintf("CronSchedule{second: %v, minute: %v, hour: %v, day: %v, month: %v, weekday: %v}",
		s.second, s.minute, s.hour, s.day, s.month, s.weekday)
}

func ParseCronExpression(expr string) (*CronSchedule, error) {
	parser := NewCronParser()
	return parser.Parse(expr)
}

func NextRunTime(expr string, from time.Time) (time.Time, error) {
	schedule, err := ParseCronExpression(expr)
	if err != nil {
		return time.Time{}, err
	}
	return schedule.Next(from), nil
}

func IsValidCronExpression(expr string) bool {
	_, err := ParseCronExpression(expr)
	return err == nil
}
