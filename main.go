package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBold   = "\033[1m"
)

// ─── Data structures ──────────────────────────────────────────────────────────

type ActivityRow struct {
	LineNum         int
	ID              int
	Server          string
	ExceptServer    string
	ActivityID      int
	ActivityGroup   int
	ActivityExclude string
	Note            string
	IsOpen          int
	Area            int
	AcType          int
	SeType          int
	StartTimeType   int
	StartTime       string
	EndTime         string
	AppearTime      string
	DisappearTime   string
	EffectiveTime   string
	LevelVis        int
	VipVis          int
	ConditionVis    int
	ConditionNum    int
	ServerLimit     int
	PlayerLimit     int
	Reissue         int
	Plate           int
	ChannelType     int
	Channel         int
	Integral        int
	RawFields       []string
}

type ValidationError struct {
	LineNum int
	RowID   int
	Message string
}

type RuleResult struct {
	RuleNum int
	Name    string
	Errors  []ValidationError
}

// ─── Entry point ──────────────────────────────────────────────────────────────

var availableRules = []struct {
	id   int
	name string
	fn   func([]ActivityRow) RuleResult
}{
	{1, "Ô trống có dấu cách", rule1},
	{2, "Trùng ID", rule2},
	{3, "Khoảng serverID không chính xác", rule3},
	{4, "Binh Pháp Nghiên Tập – chồng thời gian (±7 ngày)", rule4},
	{5, "Binh Pháp Nghiên Tập – yêu cầu field", rule5},
	{6, "Tướng UR – chồng thời gian (±7 ngày)", rule6},
	{7, "Tướng UR – yêu cầu field", rule7},
	{8, "Vũ Khí Chuyên Dụng – chồng thời gian (±7 ngày)", rule8},
	{9, "Vũ Khí Chuyên Dụng – yêu cầu field", rule9},
	{10, "serverLimit khi startTimeType=1", rule10},
	{11, "Format datetime khi startTimeType=1", rule11},
	{12, "Sự kiện song hành phải cùng thời gian", rule12},
	{13, "startTime=appearTime & endTime=disappearTime", rule13},
	{14, "Quỹ tháng cố định (5013–5025)", rule14},
}

func main() {
	path := "activityopen.csv"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	rows, warnings, err := parseCSV(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Lỗi đọc file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(colorBold + "CHỌN RULE MUỐN CHẠY:" + colorReset)
	fmt.Println("0. Chạy tất cả rule")
	for _, r := range availableRules {
		fmt.Printf("%d. %s\n", r.id, r.name)
	}
	fmt.Print(colorYellow + "Lựa chọn của bạn (nhập số): " + colorReset)

	var input string
	fmt.Scanln(&input)
	choice, _ := strconv.Atoi(input)

	var results []RuleResult
	if choice == 0 {
		results = runAllRules(rows)
	} else {
		found := false
		for _, r := range availableRules {
			if r.id == choice {
				results = append(results, r.fn(rows))
				found = true
				break
			}
		}
		if !found {
			fmt.Println(colorRed + "Lựa chọn không hợp lệ. Đang chạy tất cả rule mặc định..." + colorReset)
			results = runAllRules(rows)
		}
	}

	printReport(path, results, warnings)
}

// ─── CSV Parser ───────────────────────────────────────────────────────────────

func parseCSV(path string) ([]ActivityRow, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	var rows []ActivityRow
	var warnings []string

	// First 4 rows are metadata headers — skip them
	for i := 4; i < len(all); i++ {
		lineNum := i + 1
		record := all[i]

		if len(record) == 0 {
			continue
		}
		// Skip cluster label rows and blank rows (id field empty)
		if strings.TrimSpace(record[0]) == "" {
			continue
		}

		get := func(idx int) string {
			if idx < len(record) {
				return record[idx]
			}
			return ""
		}

		row := ActivityRow{LineNum: lineNum, RawFields: record}

		parseInt := func(idx int, name string) int {
			s := strings.TrimSpace(get(idx))
			if s == "" {
				return 0
			}
			v, err := strconv.Atoi(s)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("Line %d: không parse được %s=%q thành int", lineNum, name, s))
			}
			return v
		}

		row.ID = parseInt(0, "id")
		row.Server = strings.TrimSpace(get(1))
		row.ExceptServer = strings.TrimSpace(get(2))
		row.ActivityID = parseInt(3, "activityId")
		row.ActivityGroup = parseInt(4, "activityGroup")
		row.ActivityExclude = strings.TrimSpace(get(5))
		row.Note = strings.TrimSpace(get(6))
		row.IsOpen = parseInt(7, "isopen")
		row.Area = parseInt(8, "area")
		row.AcType = parseInt(9, "acType")
		row.SeType = parseInt(10, "seType")
		row.StartTimeType = parseInt(11, "startTimeType")
		row.StartTime = strings.TrimSpace(get(12))
		row.EndTime = strings.TrimSpace(get(13))
		row.AppearTime = strings.TrimSpace(get(14))
		row.DisappearTime = strings.TrimSpace(get(15))
		row.EffectiveTime = strings.TrimSpace(get(16))
		row.LevelVis = parseInt(17, "levelVis")
		row.VipVis = parseInt(18, "vipVis")
		row.ConditionVis = parseInt(19, "conditionVis")
		row.ConditionNum = parseInt(20, "conditionNum")
		row.ServerLimit = parseInt(21, "serverLimit")
		row.PlayerLimit = parseInt(22, "playerLimit")
		row.Reissue = parseInt(23, "reissue")
		row.Plate = parseInt(24, "plate")
		row.ChannelType = parseInt(25, "channelType")
		row.Channel = parseInt(26, "channel")
		row.Integral = parseInt(27, "integral")

		rows = append(rows, row)
	}

	return rows, warnings, nil
}

