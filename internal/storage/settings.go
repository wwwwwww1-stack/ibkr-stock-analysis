package storage

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ibkr-stock-analysis/internal/domain"

	_ "modernc.org/sqlite"
)

type Snapshot struct {
	Settings domain.Settings         `json:"settings"`
	History  []domain.AnalysisResult `json:"history"`
}

type Store struct {
	path         string
	legacyPath   string
	historyLimit int
}

func NewStore(path string, historyLimit int) *Store {
	return NewStoreWithLegacyPath(path, "", historyLimit)
}

func NewStoreWithLegacyPath(path string, legacyPath string, historyLimit int) *Store {
	if historyLimit <= 0 {
		historyLimit = 50
	}
	return &Store{path: path, legacyPath: legacyPath, historyLimit: historyLimit}
}

func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "ibkr-stock-analysis", "app.db"), nil
}

func LegacyJSONPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "ibkr-stock-analysis", "settings.json"), nil
}

func (s *Store) Load() (Snapshot, error) {
	db, err := s.open()
	if err != nil {
		return Snapshot{}, err
	}
	defer db.Close()

	settings, err := s.loadSettings(db)
	if err != nil {
		return Snapshot{}, err
	}
	history, err := s.latestResults(db)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Settings: settings, History: history}, nil
}

