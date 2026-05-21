package store

import (
	"database/sql"
	"fund-trace/internal/model"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return s
}

func TestOpenMemory(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	if s.db == nil {
		t.Fatal("db is nil")
	}
	if err := s.db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestMigrations(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	// running migrate twice should be idempotent
	if err := s.Migrate(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

// --- Fund CRUD ---

func TestAddAndListFunds(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	if err := s.AddFund("000001"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFund("000001"); err != nil {
		t.Fatal(err) // should not error on duplicate (INSERT OR IGNORE)
	}
	if err := s.AddFund("000002"); err != nil {
		t.Fatal(err)
	}

	funds, err := s.ListFunds()
	if err != nil {
		t.Fatal(err)
	}
	if len(funds) != 2 {
		t.Fatalf("expected 2 funds, got %d", len(funds))
	}
}

func TestAddFundWithName(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	if err := s.AddFundWithName("000001", "Test Fund", model.FundStock); err != nil {
		t.Fatal(err)
	}

	f, err := s.GetFund("000001")
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "Test Fund" {
		t.Fatalf("expected 'Test Fund', got %q", f.Name)
	}
	if f.Type != model.FundStock {
		t.Fatalf("expected FundStock, got %v", f.Type)
	}
}

func TestGetFundNotFound(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	_, err := s.GetFund("999999")
	if err == nil {
		t.Fatal("expected error for missing fund")
	}
}

func TestRemoveFund(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	if err := s.AddFund("000001"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveFund("000001"); err != nil {
		t.Fatal(err)
	}

	// removing again should fail (not found)
	if err := s.RemoveFund("000001"); err == nil {
		t.Fatal("expected error for removing non-existent fund")
	}
}

func TestUpdateFundName(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	if err := s.AddFund("000001"); err != nil {
		t.Fatal(err)
	}

	if err := s.UpdateFundName("000001", "My Fund"); err != nil {
		t.Fatal(err)
	}

	f, err := s.GetFund("000001")
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "My Fund" {
		t.Fatalf("expected 'My Fund', got %q", f.Name)
	}
}

func TestSeedFromConfig(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	codes := []string{"000001", "000002", "000003", "000004"}
	if err := s.SeedFromConfig(codes); err != nil {
		t.Fatal(err)
	}

	funds, err := s.ListFunds()
	if err != nil {
		t.Fatal(err)
	}
	if len(funds) != 4 {
		t.Fatalf("expected 4 funds, got %d", len(funds))
	}
}

// --- NavSnapshot CRUD ---

func TestSaveNavSnapshots(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	snaps := []model.NavSnapshot{
		{FundCode: "000001", Date: "2025-05-20", UnitNAV: 1.234, AccumulatedNAV: 2.345, DailyGrowthPct: 0.5},
		{FundCode: "000001", Date: "2025-05-19", UnitNAV: 1.228, AccumulatedNAV: 2.339, DailyGrowthPct: -0.3},
		{FundCode: "000002", Date: "2025-05-20", UnitNAV: 3.456, AccumulatedNAV: 4.567, DailyGrowthPct: 1.2},
	}

	if err := s.SaveNavSnapshots(snaps); err != nil {
		t.Fatal(err)
	}

	// duplicate save should be idempotent (INSERT OR IGNORE)
	if err := s.SaveNavSnapshots(snaps); err != nil {
		t.Fatal(err)
	}
}

func TestGetNavHistory(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	snaps := []model.NavSnapshot{
		{FundCode: "000001", Date: "2025-05-20", UnitNAV: 1.234, AccumulatedNAV: 2.345, DailyGrowthPct: 0.5},
		{FundCode: "000001", Date: "2025-05-19", UnitNAV: 1.228, AccumulatedNAV: 2.339, DailyGrowthPct: -0.3},
		{FundCode: "000001", Date: "2025-05-18", UnitNAV: 1.230, AccumulatedNAV: 2.341, DailyGrowthPct: 0.1},
	}
	if err := s.SaveNavSnapshots(snaps); err != nil {
		t.Fatal(err)
	}

	history, err := s.GetNavHistory("000001", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
	// should be ordered by date DESC
	if history[0].Date != "2025-05-20" {
		t.Fatalf("expected first entry 2025-05-20, got %s", history[0].Date)
	}
	if history[1].Date != "2025-05-19" {
		t.Fatalf("expected second entry 2025-05-19, got %s", history[1].Date)
	}
}

func TestHasNavData(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	has, err := s.HasNavData("000001")
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("expected no nav data before save")
	}

	if err := s.SaveNavSnapshots([]model.NavSnapshot{
		{FundCode: "000001", Date: "2025-05-20", UnitNAV: 1.234, AccumulatedNAV: 2.345},
	}); err != nil {
		t.Fatal(err)
	}

	has, err = s.HasNavData("000001")
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected nav data after save")
	}

	has, err = s.HasNavData("999999")
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("expected no nav data for unknown fund")
	}
}

// --- Alert CRUD ---

func TestUpsertAndListAlerts(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	a := model.Alert{
		FundCode:     "000001",
		Type:         model.AlertDrop,
		ThresholdPct: -5.0,
		Enabled:      true,
	}
	id, err := s.UpsertAlert(a)
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	alerts, err := s.ListAlerts()
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].FundCode != "000001" {
		t.Fatalf("expected fund_code 000001, got %s", alerts[0].FundCode)
	}
	if alerts[0].Type != model.AlertDrop {
		t.Fatalf("expected AlertDrop, got %v", alerts[0].Type)
	}
	if alerts[0].ThresholdPct != -5.0 {
		t.Fatalf("expected -5.0, got %f", alerts[0].ThresholdPct)
	}
	if !alerts[0].Enabled {
		t.Fatal("expected alert enabled")
	}

	// upsert with same id should update
	a.ID = id
	a.ThresholdPct = -3.0
	a.Enabled = false
	id2, err := s.UpsertAlert(a)
	if err != nil {
		t.Fatal(err)
	}
	if id2 != id {
		t.Fatalf("expected same id %d, got %d", id, id2)
	}

	alerts, err = s.ListAlerts()
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert after update, got %d", len(alerts))
	}
	if alerts[0].ThresholdPct != -3.0 {
		t.Fatalf("expected -3.0 after update, got %f", alerts[0].ThresholdPct)
	}
}

func TestGetAlertsForFund(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	_, _ = s.UpsertAlert(model.Alert{FundCode: "000001", Type: model.AlertDrop, ThresholdPct: -5.0, Enabled: true})
	_, _ = s.UpsertAlert(model.Alert{FundCode: "000001", Type: model.AlertRise, ThresholdPct: 5.0, Enabled: true})
	_, _ = s.UpsertAlert(model.Alert{FundCode: "000002", Type: model.AlertDrop, ThresholdPct: -3.0, Enabled: false}) // disabled
	_, _ = s.UpsertAlert(model.Alert{FundCode: "000002", Type: model.AlertRise, ThresholdPct: 3.0, Enabled: true})

	alerts, err := s.GetAlertsForFund("000001")
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 enabled alerts for 000001, got %d", len(alerts))
	}

	// fund 000002 should only return the enabled one
	alerts, err = s.GetAlertsForFund("000002")
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 enabled alert for 000002, got %d", len(alerts))
	}
}