// ─── Datetime helpers ─────────────────────────────────────────────────────────

// Data uses YYYY/MM/DD HH/MM/SS (all slashes). Also support colon variant.
var dateFormats = []string{
	"2006/01/02 15/04/05",
	"2006/01/02 15:04:05",
	"2006/1/2 15/04/05",
	"2006/1/2  15/04/05",
	"2006/01/02  15/04/05",
}

func parseDateTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range dateFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("không parse được datetime %q", s)
}

// ─── Server range validation (Rule 3) ────────────────────────────────────────

func validateServerRange(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// Unmarshal outer array — elements can be inner arrays or plain numbers
	var outer []json.RawMessage
	if err := json.Unmarshal([]byte(s), &outer); err != nil {
		return fmt.Errorf("không parse được JSON: %v", err)
	}
	if len(outer) == 0 {
		return fmt.Errorf("mảng rỗng")
	}

	for _, elem := range outer {
		// Try inner array [start, end, ...]
		var arr []float64
		if err := json.Unmarshal(elem, &arr); err == nil {
			if len(arr) == 0 {
				return fmt.Errorf("inner array rỗng")
			}
			for _, v := range arr {
				if v < 0 || v > 9999999999 {
					return fmt.Errorf("giá trị %.0f nằm ngoài [0, 9999999999]", v)
				}
			}
			if len(arr) == 2 && arr[0] >= arr[1] {
				return fmt.Errorf("start=%.0f >= end=%.0f", arr[0], arr[1])
			}
		} else {
			// Try single number
			var num float64
			if err2 := json.Unmarshal(elem, &num); err2 != nil {
				return fmt.Errorf("phần tử không hợp lệ: %s", string(elem))
			}
			if num < 0 || num > 9999999999 {
				return fmt.Errorf("giá trị %.0f nằm ngoài [0, 9999999999]", num)
			}
		}
	}
	return nil
}

// ─── Time range helpers for Rules 4/6/8/12 ───────────────────────────────────

type timeRange struct {
	row      *ActivityRow
	absolute bool
	absStart time.Time
	absEnd   time.Time
	relStart float64 // hours from server open (startTimeType=2/3)
	relEnd   float64
}

func parseTimeRange(row *ActivityRow) (timeRange, bool) {
	tr := timeRange{row: row}
	switch row.StartTimeType {
	case 1:
		s, err1 := parseDateTime(row.StartTime)
		e, err2 := parseDateTime(row.EndTime)
		if err1 != nil || err2 != nil {
			return tr, false
		}
		tr.absolute = true
		tr.absStart, tr.absEnd = s, e
		return tr, true
	case 2, 3:
		s, err1 := strconv.ParseFloat(strings.TrimSpace(row.StartTime), 64)
		e, err2 := strconv.ParseFloat(strings.TrimSpace(row.EndTime), 64)
		if err1 != nil || err2 != nil {
			return tr, false
		}
		tr.absolute = false
		tr.relStart, tr.relEnd = s, e
		return tr, true
	}
	return tr, false
}