func (s *Store) Save(snapshot Snapshot) error {
	if err := s.SaveSettings(snapshot.Settings); err != nil {
		return err
	}
	for _, result := range snapshot.History {
		if _, err := s.AppendAnalysisResult(result); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SaveSettings(settings domain.Settings) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	return s.saveSettings(db, settings.Normalize())
}

func (s *Store) AppendAnalysisResult(result domain.AnalysisResult) (int64, error) {
	db, err := s.open()
	if err != nil {
		return 0, err
	}
	defer db.Close()

	return s.appendAnalysisResult(db, result)
}

func (s *Store) QueryAnalysisHistory(query domain.AnalysisHistoryQuery) (domain.AnalysisHistoryPage, error) {
	db, err := s.open()
	if err != nil {
		return domain.AnalysisHistoryPage{}, err
	}
	defer db.Close()

	query = s.normalizeQuery(query)
	total, err := s.countHistory(db, query)
	if err != nil {
		return domain.AnalysisHistoryPage{}, err
	}
	records, err := s.queryHistory(db, query, true)
	if err != nil {
		return domain.AnalysisHistoryPage{}, err
	}
	return domain.AnalysisHistoryPage{
		Records: records,
		Total:   total,
		Limit:   query.Limit,
		Offset:  query.Offset,
	}, nil
}

func (s *Store) WriteAnalysisHistoryCSV(w io.Writer, query domain.AnalysisHistoryQuery) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	query = s.normalizeQuery(query)
	records, err := s.queryHistory(db, query, false)
	if err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	if err := writer.Write([]string{
		"updated_at",
		"symbol",
		"timeframe",
		"direction",
		"current_price",
		"entry_low",
		"entry_high",
		"stop_loss",
		"take_profit",
		"risk_reward",
		"confidence",
		"summary",
		"invalidated_if",
		"price_action",
	}); err != nil {
		return err
	}
	for _, record := range records {
		if err := writer.Write(csvRow(record.Result)); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func (s *Store) open() (*sql.DB, error) {
	if strings.TrimSpace(s.path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.ensureSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.migrateLegacyJSON(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func (s *Store) ensureSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			payload TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS analysis_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol TEXT NOT NULL,
			timeframe TEXT NOT NULL,
			direction TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			result_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_results_symbol_updated ON analysis_results(symbol, updated_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_results_timeframe_updated ON analysis_results(timeframe, updated_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_results_direction_updated ON analysis_results(direction, updated_at DESC, id DESC)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) migrateLegacyJSON(db *sql.DB) error {
	if strings.TrimSpace(s.legacyPath) == "" {
		return nil
	}
	var settingsCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&settingsCount); err != nil {
		return err
	}
	if settingsCount > 0 {
		return nil
	}
	data, err := os.ReadFile(s.legacyPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.saveSettingsTx(tx, snapshot.Settings.Normalize()); err != nil {
		return err
	}
	for _, result := range snapshot.History {
		if result.Stale {
			continue
		}
		if _, err := s.appendAnalysisResultTx(tx, result); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) loadSettings(db *sql.DB) (domain.Settings, error) {
	var payload string
	err := db.QueryRow(`SELECT payload FROM settings WHERE id = 1`).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.DefaultSettings(), nil
		}
		return domain.Settings{}, err
	}
	var settings domain.Settings
	if err := json.Unmarshal([]byte(payload), &settings); err != nil {
		return domain.DefaultSettings(), nil
	}
	return settings.Normalize(), nil
}

func (s *Store) saveSettings(db *sql.DB, settings domain.Settings) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.saveSettingsTx(tx, settings.Normalize()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) saveSettingsTx(tx *sql.Tx, settings domain.Settings) error {
	payload, err := json.Marshal(settings.Normalize())
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO settings (id, payload, updated_at)
		 VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET payload = excluded.payload, updated_at = excluded.updated_at`,
		string(payload),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) appendAnalysisResult(db *sql.DB, result domain.AnalysisResult) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	id, err := s.appendAnalysisResultTx(tx, result)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) appendAnalysisResultTx(tx *sql.Tx, result domain.AnalysisResult) (int64, error) {
	result.Symbol = normalizeSymbol(result.Symbol)
	if result.Symbol == "" {
		return 0, fmt.Errorf("analysis result symbol is required")
	}
	if result.UpdatedAt.IsZero() {
		result.UpdatedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return 0, err
	}
	res, err := tx.Exec(
		`INSERT INTO analysis_results (symbol, timeframe, direction, updated_at, result_json)
		 VALUES (?, ?, ?, ?, ?)`,
		result.Symbol,
		string(result.Timeframe),
		string(result.Output.Direction),
		result.UpdatedAt.UTC().Format(time.RFC3339Nano),
		string(payload),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) latestResults(db *sql.DB) ([]domain.AnalysisResult, error) {
	rows, err := db.Query(`
		SELECT result_json
		FROM (
			SELECT result_json,
			       ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY updated_at DESC, id DESC) AS row_num
			FROM analysis_results
		)
		WHERE row_num = 1
		ORDER BY json_extract(result_json, '$.symbol')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.AnalysisResult
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		result, err := decodeResult(payload)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

func (s *Store) countHistory(db *sql.DB, query domain.AnalysisHistoryQuery) (int, error) {
	where, args := historyWhere(query)
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM analysis_results`+where, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Store) queryHistory(db *sql.DB, query domain.AnalysisHistoryQuery, paginate bool) ([]domain.AnalysisHistoryRecord, error) {
	where, args := historyWhere(query)
	statement := `SELECT id, result_json FROM analysis_results` + where + ` ORDER BY updated_at DESC, id DESC`
	if paginate {
		statement += ` LIMIT ? OFFSET ?`
		args = append(args, query.Limit, query.Offset)
	}
	rows, err := db.Query(statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []domain.AnalysisHistoryRecord{}
	for rows.Next() {
		var record domain.AnalysisHistoryRecord
		var payload string
		if err := rows.Scan(&record.ID, &payload); err != nil {
			return nil, err
		}
		result, err := decodeResult(payload)
		if err != nil {
			return nil, err
		}
		record.Result = result
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) normalizeQuery(query domain.AnalysisHistoryQuery) domain.AnalysisHistoryQuery {
	query.Symbol = normalizeSymbol(query.Symbol)
	if query.Limit <= 0 {
		query.Limit = s.historyLimit
	}
	if query.Limit > 500 {
		query.Limit = 500
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query
}

func historyWhere(query domain.AnalysisHistoryQuery) (string, []any) {
	var clauses []string
	var args []any
	if query.Symbol != "" {
		clauses = append(clauses, "symbol = ?")
		args = append(args, query.Symbol)
	}
	if query.Timeframe != "" {
		clauses = append(clauses, "timeframe = ?")
		args = append(args, string(query.Timeframe))
	}
	if query.Direction != "" {
		clauses = append(clauses, "direction = ?")
		args = append(args, string(query.Direction))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func decodeResult(payload string) (domain.AnalysisResult, error) {
	var result domain.AnalysisResult
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return domain.AnalysisResult{}, err
	}
	return result, nil
}

func csvRow(result domain.AnalysisResult) []string {
	output := result.Output
	entryLow := ""
	entryHigh := ""
	if output.EntryZone != nil {
		entryLow = formatFloat(output.EntryZone.Low)
		entryHigh = formatFloat(output.EntryZone.High)
	}
	stopLoss := ""
	if output.StopLoss != nil {
		stopLoss = formatFloat(*output.StopLoss)
	}
	riskReward := ""
	if output.RiskReward != nil {
		riskReward = formatFloat(*output.RiskReward)
	}
	targets := make([]string, 0, len(output.TakeProfit))
	for _, target := range output.TakeProfit {
		targets = append(targets, formatFloat(target))
	}
	return []string{
		result.UpdatedAt.UTC().Format(time.RFC3339),
		result.Symbol,
		string(result.Timeframe),
		string(output.Direction),
		formatFloat(result.CurrentPrice),
		entryLow,
		entryHigh,
		stopLoss,
		strings.Join(targets, ";"),
		riskReward,
		formatFloat(output.Confidence),
		output.Summary,
		output.InvalidatedIf,
		strings.Join(output.PriceAction, ";"),
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
