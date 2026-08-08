package main

import (
	"database/sql"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// WHO girls weight-for-age: index = month, [median, -2SD, +2SD] in grams
var whoGirlsWeight = [][3]int{
	{3232, 2400, 4300}, {4176, 3200, 5400}, {5128, 3900, 6600}, {5990, 4600, 7600}, {6694, 5100, 8400},
	{7268, 5600, 9000}, {7775, 6000, 9500}, {8218, 6300, 10000}, {8613, 6600, 10400}, {8986, 6900, 10900},
	{9325, 7100, 11200}, {9637, 7300, 11500}, {9924, 7500, 11800}, {10189, 7700, 12100}, {10436, 7900, 12400},
	{10668, 8100, 12700}, {10887, 8300, 12900}, {11095, 8400, 13200}, {11294, 8600, 13400}, {11486, 8700, 13600},
	{11671, 8900, 13900}, {11851, 9000, 14100}, {12027, 9100, 14300}, {12199, 9300, 14500}, {12368, 9400, 14700},
	{12534, 9500, 14900}, {12697, 9700, 15100}, {12858, 9800, 15300}, {13017, 9900, 15500}, {13174, 10000, 15700},
	{13329, 10100, 15900}, {13482, 10200, 16100}, {13634, 10300, 16300}, {13784, 10400, 16500}, {13932, 10500, 16700},
	{14079, 10600, 16900}, {14224, 10700, 17100}, {14368, 10800, 17300}, {14510, 10900, 17500}, {14651, 11000, 17700},
	{14790, 11100, 17900}, {14928, 11200, 18100}, {15065, 11300, 18300}, {15200, 11400, 18500}, {15334, 11500, 18700},
	{15467, 11600, 18900}, {15599, 11700, 19100}, {15730, 11800, 19300}, {15860, 11900, 19500}, {15989, 12000, 19700},
	{16117, 12100, 19900}, {16244, 12200, 20100}, {16370, 12300, 20300}, {16495, 12400, 20500}, {16619, 12500, 20700},
	{16742, 12600, 20900}, {16864, 12700, 21100}, {16985, 12800, 21300}, {17105, 12900, 21500}, {17224, 13000, 21700},
	{17342, 13100, 21900},
}

func whoWeightStatus(weightG int, ageMonths int) string {
	if ageMonths < 0 || ageMonths > 60 {
		return ""
	}
	ref := whoGirlsWeight[ageMonths]
	median, minus2, plus2 := ref[0], ref[1], ref[2]
	switch {
	case weightG < minus2:
		return fmt.Sprintf("⚠️ Below -2SD (underweight) — WHO median: %dg, -2SD: %dg", median, minus2)
	case weightG <= median:
		return fmt.Sprintf("✅ Normal, below median — WHO median: %dg, -2SD: %dg", median, minus2)
	case weightG <= plus2:
		return fmt.Sprintf("✅ Normal, above median — WHO median: %dg, +2SD: %dg", median, plus2)
	default:
		return fmt.Sprintf("⚠️ Above +2SD (overweight) — WHO median: %dg, +2SD: %dg", median, plus2)
	}
}

type weeklyReport struct {
	From             string
	To               string
	AgeWeeks         int
	AgeDays          int
	// Feeding
	TotalMilkG       int
	FeedCount        int
	BreastCount      int
	BottleCount      int
	AvgMilkPerFeedG  int
	LongestFeedGapH  float64
	// Sleep
	TotalSleepMin    int
	TotalAwakeMin    int
	LongestSleepMin  int
	AvgSleepSessionMin int
	BedtimesPerDay   []string
	// Weight
	LatestWeight     *int
	WeightGainG      *int
	WHOStatus        string
	// Diapers
	WetCount         int
	DirtyCount       int
	BothCount        int
	DaysWithoutDirty int
	// Baths
	BathCount        int
	// Milestones & Events
	Milestones       []string
	Events           []string
	UpcomingEvents   []string
	NewWords         []string
	WordOfTheWeek    string
}

func buildWeeklyReport() (*weeklyReport, error) {
	bp := budapest()
	now := time.Now().In(bp)
	to := now.Format("2006-01-02")
	from := now.AddDate(0, 0, -6).Format("2006-01-02")

	r := &weeklyReport{From: from, To: to}

	// Age
	var birthDateStr string
	db.QueryRow(`SELECT value FROM app_settings WHERE key = 'birth-date'`).Scan(&birthDateStr)
	if birthDateStr != "" {
		birthDate, err := time.ParseInLocation("2006-01-02", birthDateStr, bp)
		if err == nil {
			days := int(now.Sub(birthDate).Hours() / 24)
			r.AgeWeeks = days / 7
			r.AgeDays = days % 7
		}
	}

	// Feeding
	rows, err := db.Query(`
		SELECT
			COALESCE(SUM(post_feed_weight_g - pre_feed_weight_g), 0),
			COUNT(*),
			SUM(CASE WHEN fed_breast THEN 1 ELSE 0 END),
			SUM(CASE WHEN fed_bottle THEN 1 ELSE 0 END)
		FROM zili_daily_log
		WHERE pre_feed_weight_g IS NOT NULL
		  AND post_feed_weight_g IS NOT NULL
		  AND post_feed_weight_g > pre_feed_weight_g
		  AND log_date BETWEEN $1 AND $2
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&r.TotalMilkG, &r.FeedCount, &r.BreastCount, &r.BottleCount)
	}
	if r.FeedCount > 0 {
		r.AvgMilkPerFeedG = r.TotalMilkG / r.FeedCount
	}

	// Longest gap between feedings
	feedRows, err := db.Query(`
		SELECT log_date::text, log_time::text FROM zili_daily_log
		WHERE pre_feed_weight_g IS NOT NULL
		  AND post_feed_weight_g IS NOT NULL
		  AND post_feed_weight_g > pre_feed_weight_g
		  AND log_date BETWEEN $1 AND $2
		  AND log_time IS NOT NULL
		ORDER BY log_date, log_time
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer feedRows.Close()
	var feedTimes []time.Time
	for feedRows.Next() {
		var d, t string
		feedRows.Scan(&d, &t)
		if len(d) > 10 { d = d[:10] }
		if len(t) > 5 { t = t[:5] }
		ft, err := time.ParseInLocation("2006-01-02 15:04", d+" "+t, bp)
		if err == nil {
			feedTimes = append(feedTimes, ft)
		}
	}
	for i := 1; i < len(feedTimes); i++ {
		gap := feedTimes[i].Sub(feedTimes[i-1]).Hours()
		if gap > r.LongestFeedGapH {
			r.LongestFeedGapH = gap
		}
	}

	// Sleep
	sleepRows, err := db.Query(`
		SELECT log_date::text, log_time::text, sleep_event
		FROM zili_daily_log
		WHERE log_time IS NOT NULL
		  AND sleep_event IS NOT NULL
		  AND log_date BETWEEN $1::date - INTERVAL '1 day' AND $2::date
		ORDER BY log_date, log_time
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer sleepRows.Close()
	var sleepLogs []sleepLog
	for sleepRows.Next() {
		var l sleepLog
		var event string
		sleepRows.Scan(&l.Date, &l.Time, &event)
		if len(l.Date) > 10 { l.Date = l.Date[:10] }
		if len(l.Time) > 5 { l.Time = l.Time[:5] }
		if event == "fell_asleep" { l.Summary = "elaludt" }
		if event == "woke_up" { l.Summary = "ébredt" }
		sleepLogs = append(sleepLogs, l)
	}
	sleepResult := calcSleepAwake(sleepLogs, from, to, now)
	for _, d := range sleepResult {
		r.TotalSleepMin += d.SleepMin
		r.TotalAwakeMin += d.AwakeMin
	}

	// Longest sleep stretch & avg session & bedtimes
	var sleepStart *time.Time
	var sessionCount int
	bedtimeMap := map[string]string{}
	toTime := func(date, t string) time.Time {
		dt, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+t, bp)
		return dt
	}
	for _, l := range sleepLogs {
		if l.Summary == "elaludt" && sleepStart == nil {
			t := toTime(l.Date, l.Time)
			sleepStart = &t
			bedtimeMap[l.Date] = l.Time
		} else if l.Summary == "ébredt" && sleepStart != nil {
			wt := toTime(l.Date, l.Time)
			dur := int(wt.Sub(*sleepStart).Minutes())
			if dur > r.LongestSleepMin {
				r.LongestSleepMin = dur
			}
			r.TotalSleepMin += 0 // already counted above
			sessionCount++
			sleepStart = nil
		}
	}
	if sessionCount > 0 {
		r.AvgSleepSessionMin = r.TotalSleepMin / sessionCount
	}
	for date, t := range bedtimeMap {
		r.BedtimesPerDay = append(r.BedtimesPerDay, fmt.Sprintf("%s: %s", date, t))
	}

	// Weight
	var latestWeight sql.NullInt64
	db.QueryRow(`
		SELECT measurement_weight_g FROM zili_daily_log
		WHERE measurement_weight_g IS NOT NULL AND log_date BETWEEN $1 AND $2
		ORDER BY log_date DESC, id DESC LIMIT 1
	`, from, to).Scan(&latestWeight)
	if latestWeight.Valid {
		v := int(latestWeight.Int64)
		r.LatestWeight = &v
		var prevWeight sql.NullInt64
		db.QueryRow(`
			SELECT measurement_weight_g FROM zili_daily_log
			WHERE measurement_weight_g IS NOT NULL AND log_date < $1
			ORDER BY log_date DESC, id DESC LIMIT 1
		`, from).Scan(&prevWeight)
		if prevWeight.Valid {
			gain := v - int(prevWeight.Int64)
			r.WeightGainG = &gain
		}
		ageMonths := (r.AgeWeeks * 7 + r.AgeDays) * 10 / 304
		r.WHOStatus = whoWeightStatus(v, ageMonths)
	}

	// Diapers
	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log WHERE diaper = 'wet' AND log_date BETWEEN $1 AND $2`, from, to).Scan(&r.WetCount)
	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log WHERE diaper = 'dirty' AND log_date BETWEEN $1 AND $2`, from, to).Scan(&r.DirtyCount)
	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log WHERE diaper = 'both' AND log_date BETWEEN $1 AND $2`, from, to).Scan(&r.BothCount)

	// Consecutive days without dirty diaper
	dirtyRows, err := db.Query(`
		SELECT DISTINCT log_date::text FROM zili_daily_log
		WHERE diaper IN ('dirty', 'both')
		ORDER BY log_date DESC LIMIT 60
	`)
	if err == nil {
		dirtyDays := map[string]bool{}
		for dirtyRows.Next() {
			var d string
			dirtyRows.Scan(&d)
			if len(d) > 10 { d = d[:10] }
			dirtyDays[d] = true
		}
		dirtyRows.Close()
		for i := 0; i < 60; i++ {
			day := now.AddDate(0, 0, -i).Format("2006-01-02")
			if dirtyDays[day] {
				break
			}
			r.DaysWithoutDirty++
		}
	}

	// Baths
	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log WHERE bathed = true AND log_date BETWEEN $1 AND $2`, from, to).Scan(&r.BathCount)

	// Milestones
	mRows, err := db.Query(`
		SELECT daily_summary FROM zili_daily_log
		WHERE milestone = true AND log_date BETWEEN $1 AND $2
		ORDER BY log_date, log_time
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer mRows.Close()
	for mRows.Next() {
		var s string
		mRows.Scan(&s)
		r.Milestones = append(r.Milestones, s)
	}

	// Events this week
	eRows, err := db.Query(`
		SELECT title, event_date::text FROM zili_events
		WHERE event_date BETWEEN $1 AND $2
		ORDER BY event_date
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer eRows.Close()
	for eRows.Next() {
		var title, date string
		eRows.Scan(&title, &date)
		if len(date) > 10 { date = date[:10] }
		r.Events = append(r.Events, fmt.Sprintf("%s — %s", date, title))
	}

	// Upcoming events next week
	nextFrom := now.AddDate(0, 0, 1).Format("2006-01-02")
	nextTo := now.AddDate(0, 0, 7).Format("2006-01-02")
	nRows, err := db.Query(`
		SELECT title, event_date::text FROM zili_events
		WHERE event_date BETWEEN $1 AND $2
		ORDER BY event_date
	`, nextFrom, nextTo)
	if err != nil {
		return nil, err
	}
	defer nRows.Close()
	for nRows.Next() {
		var title, date string
		nRows.Scan(&title, &date)
		if len(date) > 10 { date = date[:10] }
		r.UpcomingEvents = append(r.UpcomingEvents, fmt.Sprintf("%s — %s", date, title))
	}

	// Words
	wRows, err := db.Query(`SELECT word FROM zili_words WHERE noted_date BETWEEN $1 AND $2 ORDER BY noted_date, id`, from, to)
	if err != nil {
		return nil, err
	}
	defer wRows.Close()
	for wRows.Next() {
		var w string
		wRows.Scan(&w)
		r.NewWords = append(r.NewWords, w)
	}
	if len(r.NewWords) > 0 {
		r.WordOfTheWeek = r.NewWords[len(r.NewWords)-1]
	} else {
		db.QueryRow(`SELECT word FROM zili_words ORDER BY RANDOM() LIMIT 1`).Scan(&r.WordOfTheWeek)
	}

	return r, nil
}

func formatReport(r *weeklyReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Zili weekly report: %s – %s\n", r.From, r.To))
	sb.WriteString(fmt.Sprintf("Age: %d weeks %d days\n\n", r.AgeWeeks, r.AgeDays))

	sb.WriteString("🍼 Feeding\n")
	sb.WriteString(fmt.Sprintf("  Total milk: %d g\n", r.TotalMilkG))
	sb.WriteString(fmt.Sprintf("  Feedings: %d (breast: %d, bottle: %d)\n", r.FeedCount, r.BreastCount, r.BottleCount))
	sb.WriteString(fmt.Sprintf("  Avg per feeding: %d g\n", r.AvgMilkPerFeedG))
	if r.LongestFeedGapH > 0 {
		sb.WriteString(fmt.Sprintf("  Longest gap between feedings: %.1fh\n", r.LongestFeedGapH))
	}
	sb.WriteString("\n")

	sb.WriteString("😴 Sleep\n")
	sb.WriteString(fmt.Sprintf("  Total sleep: %dh %dm\n", r.TotalSleepMin/60, r.TotalSleepMin%60))
	sb.WriteString(fmt.Sprintf("  Total awake: %dh %dm\n", r.TotalAwakeMin/60, r.TotalAwakeMin%60))
	if r.LongestSleepMin > 0 {
		sb.WriteString(fmt.Sprintf("  Longest stretch: %dh %dm\n", r.LongestSleepMin/60, r.LongestSleepMin%60))
	}
	if r.AvgSleepSessionMin > 0 {
		sb.WriteString(fmt.Sprintf("  Avg session: %dh %dm\n", r.AvgSleepSessionMin/60, r.AvgSleepSessionMin%60))
	}
	if len(r.BedtimesPerDay) > 0 {
		sb.WriteString("  Bedtimes:\n")
		for _, b := range r.BedtimesPerDay {
			sb.WriteString(fmt.Sprintf("    • %s\n", b))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("⚖️ Weight\n")
	if r.LatestWeight != nil {
		sb.WriteString(fmt.Sprintf("  Latest: %d g\n", *r.LatestWeight))
	}
	if r.WeightGainG != nil {
		sb.WriteString(fmt.Sprintf("  Gain since last measurement: %+d g\n", *r.WeightGainG))
	}
	if r.WHOStatus != "" {
		sb.WriteString(fmt.Sprintf("  WHO standard: %s\n", r.WHOStatus))
	}
	sb.WriteString("\n")

	sb.WriteString("🚼 Diapers\n")
	total := r.WetCount + r.DirtyCount + r.BothCount
	sb.WriteString(fmt.Sprintf("  Total: %d (wet: %d, dirty: %d, both: %d)\n", total, r.WetCount, r.DirtyCount, r.BothCount))
	sb.WriteString(fmt.Sprintf("  Daily avg: %.1f\n", float64(total)/7))
	if r.DaysWithoutDirty >= 1 {
		sb.WriteString(fmt.Sprintf("  ⚠️ %d day(s) without dirty diaper!\n", r.DaysWithoutDirty))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("🛁 Baths: %d\n\n", r.BathCount))

	if len(r.Milestones) > 0 {
		sb.WriteString("🎉 Milestones\n")
		for _, m := range r.Milestones {
			sb.WriteString(fmt.Sprintf("  • %s\n", m))
		}
		sb.WriteString("\n")
	}

	if r.WordOfTheWeek != "" {
		sb.WriteString("💬 Word of the week: " + r.WordOfTheWeek + "\n")
		if len(r.NewWords) > 1 {
			sb.WriteString("  New words this week:\n")
			for _, w := range r.NewWords {
				sb.WriteString(fmt.Sprintf("  • %s\n", w))
			}
		}
		sb.WriteString("\n")
	}

	if len(r.Events) > 0 {
		sb.WriteString("📅 Events this week\n")
		for _, e := range r.Events {
			sb.WriteString(fmt.Sprintf("  • %s\n", e))
		}
		sb.WriteString("\n")
	}

	if len(r.UpcomingEvents) > 0 {
		sb.WriteString("🔜 Upcoming next week\n")
		for _, e := range r.UpcomingEvents {
			sb.WriteString(fmt.Sprintf("  • %s\n", e))
		}
	}

	return sb.String()
}

func sendWeeklyReport() error {
	r, err := buildWeeklyReport()
	if err != nil {
		return err
	}

	from := getEnv("SMTP_FROM", "")
	toRaw := getEnv("SMTP_TO", "")
	var toList []string
	for _, addr := range strings.Split(toRaw, ",") {
		if t := strings.TrimSpace(addr); t != "" {
			toList = append(toList, t)
		}
	}
	password := getEnv("SMTP_PASSWORD", "")
	host := getEnv("SMTP_HOST", "smtp.gmail.com")
	smtpAddr := host + ":587"

	subject := fmt.Sprintf("Zili weekly report %s – %s", r.From, r.To)
	body := formatReport(r)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, strings.Join(toList, ", "), subject, body)

	auth := smtp.PlainAuth("", from, password, host)
	return smtp.SendMail(smtpAddr, auth, from, toList, []byte(msg))
}