// overlapsWithBuffer checks whether [b.start, b.end] intersects [a.start−buf, a.end+buf].
func overlapsWithBuffer(a, b timeRange, bufDays int) bool {
	if a.absolute != b.absolute {
		return false
	}
	if a.absolute {
		buf := time.Duration(bufDays) * 24 * time.Hour
		return b.absStart.Before(a.absEnd.Add(buf)) && b.absEnd.After(a.absStart.Add(-buf))
	}
	buf := float64(bufDays * 24)
	return b.relStart < a.relEnd+buf && b.relEnd > a.relStart-buf
}

func overlaps(a, b timeRange) bool {
	return overlapsWithBuffer(a, b, 0)
}

// ─── Shared rule helpers ──────────────────────────────────────────────────────

// checkTimeOverlap implements Rules 4, 6, 8.
func checkTimeOverlap(rows []ActivityRow, activityIDs map[int]bool, ruleNum int, name string) RuleResult {
	result := RuleResult{RuleNum: ruleNum, Name: name}

	grouped := make(map[int][]*ActivityRow)
	for i := range rows {
		if activityIDs[rows[i].ActivityID] {
			grouped[rows[i].ActivityID] = append(grouped[rows[i].ActivityID], &rows[i])
		}
	}

	for _, group := range grouped {
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				a, b := group[i], group[j]
				trA, okA := parseTimeRange(a)
				trB, okB := parseTimeRange(b)
				if !okA || !okB {
					continue
				}
				if overlapsWithBuffer(trA, trB, 7) {
					result.Errors = append(result.Errors, ValidationError{
						LineNum: b.LineNum, RowID: b.ID,
						Message: fmt.Sprintf(
							"activityId=%d chồng thời gian (±7 ngày) với line %d (id=%d)",
							b.ActivityID, a.LineNum, a.ID,
						),
					})
				}
			}
		}
	}
	return result
}

// FieldReq describes one required field value.
type FieldReq struct {
	Name     string
	Expected int
	Actual   func(*ActivityRow) int
}

// checkRequiredFields implements Rules 5, 7, 9.
func checkRequiredFields(rows []ActivityRow, activityIDs map[int]bool, reqs []FieldReq, ruleNum int, name string) RuleResult {
	result := RuleResult{RuleNum: ruleNum, Name: name}
	for i := range rows {
		row := &rows[i]
		if !activityIDs[row.ActivityID] {
			continue
		}
		for _, req := range reqs {
			if got := req.Actual(row); got != req.Expected {
				result.Errors = append(result.Errors, ValidationError{
					LineNum: row.LineNum, RowID: row.ID,
					Message: fmt.Sprintf("activityId=%d  %s=%d (expected %d)", row.ActivityID, req.Name, got, req.Expected),
				})
			}
		}
	}
	return result
}

// ─── Rule implementations ─────────────────────────────────────────────────────

func runAllRules(rows []ActivityRow) []RuleResult {
	return []RuleResult{
		rule1(rows),
		rule2(rows),
		rule3(rows),
		rule4(rows),
		rule5(rows),
		rule6(rows),
		rule7(rows),
		rule8(rows),
		rule9(rows),
		rule10(rows),
		// rule11(rows),
		rule12(rows),
		rule13(rows),
		rule14(rows),
	}
}

// Rule 1 – Ô trống có dấu cách
func rule1(rows []ActivityRow) RuleResult {
	result := RuleResult{RuleNum: 1, Name: "Ô trống có dấu cách"}
	for _, row := range rows {
		for colIdx, val := range row.RawFields {
			if val == "" {
				continue
			}
			if strings.TrimSpace(val) == "" {
				result.Errors = append(result.Errors, ValidationError{
					LineNum: row.LineNum, RowID: row.ID,
					Message: fmt.Sprintf("cột %d: %q chứa khoảng trắng (phải là \"\")", colIdx, val),
				})
			}
		}
	}
	return result
}