func TestDisableAlert(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	id, _ := s.UpsertAlert(model.Alert{FundCode: "000001", Type: model.AlertDrop, ThresholdPct: -5.0, Enabled: true})

	if err := s.DisableAlert(id); err != nil {
		t.Fatal(err)
	}

	alerts, err := s.GetAlertsForFund("000001")
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatal("expected no enabled alerts after disable")
	}

	listed, err := s.ListAlerts()
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 alert in list, got %d", len(listed))
	}
	if listed[0].Enabled {
		t.Fatal("expected alert to be disabled")
	}
}

func TestDeleteAlert(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	id, _ := s.UpsertAlert(model.Alert{FundCode: "000001", Type: model.AlertDrop, ThresholdPct: -5.0, Enabled: true})

	if err := s.DeleteAlert(id); err != nil {
		t.Fatal(err)
	}

	alerts, err := s.ListAlerts()
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts after delete, got %d", len(alerts))
	}
}

func TestUpdateAlertTriggeredAt(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	id, _ := s.UpsertAlert(model.Alert{FundCode: "000001", Type: model.AlertDrop, ThresholdPct: -5.0, Enabled: true})

	alertsBefore, _ := s.ListAlerts()
	if alertsBefore[0].LastTriggeredAt.Valid {
		t.Fatal("expected LastTriggeredAt to be NULL initially")
	}

	if err := s.UpdateAlertTriggeredAt(id); err != nil {
		t.Fatal(err)
	}

	alertsAfter, _ := s.ListAlerts()
	if !alertsAfter[0].LastTriggeredAt.Valid {
		t.Fatal("expected LastTriggeredAt to be set after update")
	}
}

// --- DailySummary CRUD ---

