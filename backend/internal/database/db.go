package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ParticipationRate struct {
	Addr              string
	Moniker           string
	ParticipationRate float64
}

type User struct {
	UserID    string    `gorm:"primaryKey;column:user_id" json:"user_id"`
	Name      string    `gorm:"column:nameuser;not null" json:"name"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}
type HourReport struct {
	UserID            string `gorm:"primaryKey;column:user_id;not null" `
	DailyReportHour   int    `gorm:"column:daily_report_hour;default:9" json:"daily_report_hour"`
	DailyReportMinute int    `gorm:"column:daily_report_minute;default:0" json:"daily_report_minute"`
	Timezone          string `gorm:"column:timezone;default:Europe/Paris" `
}

type AlertContact struct {
	ID          int       `gorm:"primaryKey;autoIncrement;column:id"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UserID      string    `gorm:"column:user_id;not null" `
	Moniker     string    `gorm:"column:moniker;not null" `
	NameContact string    `gorm:"column:namecontact;not null" `
	MentionTag  string    `gorm:"column:mention_tag" `
	IDwebhook   int       `gorm:"column:id_webhook" `
}

type WebhookGovDAO struct {
	ID            int       `gorm:"primaryKey;autoIncrement;column:id" `
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" `
	Description   string    `gorm:"column:description" `
	UserID        string    `gorm:"column:user_id;not null;index:idx_webhooks_govdao_user" `
	URL           string    `gorm:"column:url;not null" `
	Type          string    `gorm:"column:type;not null;check:type IN ('discord','slack')" `
	LastCheckedID int       `gorm:"column:last_checked_id;not null;default:-1" `
}
type WebhookValidator struct {
	ID          int       `gorm:"primaryKey;autoIncrement;column:id"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" `
	Description string    `gorm:"column:description" `
	UserID      string    `gorm:"column:user_id;not null;index:idx_webhooks_validator_user" `
	URL         string    `gorm:"column:url;not null" `
	Type        string    `gorm:"column:type;not null;check:type IN ('discord','slack')" `
}
type DailyParticipation struct {
	Date         time.Time `gorm:"column:date;primaryKey;index:idx_participation_date,priority:1" `
	BlockHeight  int       `gorm:"column:block_height;primaryKey" `
	Moniker      string    `gorm:"column:moniker;primaryKey" `
	Addr         string    `gorm:"column:addr;not null;index:idx_participation_date,priority:2" `
	Participated bool      `gorm:"column:participated;not null" `
}
type AlertLog struct {
	UserID      string    `gorm:"column:user_id;primaryKey" `
	Addr        string    `gorm:"column:addr;primaryKey" `
	Moniker     string    `gorm:"column:moniker;not null" `
	Level       string    `gorm:"column:level;primaryKey" `
	URL         string    `gorm:"column:url;primaryKey" `
	StartHeight int       `gorm:"column:start_height;primaryKey;not null" `
	EndHeight   int       `gorm:"column:end_height;primaryKey;not null" `
	Skipped     bool      `gorm:"column:skipped;not null" `
	Msg         string    `gorm:"column:msg" `
	SentAt      time.Time `gorm:"column:sent_at;autoCreateTime" `
}
type GovDAOState struct {
	ID             int       `gorm:"primaryKey;check:id = 1"`
	LastProposalID int       `gorm:"column:last_proposal_id;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}
type AddrMoniker struct {
	Addr    string `gorm:"column:addr;primaryKey" `
	Moniker string `gorm:"column:moniker;not null" `
}
type AlertSummary struct {
	Moniker     string
	Level       string
	StartHeight int
	EndHeight   int
	Msg         string
	SentAt      time.Time
}

// CReate index
func InitDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("./db/webhooks.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("DB opening error: %v", err)
	}

	// Activate WAL mode
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	_, err = sqlDB.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	_, _ = sqlDB.Exec("PRAGMA synchronous = NORMAL;")
	_, _ = sqlDB.Exec("PRAGMA temp_store = MEMORY;")

	err = db.AutoMigrate(
		&User{}, &AlertContact{}, &WebhookValidator{},
		&WebhookGovDAO{}, &HourReport{}, &GovDAOState{},
		&DailyParticipation{}, &AlertLog{}, &AddrMoniker{},
	)
	if err != nil {
		return nil, err
	}

	InitGovDaoState(db)
	CreateMissingBlocksView(db)

	return db, nil
}

// =================================== VIEW MISSING BLOCK ============================
func CreateMissingBlocksView(db *gorm.DB) error {
	createViewSQL := `
	CREATE VIEW IF NOT EXISTS daily_missing_series AS
	WITH misses AS (
		SELECT
			addr,
			moniker,
			date,
			block_height,
			ROW_NUMBER() OVER (PARTITION BY addr, moniker, date ORDER BY block_height) AS rn
		FROM daily_participations
		WHERE participated = 0
	),
	islands AS (
		SELECT
			addr,
			moniker,
			date,
			block_height,
			block_height - rn AS grp
		FROM misses
	)
	SELECT
		addr,
		moniker,
		date,
		MIN(block_height) AS start_height,
		MAX(block_height) AS end_height,
		COUNT(*) AS missed
	FROM islands
	GROUP BY addr, moniker, date, grp
	HAVING COUNT(*) > 1;
	`

	if err := db.Exec(createViewSQL).Error; err != nil {
		return fmt.Errorf("failed to create view: %w", err)
	}

	log.Println("✅ View `daily_missing_series` created")
	return nil
}

// ===================================State GovDao=====================================
func InitGovDaoState(db *gorm.DB) error {

	state := GovDAOState{
		ID:             1,
		LastProposalID: -1,
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&state).Error

}
func GetLastGovDaoProposalID(db *gorm.DB) (int, error) {
	var state GovDAOState

	err := db.First(&state, "id = ?", 1).Error
	if err != nil {
		return 0, err
	}
	return state.LastProposalID, nil
}
func UpdateLastGovDaoProposalID(db *gorm.DB, newID int) error {
	return db.Model(&GovDAOState{}).
		Where("id = ?", 1).
		Updates(map[string]interface{}{
			"last_proposal_id": newID,
			"updated_at":       gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

// // ==================================== GovDao ======================================
func InsertWebhook(user_id string, url string, description, wtype string, db *gorm.DB) error {
	govdao := WebhookGovDAO{
		UserID:      user_id,
		URL:         url,
		Description: description,
		Type:        wtype,
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&govdao).Error

}

func LoadWebhooks(db *gorm.DB) ([]WebhookGovDAO, error) {
	var webhooks []WebhookGovDAO
	err := db.Find(&webhooks).Error
	return webhooks, err
}
func UpdateLastCheckedID(url string, newID int, db *gorm.DB) error {
	return db.Model(&WebhookGovDAO{}).
		Where("url = ?", url).
		Update("last_checked_id", newID).
		Error
}

func ListWebhooks(db *gorm.DB, userID string) ([]WebhookGovDAO, error) {
	var list []WebhookGovDAO
	err := db.
		Select("id, description, user_id, url, type, last_checked_id").
		Where("user_id = ?", userID).
		Order("id ASC").
		Find(&list).Error

	if err != nil {
		return nil, err
	}
	return list, nil
}

func DeleteWebhook(id int, userID string, db *gorm.DB) error {
	err := db.
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&WebhookGovDAO{}).
		Error

	return err
}

// // ==========================webhooks_validator ===============================================

func InsertMonitoringWebhook(userID, url, description, typ string, db *gorm.DB) error {
	wh := WebhookValidator{
		UserID:      userID,
		URL:         url,
		Description: description,
		Type:        typ,
	}
	return db.Create(&wh).Error
}

func DeleteMonitoringWebhook(id int, userID string, db *gorm.DB) error {
	return db.
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&WebhookValidator{}).
		Error
}

func ListMonitoringWebhooks(db *gorm.DB, userID string) ([]WebhookValidator, error) {
	var result []WebhookValidator
	err := db.
		Select("id, description, user_id, url, type").
		Where("user_id = ?", userID).
		Order("id ASC").
		Find(&result).Error
	return result, err
}

// =============================== gnovalidator y Govdao ======================================
func UpdateMonitoringWebhook(db *gorm.DB, id int, userID, description, newURL, newType, tablename string) error {
	stmt := fmt.Sprintf("UPDATE %s SET url = ?, description = ?, type = ? WHERE user_id = ? AND id = ?", tablename)
	return db.Exec(stmt, newURL, description, newType, userID, id).Error
}

func GetWebhookByID(db *gorm.DB, userID, table string) (*WebhookValidator, error) {
	var wh WebhookValidator
	query := fmt.Sprintf("SELECT user_id, description, url, type FROM %s WHERE user_id = ?", table)
	err := db.Raw(query, userID).Scan(&wh).Error
	if err != nil {
		return nil, err
	}
	return &wh, nil
}

//============================== USERS ===================================================

func InsertUser(userID, email, name string, db *gorm.DB) error {
	u := User{
		UserID: userID,
		Email:  email,
		Name:   name,
	}
	err := db.Create(&u).Error
	if err != nil {
		return err
	}
	return createHourReport(db, userID)
}
func DeleteUser(userID string, db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		tables := []any{
			&WebhookGovDAO{}, &WebhookValidator{},
			&AlertContact{}, &HourReport{},
			&User{},
		}
		for _, model := range tables {
			if err := tx.Where("user_id = ?", userID).Delete(model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func UpdateUser(db *gorm.DB, name, email, userID string) error {
	return db.
		Model(&User{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"nameuser": name,
			"email":    email,
		}).Error
}
func GetUserById(db *gorm.DB, userID string) (*User, error) {
	var usr User
	err := db.
		Select("user_id, nameuser, email").
		Where("user_id = ?", userID).
		First(&usr).Error
	if err != nil {
		return nil, err
	}
	return &usr, nil
}

// ============================== Report Hour =============================================
func UpdateHeureReport(db *gorm.DB, H, M int, T, userID string) error {
	// Validate timezone
	if _, err := time.LoadLocation(T); err != nil {
		log.Printf("Invalid timezone '%s', defaulting to UTC", T)
		T = "UTC"
	}
	return db.
		Model(&HourReport{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"daily_report_hour":   H,
			"daily_report_minute": M,
			"timezone":            T,
		}).Error
}
func GetHourReport(db *gorm.DB, userID string) (*HourReport, error) {
	var hr HourReport
	err := db.
		Select("daily_report_hour, daily_report_minute").
		Where("user_id = ?", userID).
		First(&hr).Error
	if err != nil {
		return nil, err
	}
	return &hr, nil
}
func createHourReport(db *gorm.DB, userID string) error {
	return db.Create(&HourReport{UserID: userID}).Error
}

// ============================== Alert_contact =============================================
func InsertAlertContact(db *gorm.DB, userID, moniker, namecontact, mentionTag string, idwebhook int) error {
	contact := AlertContact{
		UserID:      userID,
		Moniker:     moniker,
		NameContact: namecontact,
		MentionTag:  mentionTag,
		IDwebhook:   idwebhook,
	}
	return db.Create(&contact).Error
}

func ListAlertContacts(db *gorm.DB, userID string) ([]AlertContact, error) {
	var contacts []AlertContact
	err := db.
		Where("user_id = ?", userID).
		Order("id ASC").
		Find(&contacts).Error
	return contacts, err
}

func UpdateAlertContact(db *gorm.DB, id int, userID, moniker, namecontact, mentionTag string, idwebhook int) error {

	return db.
		Model(&AlertContact{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"moniker":     moniker,
			"namecontact": namecontact,
			"mention_tag": mentionTag,
			"id_webhook":  idwebhook,
		}).Error
}
func DeleteAlertContact(db *gorm.DB, id int) error {
	return db.Delete(&AlertContact{}, id).Error
}

// ==================================== Purge ==========================================
func PruneOldParticipationData(db *gorm.DB, keepDays int) error {
	cutoff := time.Now().AddDate(0, 0, -keepDays).Format("2006-01-02")
	res := db.
		Where("date < ?", cutoff).
		Delete(&DailyParticipation{})

	if res.Error != nil {
		return res.Error
	}

	log.Printf("🧹 Pruned %d old rows (before %s)", res.RowsAffected, cutoff)
	return nil
}

// ====================================== ALERT LOG ======================================
func InsertAlertlog(db *gorm.DB, userID, addr, moniker, level, url string, startheight, endheight int, skipped bool, msg string, sent time.Time) error {
	alert := AlertLog{
		UserID:      userID,
		Addr:        addr,
		Moniker:     moniker,
		Level:       level,
		URL:         url,
		StartHeight: startheight,
		EndHeight:   endheight,
		Skipped:     skipped,
		Msg:         msg,
		SentAt:      sent,
	}
	return db.Create(&alert).Error
}

func GetAlertLog(db *gorm.DB) ([]AlertSummary, error) {
	var alerts []AlertSummary
	result := db.
		Model(&AlertLog{}).
		Select("DISTINCT moniker, level, start_height, end_height,msg,sent_at").
		Order("start_height desc").
		Limit(10).
		Scan(&alerts)

	return alerts, result.Error
}

func GetCurrentPeriodParticipationRate(db *gorm.DB, period string) ([]ParticipationRate, error) {
	log.Println("==========Start Get Participate Rate ")
	var results []ParticipationRate

	var start, end time.Time
	now := time.Now()

	switch period {
	case "current_week":
		today := time.Now()
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday => 7
		}
		start = today.AddDate(0, 0, -weekday+1) // Reculer jusqu'au lundi
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
		end = start.AddDate(0, 0, 7)
	case "current_month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 1, 0)

	case "current_year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(1, 0, 0)

	default:
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")
	log.Printf("start %s", startStr)
	log.Printf("end %s", endStr)
	query := fmt.Sprintf(`
		SELECT
			addr,
			moniker,
			ROUND(SUM(participated) * 100.0 / COUNT(*), 1) AS participation_rate
		FROM
			daily_participations
		WHERE
			date >= '%s' AND date < '%s'
		GROUP BY
			addr, moniker
		ORDER BY
			participation_rate DESC;
	`, startStr, endStr)

	err := db.Raw(query).Scan(&results).Error

	return results, err
}

// ====================================== ADDR MONIKER =============================

func InsertAddrMoniker(db *gorm.DB, addr, moniker string) error {
	addrmoniker := AddrMoniker{

		Addr:    addr,
		Moniker: moniker,
	}
	return db.Create(&addrmoniker).Error
}
func GetMoniker(db *gorm.DB) (map[string]string, error) {
	var entries []AddrMoniker
	result := db.Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}

	monikerMap := make(map[string]string)
	for _, e := range entries {
		monikerMap[e.Addr] = e.Moniker
		log.Printf("📦 Loaded from DB — Addr: %s, Moniker: %s", e.Addr, e.Moniker)
	}
	log.Printf("✅ Loaded %d monikers from DB", len(monikerMap))
	return monikerMap, nil
}