// Rule 2 – Trùng ID
func rule2(rows []ActivityRow) RuleResult {
	result := RuleResult{RuleNum: 2, Name: "Trùng ID"}
	seen := make(map[int]int) // id → lineNum
	for _, row := range rows {
		if first, exists := seen[row.ID]; exists {
			result.Errors = append(result.Errors, ValidationError{
				LineNum: row.LineNum, RowID: row.ID,
				Message: fmt.Sprintf("id=%d đã xuất hiện ở line %d", row.ID, first),
			})
		} else {
			seen[row.ID] = row.LineNum
		}
	}
	return result
}

// Rule 3 – Khoảng serverID không chính xác
func rule3(rows []ActivityRow) RuleResult {
	result := RuleResult{RuleNum: 3, Name: "Khoảng serverID không chính xác"}
	for _, row := range rows {
		if row.Server != "" {
			if err := validateServerRange(row.Server); err != nil {
				result.Errors = append(result.Errors, ValidationError{
					LineNum: row.LineNum, RowID: row.ID,
					Message: fmt.Sprintf("server=%q  →  %v", row.Server, err),
				})
			}
		}
		if row.ExceptServer != "" {
			if err := validateServerRange(row.ExceptServer); err != nil {
				result.Errors = append(result.Errors, ValidationError{
					LineNum: row.LineNum, RowID: row.ID,
					Message: fmt.Sprintf("except_server=%q  →  %v", row.ExceptServer, err),
				})
			}
		}
	}
	return result
}

// Rule 4 – Binh Pháp Nghiên Tập: chồng thời gian
func rule4(rows []ActivityRow) RuleResult {
	ids := map[int]bool{2002: true, 2166: true, 2167: true, 2168: true}
	return checkTimeOverlap(rows, ids, 4, "Binh Pháp Nghiên Tập – chồng thời gian (±7 ngày)")
}

// Rule 5 – Binh Pháp Nghiên Tập: yêu cầu field
func rule5(rows []ActivityRow) RuleResult {
	ids := map[int]bool{2002: true, 2166: true, 2167: true, 2168: true}
	reqs := []FieldReq{
		{"isopen", 1, func(r *ActivityRow) int { return r.IsOpen }},
		{"area", 2, func(r *ActivityRow) int { return r.Area }},
		{"acType", 159, func(r *ActivityRow) int { return r.AcType }},
		{"seType", 159, func(r *ActivityRow) int { return r.SeType }},
		{"levelVis", 55, func(r *ActivityRow) int { return r.LevelVis }},
		{"vipVis", 0, func(r *ActivityRow) int { return r.VipVis }},
		{"serverLimit", 168, func(r *ActivityRow) int { return r.ServerLimit }},
		{"reissue", 2, func(r *ActivityRow) int { return r.Reissue }},
	}
	return checkRequiredFields(rows, ids, reqs, 5, "Binh Pháp Nghiên Tập – yêu cầu field")
}

// Rule 6 – Tướng UR: chồng thời gian
func rule6(rows []ActivityRow) RuleResult {
	ids := map[int]bool{7601: true, 7602: true, 7603: true, 7604: true}
	return checkTimeOverlap(rows, ids, 6, "Tướng UR – chồng thời gian (±7 ngày)")
}

// Rule 7 – Tướng UR: yêu cầu field
func rule7(rows []ActivityRow) RuleResult {
	ids := map[int]bool{7601: true, 7602: true, 7603: true, 7604: true}
	reqs := []FieldReq{
		{"isopen", 1, func(r *ActivityRow) int { return r.IsOpen }},
		{"area", 1, func(r *ActivityRow) int { return r.Area }},
		{"acType", 165, func(r *ActivityRow) int { return r.AcType }},
		{"seType", 165, func(r *ActivityRow) int { return r.SeType }},
		{"levelVis", 10, func(r *ActivityRow) int { return r.LevelVis }},
		{"vipVis", 0, func(r *ActivityRow) int { return r.VipVis }},
		{"serverLimit", 168, func(r *ActivityRow) int { return r.ServerLimit }},
		{"reissue", 1, func(r *ActivityRow) int { return r.Reissue }},
		{"integral", 1, func(r *ActivityRow) int { return r.Integral }},
	}
	return checkRequiredFields(rows, ids, reqs, 7, "Tướng UR – yêu cầu field")
}