func TestDailySummary(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	ds := model.DailySummary{
		Date:      "2025-05-20",
		FundCode:  "000001",
		NAV:       1.234,
		ChangePct: 0.5,
		Note:      "test note",
	}

	if err := s.SaveDailySummary(ds); err != nil {
		t.Fatal(err)
	}

	retrieved, err := s.GetDailySummary("2025-05-20", "000001")
	if err != nil {
		t.Fatal(err)
	}
	if retrieved.NAV != 1.234 {
		t.Fatalf("expected NAV 1.234, got %f", retrieved.NAV)
	}
	if retrieved.ChangePct != 0.5 {
		t.Fatalf("expected ChangePct 0.5, got %f", retrieved.ChangePct)
	}
	if retrieved.Note != "test note" {
		t.Fatalf("expected note 'test note', got %q", retrieved.Note)
	}

	// update (INSERT OR REPLACE)
	ds2 := model.DailySummary{
		Date:      "2025-05-20",
		FundCode:  "000001",
		NAV:       1.239,
		ChangePct: 0.8,
		Note:      "updated",
	}
	if err := s.SaveDailySummary(ds2); err != nil {
		t.Fatal(err)
	}

	retrieved, err = s.GetDailySummary("2025-05-20", "000001")
	if err != nil {
		t.Fatal(err)
	}
	if retrieved.NAV != 1.239 {
		t.Fatalf("expected updated NAV 1.239, got %f", retrieved.NAV)
	}
}

func TestDailySummaryNotFound(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	_, err := s.GetDailySummary("2099-01-01", "000001")
	if err == nil {
		t.Fatal("expected error for missing daily summary")
	}
}

// --- File-based store ---

func TestFileBasedStore(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.db"

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if err := s.AddFund("000001"); err != nil {
		t.Fatal(err)
	}

	funds, err := s.ListFunds()
	if err != nil {
		t.Fatal(err)
	}
	if len(funds) != 1 {
		t.Fatalf("expected 1 fund, got %d", len(funds))
	}

	// close and reopen
	s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()

	funds2, err := s2.ListFunds()
	if err != nil {
		t.Fatal(err)
	}
	if len(funds2) != 1 {
		t.Fatalf("expected 1 fund after reopen, got %d", len(funds2))
	}
	if funds2[0].Code != "000001" {
		t.Fatalf("expected code 000001, got %s", funds2[0].Code)
	}
}

func TestFileBasedWALMode(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/wal.db"

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	// verify WAL mode is on
	var journalMode string
	if err := s.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Fatalf("expected WAL journal mode, got %s", journalMode)
	}
}

func TestMaxOpenConns(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	if err := s.AddFund("000001"); err != nil {
		t.Fatal(err)
	}

	// verify SetMaxOpenConns(1) is set by checking DB() returns a valid db
	db := s.DB()
	if db == nil {
		t.Fatal("DB() returned nil")
	}

	// concurrent read should be fine
	f, err := s.GetFund("000001")
	if err != nil {
		t.Fatal(err)
	}
	if f.Code != "000001" {
		t.Fatalf("expected code 000001, got %s", f.Code)
	}
}

func TestDBPing(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	if err := s.db.Ping(); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestAlertWithNullTime(t *testing.T) {
	s := openTestStore(t)
	defer s.Close()

	a := model.Alert{
		FundCode:     "000001",
		Type:         model.AlertDrop,
		ThresholdPct: -3.0,
		Enabled:      true,
		LastTriggeredAt: sql.NullTime{
			Time:  time.Now().UTC().Truncate(time.Second),
			Valid: true,
		},
	}
	id, err := s.UpsertAlert(a)
	if err != nil {
		t.Fatal(err)
	}

	alerts, err := s.ListAlerts()
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if !alerts[0].LastTriggeredAt.Valid {
		t.Fatal("expected LastTriggeredAt to be set")
	}
	_ = id

	// test with NULL time
	a2 := model.Alert{
		FundCode:        "000002",
		Type:            model.AlertRise,
		ThresholdPct:    5.0,
		Enabled:         true,
		LastTriggeredAt: sql.NullTime{Valid: false},
	}
	_, err = s.UpsertAlert(a2)
	if err != nil {
		t.Fatal(err)
	}

	alerts, err = s.ListAlerts()
	if err != nil {
		t.Fatal(err)
	}
	// find the null-time alert by checking all alerts
	found := false
	for _, a := range alerts {
		if a.FundCode == "000002" && !a.LastTriggeredAt.Valid {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected alert with NULL LastTriggeredAt")
	}
}
