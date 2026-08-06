package main

import (
	"database/sql"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

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
	// Diapers
	WetCount         int
	DirtyCount       int
	BothCount        int
	// Baths
	BathCount        int
	// Milestones & Events
	Milestones       []string
	Events           []string
	UpcomingEvents   []string
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
	}

	// Diapers
	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log WHERE diaper = 'wet' AND log_date BETWEEN $1 AND $2`, from, to).Scan(&r.WetCount)
	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log WHERE diaper = 'dirty' AND log_date BETWEEN $1 AND $2`, from, to).Scan(&r.DirtyCount)
	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log WHERE diaper = 'both' AND log_date BETWEEN $1 AND $2`, from, to).Scan(&r.BothCount)

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
	sb.WriteString("\n")

	sb.WriteString("🚼 Diapers\n")
	total := r.WetCount + r.DirtyCount + r.BothCount
	sb.WriteString(fmt.Sprintf("  Total: %d (wet: %d, dirty: %d, both: %d)\n", total, r.WetCount, r.DirtyCount, r.BothCount))
	sb.WriteString(fmt.Sprintf("  Daily avg: %.1f\n\n", float64(total)/7))

	sb.WriteString(fmt.Sprintf("🛁 Baths: %d\n\n", r.BathCount))

	if len(r.Milestones) > 0 {
		sb.WriteString("🎉 Milestones\n")
		for _, m := range r.Milestones {
			sb.WriteString(fmt.Sprintf("  • %s\n", m))
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