// Rule 8 – Vũ Khí Chuyên Dụng: chồng thời gian
func rule8(rows []ActivityRow) RuleResult {
	ids := map[int]bool{80102: true, 80103: true, 80104: true, 80105: true, 80106: true}
	return checkTimeOverlap(rows, ids, 8, "Vũ Khí Chuyên Dụng – chồng thời gian (±7 ngày)")
}

// Rule 9 – Vũ Khí Chuyên Dụng: yêu cầu field
func rule9(rows []ActivityRow) RuleResult {
	ids := map[int]bool{80102: true, 80103: true, 80104: true, 80105: true, 80106: true}
	reqs := []FieldReq{
		{"isopen", 1, func(r *ActivityRow) int { return r.IsOpen }},
		{"area", 1, func(r *ActivityRow) int { return r.Area }},
		{"acType", 167, func(r *ActivityRow) int { return r.AcType }},
		{"seType", 167, func(r *ActivityRow) int { return r.SeType }},
		{"levelVis", 26, func(r *ActivityRow) int { return r.LevelVis }},
		{"vipVis", 0, func(r *ActivityRow) int { return r.VipVis }},
		{"serverLimit", 168, func(r *ActivityRow) int { return r.ServerLimit }},
		{"reissue", 2, func(r *ActivityRow) int { return r.Reissue }},
	}
	return checkRequiredFields(rows, ids, reqs, 9, "Vũ Khí Chuyên Dụng – yêu cầu field")
}

// Rule 10 – serverLimit phải là 0 hoặc 168 khi startTimeType=1
func rule10(rows []ActivityRow) RuleResult {
	result := RuleResult{RuleNum: 10, Name: "serverLimit khi startTimeType=1"}
	for _, row := range rows {
		if row.StartTimeType != 1 {
			continue
		}
		if row.ServerLimit != 0 && row.ServerLimit != 168 {
			result.Errors = append(result.Errors, ValidationError{
				LineNum: row.LineNum, RowID: row.ID,
				Message: fmt.Sprintf("startTimeType=1  serverLimit=%d (chỉ cho phép 0 hoặc 168)", row.ServerLimit),
			})
		}
	}
	return result
}

// Rule 11 – Format datetime khi startTimeType=1
func rule11(rows []ActivityRow) RuleResult {
	result := RuleResult{RuleNum: 11, Name: "Format datetime khi startTimeType=1"}
	const layout = "2006/01/02 15:04:05"
	timeFields := []struct {
		name string
		get  func(*ActivityRow) string
	}{
		{"startTime", func(r *ActivityRow) string { return r.StartTime }},
		{"endTime", func(r *ActivityRow) string { return r.EndTime }},
		{"appearTime", func(r *ActivityRow) string { return r.AppearTime }},
		{"disappearTime", func(r *ActivityRow) string { return r.DisappearTime }},
	}
	for _, row := range rows {
		if row.StartTimeType != 1 {
			continue
		}
		for _, tf := range timeFields {
			val := tf.get(&row)
			if val == "" {
				continue
			}
			if _, err := time.Parse(layout, val); err != nil {
				result.Errors = append(result.Errors, ValidationError{
					LineNum: row.LineNum, RowID: row.ID,
					Message: fmt.Sprintf("%s=%q  sai format (expected YYYY/MM/DD HH:MM:SS)", tf.name, val),
				})
			}
		}
	}
	return result
}

// Rule 12 – Sự kiện song hành phải cùng thời gian
func rule12(rows []ActivityRow) RuleResult {
	result := RuleResult{RuleNum: 12, Name: "Sự kiện song hành phải cùng thời gian"}

	type pairGroup struct {
		setA  map[int]bool
		setB  map[int]bool
		label string
	}
	groups := []pairGroup{
		{map[int]bool{3510: true}, map[int]bool{3511: true}, "3510↔3511"},
		{map[int]bool{194: true, 355: true, 356: true}, map[int]bool{156: true}, "194/355/356↔156"},
		{map[int]bool{7002: true}, map[int]bool{299: true}, "7002↔299"},
	}

	for _, g := range groups {
		var setARows, setBRows []*ActivityRow
		for i := range rows {
			if g.setA[rows[i].ActivityID] {
				setARows = append(setARows, &rows[i])
			}
			if g.setB[rows[i].ActivityID] {
				setBRows = append(setBRows, &rows[i])
			}
		}

		for _, a := range setARows {
			for _, b := range setBRows {
				trA, okA := parseTimeRange(a)
				trB, okB := parseTimeRange(b)
				if !okA || !okB {
					continue
				}
				if !overlaps(trA, trB) {
					continue
				}
				check := func(fieldName, valA, valB string) {
					if valA != valB {
						result.Errors = append(result.Errors, ValidationError{
							LineNum: b.LineNum, RowID: b.ID,
							Message: fmt.Sprintf(
								"[%s] %s không khớp: line %d (actId=%d) %q ≠ line %d (actId=%d) %q",
								g.label, fieldName, a.LineNum, a.ActivityID, valA, b.LineNum, b.ActivityID, valB,
							),
						})
					}
				}
				check("startTime", a.StartTime, b.StartTime)
				check("appearTime", a.AppearTime, b.AppearTime)
				check("endTime", a.EndTime, b.EndTime)
				check("disappearTime", a.DisappearTime, b.DisappearTime)
			}
		}
	}
	return result
}

// Rule 13 – startTime = appearTime, endTime = disappearTime (tất cả dòng)
func rule13(rows []ActivityRow) RuleResult {
	result := RuleResult{RuleNum: 13, Name: "startTime=appearTime & endTime=disappearTime"}
	for _, row := range rows {
		if row.StartTime != row.AppearTime {
			result.Errors = append(result.Errors, ValidationError{
				LineNum: row.LineNum, RowID: row.ID,
				Message: fmt.Sprintf("startTime=%q ≠ appearTime=%q", row.StartTime, row.AppearTime),
			})
		}
		if row.EndTime != row.DisappearTime {
			result.Errors = append(result.Errors, ValidationError{
				LineNum: row.LineNum, RowID: row.ID,
				Message: fmt.Sprintf("endTime=%q ≠ disappearTime=%q", row.EndTime, row.DisappearTime),
			})
		}
	}
	return result
}

// Rule 14 – Quỹ tháng cố định (activityId 5013–5025)
func rule14(rows []ActivityRow) RuleResult {
	result := RuleResult{RuleNum: 14, Name: "Quỹ tháng cố định (5013–5025)"}

	type monthSpec struct {
		startMonth int
		endMonth   int
		nextYear   bool // endTime năm = startTime năm + 1 (chỉ áp dụng 5025)
	}
	schedule := map[int]monthSpec{
		5013: {1, 2, false},
		5014: {2, 3, false},
		5015: {3, 4, false},
		5016: {4, 5, false},
		5017: {5, 6, false},
		5019: {6, 7, false},
		5020: {7, 8, false},
		5021: {8, 9, false},
		5022: {9, 10, false},
		5023: {10, 11, false},
		5024: {11, 12, false},
		5025: {12, 1, true},
	}

	fieldReqs := []FieldReq{
		{"isopen", 1, func(r *ActivityRow) int { return r.IsOpen }},
		{"area", 1, func(r *ActivityRow) int { return r.Area }},
		{"acType", 129, func(r *ActivityRow) int { return r.AcType }},
		{"seType", 129, func(r *ActivityRow) int { return r.SeType }},
		{"startTimeType", 5, func(r *ActivityRow) int { return r.StartTimeType }},
		{"levelVis", 26, func(r *ActivityRow) int { return r.LevelVis }},
		{"vipVis", 0, func(r *ActivityRow) int { return r.VipVis }},
		{"serverLimit", 168, func(r *ActivityRow) int { return r.ServerLimit }},
		{"playerLimit", 0, func(r *ActivityRow) int { return r.PlayerLimit }},
		{"reissue", 1, func(r *ActivityRow) int { return r.Reissue }},
		{"plate", 0, func(r *ActivityRow) int { return r.Plate }},
	}

	for _, row := range rows {
		spec, ok := schedule[row.ActivityID]
		if !ok {
			continue
		}

		for _, req := range fieldReqs {
			if got := req.Actual(&row); got != req.Expected {
				result.Errors = append(result.Errors, ValidationError{
					LineNum: row.LineNum, RowID: row.ID,
					Message: fmt.Sprintf("activityId=%d  %s=%d (expected %d)", row.ActivityID, req.Name, got, req.Expected),
				})
			}
		}

		// Check startTime: ngày 1, đúng tháng, 05:00:00
		if st, err := parseDateTime(row.StartTime); err == nil {
			if int(st.Month()) != spec.startMonth || st.Day() != 1 || st.Hour() != 5 || st.Minute() != 0 || st.Second() != 0 {
				result.Errors = append(result.Errors, ValidationError{
					LineNum: row.LineNum, RowID: row.ID,
					Message: fmt.Sprintf(
						"activityId=%d  startTime=%q: expected tháng %02d ngày 01 lúc 05:00:00",
						row.ActivityID, row.StartTime, spec.startMonth,
					),
				})
			}
		} else {
			result.Errors = append(result.Errors, ValidationError{
				LineNum: row.LineNum, RowID: row.ID,
				Message: fmt.Sprintf("activityId=%d  startTime=%q không parse được datetime", row.ActivityID, row.StartTime),
			})
		}

		// Check endTime: ngày 1, đúng tháng, 05:00:00
		if et, err := parseDateTime(row.EndTime); err == nil {
			if int(et.Month()) != spec.endMonth || et.Day() != 1 || et.Hour() != 5 || et.Minute() != 0 || et.Second() != 0 {
				result.Errors = append(result.Errors, ValidationError{
					LineNum: row.LineNum, RowID: row.ID,
					Message: fmt.Sprintf(
						"activityId=%d  endTime=%q: expected tháng %02d ngày 01 lúc 05:00:00",
						row.ActivityID, row.EndTime, spec.endMonth,
					),
				})
			}
			if spec.nextYear {
				if st, err2 := parseDateTime(row.StartTime); err2 == nil && et.Year() != st.Year()+1 {
					result.Errors = append(result.Errors, ValidationError{
						LineNum: row.LineNum, RowID: row.ID,
						Message: fmt.Sprintf(
							"activityId=5025  endTime năm %d phải = startTime năm %d + 1",
							et.Year(), st.Year(),
						),
					})
				}
			}
		} else {
			result.Errors = append(result.Errors, ValidationError{
				LineNum: row.LineNum, RowID: row.ID,
				Message: fmt.Sprintf("activityId=%d  endTime=%q không parse được datetime", row.ActivityID, row.EndTime),
			})
		}
	}
	return result
}

// ─── Terminal report ──────────────────────────────────────────────────────────

func printReport(path string, results []RuleResult, warnings []string) {
	bold := func(s string) string { return colorBold + s + colorReset }
	green := func(s string) string { return colorGreen + s + colorReset }
	red := func(s string) string { return colorRed + s + colorReset }
	yellow := func(s string) string { return colorYellow + s + colorReset }

	sep := strings.Repeat("═", 70)
	fmt.Println(bold("╔" + sep + "╗"))
	header := fmt.Sprintf("  VALIDATION REPORT: %s", path)
	pad := 70 - len([]rune(header))
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("%s%s%s%s\n", bold("║"), header, strings.Repeat(" ", pad), bold("║"))
	fmt.Println(bold("╚" + sep + "╝"))
	fmt.Println()

	if len(warnings) > 0 {
		fmt.Println(yellow("⚠  Cảnh báo khi parse:"))
		for _, w := range warnings {
			fmt.Printf("   %s\n", w)
		}
		fmt.Println()
	}

	passCount, failCount := 0, 0
	for _, r := range results {
		errCount := len(r.Errors)
		if errCount == 0 {
			passCount++
			fmt.Printf("  %s  Rule %-2d  %s\n", green("✅"), r.RuleNum, r.Name)
		} else {
			failCount++
			fmt.Printf("  %s  Rule %-2d  %-50s %s\n",
				red("❌"), r.RuleNum, r.Name, red(fmt.Sprintf("(%d lỗi)", errCount)))
			for _, e := range r.Errors {
				fmt.Printf("      %s  Line %-5d (id=%-6d)  %s\n",
					red("└─"), e.LineNum, e.RowID, e.Message)
			}
		}
	}

	fmt.Println()
	fmt.Println(bold(sep))
	summary := fmt.Sprintf("Kết quả: %d/%d PASS   |   %d/%d FAIL",
		passCount, len(results), failCount, len(results))
	if failCount == 0 {
		fmt.Println("  " + green(summary))
	} else {
		fmt.Println("  " + red(summary))
	}
	fmt.Println(bold(sep))
}
