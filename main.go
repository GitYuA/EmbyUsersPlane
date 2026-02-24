package main

import (
    "bufio"
    "bytes"
    "crypto/md5"
    crand "crypto/rand"
    "crypto/subtle"
    "crypto/tls"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "html/template"
    "io"
    "log"
    "mime"
    "net"
    "net/http"
    "net/smtp"
    "net/url"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "sync"
    "sync/atomic"
    "time"
    "unicode/utf8"

    "golang.org/x/crypto/bcrypt"
    _ "modernc.org/sqlite"
)

const (
    appDataDirDefault = "/data"
    appDefaultTZ      = "Asia/Shanghai"
    appPermanentDate  = "2099-12-31"

    statusEnabled  = "已启用"
    statusDisabled = "已禁用"
    statusAdmin    = "管理员"

    sessionCookieName = "emby_panel_sid"
)

type ServerConfig struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    URL  string `json:"url"`
    Key  string `json:"key"`
}

type ServerSettings struct {
    CheckTime           *string `json:"checkTime,omitempty"`
    AutoTaskEnabled     *bool   `json:"autoTaskEnabled,omitempty"`
    LogRetentionDays    *int    `json:"logRetentionDays,omitempty"`
    ExpireAction        *string `json:"expireAction,omitempty"`
    RestoreTemplateUser *string `json:"restoreTemplateUser,omitempty"`
    DefaultTemplateUser *string `json:"defaultTemplateUser,omitempty"`

    SMTPHost   *string `json:"smtp_host,omitempty"`
    SMTPPort   *int    `json:"smtp_port,omitempty"`
    SMTPUser   *string `json:"smtp_user,omitempty"`
    SMTPPass   *string `json:"smtp_pass,omitempty"`
    SMTPFrom   *string `json:"smtp_from,omitempty"`
    SMTPSecure *string `json:"smtp_secure,omitempty"`

    NotifyBeforeDays  *int  `json:"notify_before_days,omitempty"`
    NotifyOnOperation *bool `json:"notify_on_operation,omitempty"`
}

type Config struct {
    Servers          []ServerConfig             `json:"servers"`
    CurrentServerID  string                     `json:"currentServerId"`
    ExpireField      string                     `json:"expireField"`
    PermanentDate    string                     `json:"permanentDate"`
    CheckTime        string                     `json:"checkTime"`
    LogRetentionDays int                        `json:"logRetentionDays"`
    PanelPass        string                     `json:"panelPass"`
    NotifyOnOp       bool                       `json:"notify_on_operation"`
    QueryAPIFallback bool                       `json:"query_api_fallback"`
    QueryRequireTok  bool                       `json:"query_require_token"`
    QueryToken       string                     `json:"query_token"`
    LogFile          string                     `json:"logFile"`
    HiddenUsers      any                        `json:"hiddenUsers,omitempty"`
    ServerSettings   map[string]ServerSettings  `json:"server_settings,omitempty"`

    SMTPHost   string `json:"smtp_host,omitempty"`
    SMTPPort   int    `json:"smtp_port,omitempty"`
    SMTPUser   string `json:"smtp_user,omitempty"`
    SMTPPass   string `json:"smtp_pass,omitempty"`
    SMTPFrom   string `json:"smtp_from,omitempty"`
    SMTPSecure string `json:"smtp_secure,omitempty"`

    NotifyBeforeDays int    `json:"notify_before_days,omitempty"`
    ExpireAction     string `json:"expireAction,omitempty"`

    RestoreTemplateUser string `json:"restoreTemplateUser,omitempty"`
    AutoTaskEnabled     *bool  `json:"autoTaskEnabled,omitempty"`
}

type EffectiveSettings struct {
    CheckTime           string
    AutoTaskEnabled     bool
    LogRetentionDays    int
    ExpireAction        string
    RestoreTemplateUser string
    DefaultTemplateUser string

    SMTPHost   string
    SMTPPort   int
    SMTPUser   string
    SMTPPass   string
    SMTPFrom   string
    SMTPSecure string

    NotifyBeforeDays  int
    NotifyOnOperation bool
}

type ServerContext struct {
    Config        *Config
    Server        *ServerConfig
    ServerID      string
    ServerName    string
    EmbyServerURL string
    APIKey        string

    DataDir string
    UsersDir string
    LogDir string

    UserServerDir string
    LogServerDir  string

    DBFile    string
    DataFile  string
    LogFile   string
    CacheFile string

    Settings EffectiveSettings
}

type ConfigStore struct {
    path    string
    dataDir string
    mu      sync.Mutex
}

func newConfigStore(dataDir string) *ConfigStore {
    return &ConfigStore{
        path:    filepath.Join(dataDir, "config.json"),
        dataDir: dataDir,
    }
}

func defaultConfig(dataDir string) *Config {
    return &Config{
        Servers:          []ServerConfig{},
        CurrentServerID:  "",
        ExpireField:      "RemoteClientBitrateLimit",
        PermanentDate:    appPermanentDate,
        CheckTime:        "00:00",
        LogRetentionDays: 30,
        PanelPass:        "",
        NotifyOnOp:       true,
        QueryAPIFallback: false,
        QueryRequireTok:  false,
        QueryToken:       "",
        LogFile:          filepath.Join(dataDir, "operation_log.txt"),
        ServerSettings:   map[string]ServerSettings{},
        SMTPSecure:       "ssl",
        NotifyBeforeDays: 3,
        ExpireAction:     "disable",
    }
}

func (s *ConfigStore) loadUnlocked() (*Config, error) {
    cfg := defaultConfig(s.dataDir)

    b, err := os.ReadFile(s.path)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return cfg, nil
        }
        return nil, err
    }

    if len(bytes.TrimSpace(b)) == 0 {
        return cfg, nil
    }

    if err := json.Unmarshal(b, cfg); err != nil {
        return nil, err
    }

    if cfg.ServerSettings == nil {
        cfg.ServerSettings = map[string]ServerSettings{}
    }
    if cfg.PermanentDate == "" {
        cfg.PermanentDate = appPermanentDate
    }
    if cfg.CheckTime == "" {
        cfg.CheckTime = "00:00"
    }
    if cfg.LogRetentionDays < 1 {
        cfg.LogRetentionDays = 30
    }
    if cfg.SMTPSecure == "" {
        cfg.SMTPSecure = "ssl"
    }
    if cfg.NotifyBeforeDays < 0 {
        cfg.NotifyBeforeDays = 3
    }
    if cfg.ExpireAction == "" {
        cfg.ExpireAction = "disable"
    }

    return cfg, nil
}

func (s *ConfigStore) saveUnlocked(cfg *Config) error {
    if err := os.MkdirAll(filepath.Dir(s.path), 0o775); err != nil {
        return err
    }

    b, err := json.MarshalIndent(cfg, "", "    ")
    if err != nil {
        return err
    }

    tmp := s.path + ".tmp"
    if err := os.WriteFile(tmp, b, 0o644); err != nil {
        return err
    }
    return os.Rename(tmp, s.path)
}

func (s *ConfigStore) Load() (*Config, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.loadUnlocked()
}

func (s *ConfigStore) Save(cfg *Config) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.saveUnlocked(cfg)
}

func (s *ConfigStore) Modify(fn func(cfg *Config) error) (*Config, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    cfg, err := s.loadUnlocked()
    if err != nil {
        return nil, err
    }

    if err := fn(cfg); err != nil {
        return nil, err
    }

    if err := s.saveUnlocked(cfg); err != nil {
        return nil, err
    }

    return cfg, nil
}

func normalizeHiddenUsers(raw any) map[string][]string {
    out := map[string][]string{}
    if raw == nil {
        return out
    }

    switch v := raw.(type) {
    case map[string][]string:
        for k, arr := range v {
            out[k] = append([]string{}, arr...)
        }
    case map[string]any:
        for k, vv := range v {
            switch arr := vv.(type) {
            case []any:
                items := make([]string, 0, len(arr))
                for _, item := range arr {
                    s, ok := item.(string)
                    if !ok {
                        continue
                    }
                    s = strings.TrimSpace(s)
                    if s != "" {
                        items = append(items, s)
                    }
                }
                out[k] = items
            case []string:
                items := make([]string, 0, len(arr))
                for _, item := range arr {
                    item = strings.TrimSpace(item)
                    if item != "" {
                        items = append(items, item)
                    }
                }
                out[k] = items
            }
        }
    case []any:
        items := make([]string, 0, len(v))
        for _, item := range v {
            s, ok := item.(string)
            if !ok {
                continue
            }
            s = strings.TrimSpace(s)
            if s != "" {
                items = append(items, s)
            }
        }
        out[""] = items
    case []string:
        items := make([]string, 0, len(v))
        for _, item := range v {
            item = strings.TrimSpace(item)
            if item != "" {
                items = append(items, item)
            }
        }
        out[""] = items
    }

    return out
}

func mergeSettings(cfg *Config, serverID string) EffectiveSettings {
    logRetention := cfg.LogRetentionDays
    if logRetention < 1 {
        logRetention = 30
    }
    notifyBefore := cfg.NotifyBeforeDays
    if notifyBefore < 0 {
        notifyBefore = 3
    }

    settings := EffectiveSettings{
        CheckTime:           fallback(cfg.CheckTime, "00:00"),
        AutoTaskEnabled:     true,
        LogRetentionDays:    logRetention,
        ExpireAction:        fallback(cfg.ExpireAction, "disable"),
        RestoreTemplateUser: cfg.RestoreTemplateUser,
        DefaultTemplateUser: "",

        SMTPHost:   cfg.SMTPHost,
        SMTPPort:   cfg.SMTPPort,
        SMTPUser:   cfg.SMTPUser,
        SMTPPass:   cfg.SMTPPass,
        SMTPFrom:   cfg.SMTPFrom,
        SMTPSecure: fallback(cfg.SMTPSecure, "ssl"),

        NotifyBeforeDays:  notifyBefore,
        NotifyOnOperation: cfg.NotifyOnOp,
    }

    if cfg.AutoTaskEnabled != nil {
        settings.AutoTaskEnabled = *cfg.AutoTaskEnabled
    }

    if ss, ok := cfg.ServerSettings[serverID]; ok {
        if ss.CheckTime != nil {
            settings.CheckTime = fallback(*ss.CheckTime, "00:00")
        }
        if ss.AutoTaskEnabled != nil {
            settings.AutoTaskEnabled = *ss.AutoTaskEnabled
        }
        if ss.LogRetentionDays != nil {
            if *ss.LogRetentionDays >= 1 {
                settings.LogRetentionDays = *ss.LogRetentionDays
            }
        }
        if ss.ExpireAction != nil {
            ea := strings.TrimSpace(*ss.ExpireAction)
            if ea == "delete" || ea == "disable" {
                settings.ExpireAction = ea
            }
        }
        if ss.RestoreTemplateUser != nil {
            settings.RestoreTemplateUser = strings.TrimSpace(*ss.RestoreTemplateUser)
        }
        if ss.DefaultTemplateUser != nil {
            settings.DefaultTemplateUser = strings.TrimSpace(*ss.DefaultTemplateUser)
        }

        if ss.SMTPHost != nil {
            settings.SMTPHost = strings.TrimSpace(*ss.SMTPHost)
        }
        if ss.SMTPPort != nil {
            settings.SMTPPort = *ss.SMTPPort
        }
        if ss.SMTPUser != nil {
            settings.SMTPUser = strings.TrimSpace(*ss.SMTPUser)
        }
        if ss.SMTPPass != nil {
            settings.SMTPPass = *ss.SMTPPass
        }
        if ss.SMTPFrom != nil {
            settings.SMTPFrom = strings.TrimSpace(*ss.SMTPFrom)
        }
        if ss.SMTPSecure != nil {
            settings.SMTPSecure = fallback(strings.TrimSpace(*ss.SMTPSecure), "ssl")
        }

        if ss.NotifyBeforeDays != nil {
            settings.NotifyBeforeDays = maxInt(*ss.NotifyBeforeDays, 0)
        }
        if ss.NotifyOnOperation != nil {
            settings.NotifyOnOperation = *ss.NotifyOnOperation
        }
    }

    if settings.LogRetentionDays < 1 {
        settings.LogRetentionDays = 30
    }
    if settings.SMTPSecure == "" {
        settings.SMTPSecure = "ssl"
    }

    return settings
}

type Session struct {
    ID            string
    CSRFToken     string
    IsLoggedIn    bool
    LoginFailures []time.Time
    LastSeen      time.Time
}

type SessionStore struct {
    mu       sync.Mutex
    sessions map[string]*Session
}

func newSessionStore() *SessionStore {
    return &SessionStore{
        sessions: map[string]*Session{},
    }
}

func (s *SessionStore) cleanupLocked(now time.Time) {
    cutoff := now.Add(-24 * time.Hour)
    for id, sess := range s.sessions {
        if sess.LastSeen.Before(cutoff) {
            delete(s.sessions, id)
        }
    }
}

func (s *SessionStore) setCookie(w http.ResponseWriter, r *http.Request, id string) {
    secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
    http.SetCookie(w, &http.Cookie{
        Name:     sessionCookieName,
        Value:    id,
        Path:     "/",
        HttpOnly: true,
        Secure:   secure,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   0,
    })
}

func (s *SessionStore) GetOrCreate(w http.ResponseWriter, r *http.Request) *Session {
    s.mu.Lock()
    defer s.mu.Unlock()

    now := time.Now()
    s.cleanupLocked(now)

    if cookie, err := r.Cookie(sessionCookieName); err == nil {
        if sess, ok := s.sessions[cookie.Value]; ok {
            sess.LastSeen = now
            return sess
        }
    }

    id := randomHex(24)
    sess := &Session{
        ID:        id,
        CSRFToken: randomHex(32),
        LastSeen:  now,
    }
    s.sessions[id] = sess
    s.setCookie(w, r, id)
    return sess
}

func (s *SessionStore) Regenerate(w http.ResponseWriter, r *http.Request, old *Session) *Session {
    s.mu.Lock()
    defer s.mu.Unlock()

    now := time.Now()
    s.cleanupLocked(now)

    delete(s.sessions, old.ID)

    id := randomHex(24)
    sess := &Session{
        ID:            id,
        CSRFToken:     old.CSRFToken,
        IsLoggedIn:    old.IsLoggedIn,
        LoginFailures: append([]time.Time{}, old.LoginFailures...),
        LastSeen:      now,
    }

    s.sessions[id] = sess
    s.setCookie(w, r, id)
    return sess
}

type ChargeRecord struct {
    Date string `json:"date"`
    Days int    `json:"days"`
    Note string `json:"note"`
}

type LocalUser struct {
    ID             string         `json:"id,omitempty"`
    Name           string         `json:"name"`
    OpenDate       string         `json:"openDate"`
    LastRecharge   string         `json:"lastRecharge"`
    ExpireDate     string         `json:"expireDate"`
    DaysLeft       string         `json:"daysLeft"`
    Status         string         `json:"status"`
    Group          string         `json:"group"`
    Email          string         `json:"email"`
    LastNotifyDate string         `json:"lastNotifyDate"`
    ChargeHistory  []ChargeRecord `json:"chargeHistory"`

    LastActivityDate string `json:"lastActivityDate,omitempty"`
    Disabled         bool   `json:"disabled,omitempty"`
    IsAdministrator  bool   `json:"isAdministrator,omitempty"`
    SortDays         int    `json:"sortDays,omitempty"`
}

type DBManager struct {
    mu  sync.Mutex
    dbs map[string]*sql.DB
}

func newDBManager() *DBManager {
    return &DBManager{dbs: map[string]*sql.DB{}}
}

func (m *DBManager) get(file string) (*sql.DB, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if db, ok := m.dbs[file]; ok {
        return db, nil
    }

    if err := os.MkdirAll(filepath.Dir(file), 0o775); err != nil {
        return nil, err
    }

    db, err := sql.Open("sqlite", file)
    if err != nil {
        return nil, err
    }

    if _, err = db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
        db.Close()
        return nil, err
    }
    if _, err = db.Exec(`PRAGMA synchronous = NORMAL;`); err != nil {
        db.Close()
        return nil, err
    }

    schema := `CREATE TABLE IF NOT EXISTS users (
        id TEXT PRIMARY KEY,
        name TEXT,
        open_date TEXT,
        last_recharge TEXT,
        expire_date TEXT,
        days_left TEXT,
        status TEXT,
        group_name TEXT,
        email TEXT,
        last_notify_date TEXT,
        charge_history TEXT
    )`
    if _, err = db.Exec(schema); err != nil {
        db.Close()
        return nil, err
    }

    m.dbs[file] = db
    return db, nil
}

func (m *DBManager) Get(file, id string) (*LocalUser, error) {
    db, err := m.get(file)
    if err != nil {
        return nil, err
    }

    row := db.QueryRow(`SELECT id, name, open_date, last_recharge, expire_date, days_left, status, group_name, email, last_notify_date, charge_history FROM users WHERE id = ?`, id)
    user, err := scanUserRow(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return user, nil
}

func (m *DBManager) FindByName(file, name string) (*LocalUser, error) {
    db, err := m.get(file)
    if err != nil {
        return nil, err
    }

    row := db.QueryRow(`SELECT id, name, open_date, last_recharge, expire_date, days_left, status, group_name, email, last_notify_date, charge_history FROM users WHERE name = ? COLLATE NOCASE LIMIT 1`, name)
    user, err := scanUserRow(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return user, nil
}

func (m *DBManager) GetAll(file string) (map[string]LocalUser, error) {
    db, err := m.get(file)
    if err != nil {
        return nil, err
    }

    rows, err := db.Query(`SELECT id, name, open_date, last_recharge, expire_date, days_left, status, group_name, email, last_notify_date, charge_history FROM users`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    out := map[string]LocalUser{}
    for rows.Next() {
        var (
            id             string
            name           sql.NullString
            openDate       sql.NullString
            lastRecharge   sql.NullString
            expireDate     sql.NullString
            daysLeft       sql.NullString
            status         sql.NullString
            groupName      sql.NullString
            email          sql.NullString
            lastNotifyDate sql.NullString
            chargeHistory  sql.NullString
        )

        if err := rows.Scan(&id, &name, &openDate, &lastRecharge, &expireDate, &daysLeft, &status, &groupName, &email, &lastNotifyDate, &chargeHistory); err != nil {
            return nil, err
        }

        user := LocalUser{
            ID:             id,
            Name:           name.String,
            OpenDate:       openDate.String,
            LastRecharge:   lastRecharge.String,
            ExpireDate:     expireDate.String,
            DaysLeft:       daysLeft.String,
            Status:         status.String,
            Group:          groupName.String,
            Email:          email.String,
            LastNotifyDate: lastNotifyDate.String,
            ChargeHistory:  []ChargeRecord{},
        }
        if chargeHistory.String != "" {
            _ = json.Unmarshal([]byte(chargeHistory.String), &user.ChargeHistory)
        }
        out[id] = user
    }

    return out, nil
}

func (m *DBManager) Save(file, id string, user LocalUser) error {
    db, err := m.get(file)
    if err != nil {
        return err
    }

    chargeHistory, _ := json.Marshal(user.ChargeHistory)

    _, err = db.Exec(`INSERT INTO users (id, name, open_date, last_recharge, expire_date, days_left, status, group_name, email, last_notify_date, charge_history)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            name = excluded.name,
            open_date = excluded.open_date,
            last_recharge = excluded.last_recharge,
            expire_date = excluded.expire_date,
            days_left = excluded.days_left,
            status = excluded.status,
            group_name = excluded.group_name,
            email = excluded.email,
            last_notify_date = excluded.last_notify_date,
            charge_history = excluded.charge_history`,
        id,
        user.Name,
        user.OpenDate,
        user.LastRecharge,
        user.ExpireDate,
        user.DaysLeft,
        user.Status,
        user.Group,
        user.Email,
        user.LastNotifyDate,
        string(chargeHistory),
    )

    return err
}

func (m *DBManager) Delete(file, id string) error {
    db, err := m.get(file)
    if err != nil {
        return err
    }
    _, err = db.Exec(`DELETE FROM users WHERE id = ?`, id)
    return err
}

func scanUserRow(row *sql.Row) (*LocalUser, error) {
    var (
        id             string
        name           sql.NullString
        openDate       sql.NullString
        lastRecharge   sql.NullString
        expireDate     sql.NullString
        daysLeft       sql.NullString
        status         sql.NullString
        groupName      sql.NullString
        email          sql.NullString
        lastNotifyDate sql.NullString
        chargeHistory  sql.NullString
    )

    if err := row.Scan(&id, &name, &openDate, &lastRecharge, &expireDate, &daysLeft, &status, &groupName, &email, &lastNotifyDate, &chargeHistory); err != nil {
        return nil, err
    }

    user := &LocalUser{
        ID:             id,
        Name:           name.String,
        OpenDate:       openDate.String,
        LastRecharge:   lastRecharge.String,
        ExpireDate:     expireDate.String,
        DaysLeft:       daysLeft.String,
        Status:         status.String,
        Group:          groupName.String,
        Email:          email.String,
        LastNotifyDate: lastNotifyDate.String,
        ChargeHistory:  []ChargeRecord{},
    }

    if chargeHistory.String != "" {
        _ = json.Unmarshal([]byte(chargeHistory.String), &user.ChargeHistory)
    }

    return user, nil
}

type EmbyUser struct {
    ID               string         `json:"Id"`
    Name             string         `json:"Name"`
    DateCreated      string         `json:"DateCreated"`
    LastActivityDate string         `json:"LastActivityDate"`
    Policy           map[string]any `json:"Policy"`
    Configuration    map[string]any `json:"Configuration"`
}

type AutoCheckSummary struct {
    Checked  int
    Notify   int
    Disabled int
    Deleted  int
    Errors   int
    Message  string
}

type App struct {
    dataDir    string
    projectDir string

    cfgStore *ConfigStore
    sessions *SessionStore
    db       *DBManager

    loginTpl     *template.Template
    dashboardTpl *template.Template

    httpClient *http.Client

    autoCheckRunning atomic.Bool

    logMu sync.Mutex
}

type loginPageData struct {
    CSRFToken    string
    AssetVersion string
}

type dashboardPageData struct {
    CSRFToken    string
    AppConfigJSON template.JS
    UsersJSON    template.JS
    AssetVersion string
}

func newApp(projectDir string) (*App, error) {
    dataDir := os.Getenv("APP_DATA_DIR")
    if strings.TrimSpace(dataDir) == "" {
        dataDir = appDataDirDefault
    }

    if err := os.MkdirAll(filepath.Join(dataDir, "log"), 0o775); err != nil {
        return nil, err
    }
    if err := os.MkdirAll(filepath.Join(dataDir, "users"), 0o775); err != nil {
        return nil, err
    }
    if err := os.MkdirAll(filepath.Join(dataDir, "rate_limit"), 0o775); err != nil {
        return nil, err
    }

    loginTpl, err := template.ParseFiles(filepath.Join(projectDir, "templates", "login.html"))
    if err != nil {
        return nil, err
    }
    dashboardTpl, err := template.ParseFiles(filepath.Join(projectDir, "templates", "dashboard.html"))
    if err != nil {
        return nil, err
    }

    app := &App{
        dataDir:       dataDir,
        projectDir:    projectDir,
        cfgStore:      newConfigStore(dataDir),
        sessions:      newSessionStore(),
        db:            newDBManager(),
        loginTpl:      loginTpl,
        dashboardTpl:  dashboardTpl,
        httpClient:    &http.Client{Timeout: 12 * time.Second},
    }

    return app, nil
}

func main() {
    initTimezone()

    projectDir, err := os.Getwd()
    if err != nil {
        log.Fatalf("failed to get cwd: %v", err)
    }

    app, err := newApp(projectDir)
    if err != nil {
        log.Fatalf("failed to init app: %v", err)
    }

    app.startAutoCheckScheduler()

    adminMux := http.NewServeMux()
    app.registerAdminRoutes(adminMux)

    queryMux := http.NewServeMux()
    app.registerQueryRoutes(queryMux)

    go func() {
        srv := &http.Server{
            Addr:              ":8086",
            Handler:           adminMux,
            ReadHeaderTimeout: 10 * time.Second,
            ReadTimeout:       30 * time.Second,
            WriteTimeout:      120 * time.Second,
            IdleTimeout:       120 * time.Second,
        }
        log.Printf("admin server listening on :8086")
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Fatalf("admin server error: %v", err)
        }
    }()

    querySrv := &http.Server{
        Addr:              ":8085",
        Handler:           queryMux,
        ReadHeaderTimeout: 10 * time.Second,
        ReadTimeout:       30 * time.Second,
        WriteTimeout:      120 * time.Second,
        IdleTimeout:       120 * time.Second,
    }
    log.Printf("query server listening on :8085")
    if err := querySrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        log.Fatalf("query server error: %v", err)
    }
}

func (a *App) registerAdminRoutes(mux *http.ServeMux) {
    assets := http.FileServer(http.Dir(filepath.Join(a.projectDir, "public", "assets")))
    mux.Handle("/assets/", a.secure(http.StripPrefix("/assets/", assets)))

    mux.HandleFunc("/", a.secureFunc(a.handleAdminRoot))
    mux.HandleFunc("/index.php", a.secureFunc(a.handleAdminRoot))
}

func (a *App) registerQueryRoutes(mux *http.ServeMux) {
    assets := http.FileServer(http.Dir(filepath.Join(a.projectDir, "public", "assets")))
    mux.Handle("/assets/", a.secure(http.StripPrefix("/assets/", assets)))

    mux.HandleFunc("/", a.secureFunc(a.handleQueryRoot))
    mux.HandleFunc("/user/user.html", a.secureFunc(a.handleUserPage))
    mux.HandleFunc("/query.php", a.secureFunc(a.handleUserQuery))
    mux.HandleFunc("/user/query.php", a.secureFunc(a.handleUserQuery))
    mux.HandleFunc("/index.php", a.secureFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "forbidden", http.StatusForbidden)
    }))
}

func (a *App) secure(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        addSecurityHeaders(w)
        next.ServeHTTP(w, r)
    })
}

func (a *App) secureFunc(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        addSecurityHeaders(w)
        next(w, r)
    }
}

func (a *App) handleAdminRoot(w http.ResponseWriter, r *http.Request) {
    session := a.sessions.GetOrCreate(w, r)
    if session.CSRFToken == "" {
        session.CSRFToken = randomHex(32)
    }

    cfg, err := a.cfgStore.Load()
    if err != nil {
        http.Error(w, "config load failed", http.StatusInternalServerError)
        return
    }

    switch r.Method {
    case http.MethodGet:
        action := strings.TrimSpace(r.URL.Query().Get("action"))
        if action == "backup" || action == "download_log" {
            if !a.checkAuth(cfg, session) {
                _ = a.renderLoginPage(w, session.CSRFToken)
                return
            }
            if !secureCompare(session.CSRFToken, r.URL.Query().Get("token")) {
                http.Error(w, "CSRF validation failed", http.StatusForbidden)
                return
            }

            ctx, err := a.buildServerContext(cfg, cfg.CurrentServerID)
            if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }

            if action == "backup" {
                if err := a.downloadBackup(w, ctx); err != nil {
                    http.Error(w, err.Error(), http.StatusInternalServerError)
                }
                return
            }

            fileName := r.URL.Query().Get("file")
            tail := r.URL.Query().Get("tail") == "1"
            if err := a.downloadLogFile(w, ctx, fileName, tail, 128*1024); err != nil {
                status := http.StatusInternalServerError
                if strings.Contains(strings.ToLower(err.Error()), "not found") {
                    status = http.StatusNotFound
                }
                http.Error(w, err.Error(), status)
            }
            return
        }

        if !a.checkAuth(cfg, session) {
            _ = a.renderLoginPage(w, session.CSRFToken)
            return
        }

        ctx, err := a.buildServerContext(cfg, cfg.CurrentServerID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        users, err := a.getUserList(ctx)
        if err != nil {
            users = []LocalUser{}
        }

        safeCfg := a.buildSafeFrontendConfig(cfg, ctx)
        appCfgJSON := mustJSON(safeCfg)
        usersJSON := mustJSON(users)

        if err := a.renderDashboardPage(w, dashboardPageData{
            CSRFToken:     session.CSRFToken,
            AppConfigJSON: template.JS(appCfgJSON),
            UsersJSON:     template.JS(usersJSON),
            AssetVersion:  strconv.FormatInt(time.Now().Unix(), 10),
        }); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }

    case http.MethodPost:
        if err := parseRequestForm(r); err != nil {
            writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "请求格式错误"})
            return
        }

        token := r.Header.Get("X-CSRF-Token")
        if !secureCompare(session.CSRFToken, token) {
            writeJSON(w, http.StatusForbidden, map[string]any{"success": false, "message": "CSRF 验证失败，请刷新页面重试"})
            return
        }

        action := strings.TrimSpace(r.FormValue("action"))
        if action == "" {
            writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "缺少 action"})
            return
        }

        if action == "login" {
            res, code := a.ajaxLogin(w, r, session, cfg, r.FormValue("password"))
            writeJSON(w, code, res)
            return
        }

        if !a.checkAuth(cfg, session) {
            writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false, "message": "未登录或会话已过期", "needLogin": true})
            return
        }

        cfg, err = a.cfgStore.Load()
        if err != nil {
            writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "message": "配置读取失败"})
            return
        }

        ctx, err := a.buildServerContext(cfg, r.FormValue("serverId"))
        if err != nil {
            writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "message": err.Error()})
            return
        }

        res, code := a.dispatchAdminAction(w, r, action, cfg, ctx, session)
        writeJSON(w, code, res)

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func (a *App) renderLoginPage(w http.ResponseWriter, csrf string) error {
    return a.loginTpl.Execute(w, loginPageData{
        CSRFToken:    csrf,
        AssetVersion: strconv.FormatInt(time.Now().Unix(), 10),
    })
}

func (a *App) renderDashboardPage(w http.ResponseWriter, data dashboardPageData) error {
    return a.dashboardTpl.Execute(w, data)
}

func (a *App) checkAuth(cfg *Config, session *Session) bool {
    if strings.TrimSpace(cfg.PanelPass) == "" {
        return true
    }
    return session.IsLoggedIn
}

func (a *App) buildServerContext(cfg *Config, preferredServerID string) (*ServerContext, error) {
    usersDir := filepath.Join(a.dataDir, "users")
    logDir := filepath.Join(a.dataDir, "log")
    if err := os.MkdirAll(usersDir, 0o775); err != nil {
        return nil, err
    }
    if err := os.MkdirAll(logDir, 0o775); err != nil {
        return nil, err
    }

    ctx := &ServerContext{
        Config:  cfg,
        DataDir: a.dataDir,
        UsersDir: usersDir,
        LogDir: logDir,
    }

    pickID := strings.TrimSpace(preferredServerID)
    if pickID == "" {
        pickID = strings.TrimSpace(cfg.CurrentServerID)
    }

    var target *ServerConfig
    if len(cfg.Servers) > 0 {
        for i := range cfg.Servers {
            if cfg.Servers[i].ID == pickID {
                target = &cfg.Servers[i]
                break
            }
        }
        if target == nil {
            target = &cfg.Servers[0]
        }
    }

    if target == nil {
        ctx.ServerID = ""
        ctx.ServerName = "default"
        ctx.EmbyServerURL = ""
        ctx.APIKey = ""

        ctx.UserServerDir = filepath.Join(usersDir, "default")
        ctx.LogServerDir = filepath.Join(logDir, "default")
        _ = os.MkdirAll(ctx.UserServerDir, 0o775)
        _ = os.MkdirAll(ctx.LogServerDir, 0o775)

        ctx.DBFile = filepath.Join(ctx.UserServerDir, "users.db")
        ctx.DataFile = filepath.Join(ctx.UserServerDir, "users.json")
        ctx.LogFile = filepath.Join(ctx.LogServerDir, "operation_log.txt")
        ctx.CacheFile = filepath.Join(ctx.UserServerDir, "emby_users_cache_default.json")
        ctx.Settings = mergeSettings(cfg, "")
        return ctx, nil
    }

    ctx.Server = target
    ctx.ServerID = target.ID
    ctx.ServerName = target.Name
    ctx.EmbyServerURL = strings.TrimSpace(target.URL)
    ctx.APIKey = strings.TrimSpace(target.Key)

    safeName := safeFileName(target.Name)
    ctx.UserServerDir = filepath.Join(usersDir, safeName)
    ctx.LogServerDir = filepath.Join(logDir, safeName)
    if err := os.MkdirAll(ctx.UserServerDir, 0o775); err != nil {
        return nil, err
    }
    if err := os.MkdirAll(ctx.LogServerDir, 0o775); err != nil {
        return nil, err
    }

    ctx.DBFile = filepath.Join(ctx.UserServerDir, "users.db")
    ctx.DataFile = filepath.Join(ctx.UserServerDir, "users.json")
    ctx.LogFile = filepath.Join(ctx.LogServerDir, "operation_log.txt")
    ctx.CacheFile = filepath.Join(ctx.UserServerDir, "emby_users_cache_"+md5Hex(ctx.EmbyServerURL)+".json")
    ctx.Settings = mergeSettings(cfg, target.ID)

    return ctx, nil
}

func (a *App) buildSafeFrontendConfig(cfg *Config, ctx *ServerContext) map[string]any {
    safeServers := make([]map[string]any, 0, len(cfg.Servers))
    for _, s := range cfg.Servers {
        item := map[string]any{
            "id":   s.ID,
            "name": s.Name,
            "url":  s.URL,
            "key":  s.Key,
        }
        if strings.TrimSpace(s.Key) != "" {
            item["key"] = "******"
        }
        safeServers = append(safeServers, item)
    }

    safeServerSettings := map[string]map[string]any{}
    for sid, ss := range cfg.ServerSettings {
        m := serverSettingsToMap(ss)
        if v, ok := m["smtp_pass"]; ok {
            if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
                m["smtp_pass"] = "******"
            }
        }
        safeServerSettings[sid] = m
    }

    smtpPass := ""
    if strings.TrimSpace(ctx.Settings.SMTPPass) != "" {
        smtpPass = "******"
    }

    panelPass := ""
    if strings.TrimSpace(cfg.PanelPass) != "" {
        panelPass = "******"
    }

    return map[string]any{
        "servers":       safeServers,
        "currentId":     ctx.ServerID,
        "permanentDate": fallback(cfg.PermanentDate, appPermanentDate),
        "panelPass":     panelPass,
        "notify": map[string]any{
            "before_days": ctx.Settings.NotifyBeforeDays,
        },
        "smtp": map[string]any{
            "host":   ctx.Settings.SMTPHost,
            "port":   ctx.Settings.SMTPPort,
            "user":   ctx.Settings.SMTPUser,
            "pass":   smtpPass,
            "from":   ctx.Settings.SMTPFrom,
            "secure": fallback(ctx.Settings.SMTPSecure, "ssl"),
        },
        "hiddenUsers": normalizeHiddenUsers(cfg.HiddenUsers),
        "serverSettings": safeServerSettings,
        "globalDefaults": map[string]any{
            "checkTime":            ctx.Settings.CheckTime,
            "logRetentionDays":     ctx.Settings.LogRetentionDays,
            "restoreTemplateUser":  ctx.Settings.RestoreTemplateUser,
            "notify_on_operation":  ctx.Settings.NotifyOnOperation,
            "autoTaskEnabled":      true,
        },
    }
}

func serverSettingsToMap(ss ServerSettings) map[string]any {
    m := map[string]any{}
    if ss.CheckTime != nil {
        m["checkTime"] = *ss.CheckTime
    }
    if ss.AutoTaskEnabled != nil {
        m["autoTaskEnabled"] = *ss.AutoTaskEnabled
    }
    if ss.LogRetentionDays != nil {
        m["logRetentionDays"] = *ss.LogRetentionDays
    }
    if ss.ExpireAction != nil {
        m["expireAction"] = *ss.ExpireAction
    }
    if ss.RestoreTemplateUser != nil {
        m["restoreTemplateUser"] = *ss.RestoreTemplateUser
    }
    if ss.DefaultTemplateUser != nil {
        m["defaultTemplateUser"] = *ss.DefaultTemplateUser
    }

    if ss.SMTPHost != nil {
        m["smtp_host"] = *ss.SMTPHost
    }
    if ss.SMTPPort != nil {
        m["smtp_port"] = *ss.SMTPPort
    }
    if ss.SMTPUser != nil {
        m["smtp_user"] = *ss.SMTPUser
    }
    if ss.SMTPPass != nil {
        m["smtp_pass"] = *ss.SMTPPass
    }
    if ss.SMTPFrom != nil {
        m["smtp_from"] = *ss.SMTPFrom
    }
    if ss.SMTPSecure != nil {
        m["smtp_secure"] = *ss.SMTPSecure
    }

    if ss.NotifyBeforeDays != nil {
        m["notify_before_days"] = *ss.NotifyBeforeDays
    }
    if ss.NotifyOnOperation != nil {
        m["notify_on_operation"] = *ss.NotifyOnOperation
    }

    return m
}

func (a *App) ajaxLogin(w http.ResponseWriter, r *http.Request, session *Session, cfg *Config, pass string) (map[string]any, int) {
    now := time.Now()
    maxFails := 5
    window := 10 * time.Minute

    filtered := make([]time.Time, 0, len(session.LoginFailures))
    for _, ts := range session.LoginFailures {
        if now.Sub(ts) < window {
            filtered = append(filtered, ts)
        }
    }
    session.LoginFailures = filtered

    if len(filtered) >= maxFails {
        oldest := filtered[0]
        retryAfter := window - now.Sub(oldest)
        retryMinutes := int(retryAfter.Minutes()) + 1
        if retryMinutes < 1 {
            retryMinutes = 1
        }
        return map[string]any{"success": false, "message": fmt.Sprintf("失败次数过多，请 %d 分钟后再试", retryMinutes)}, http.StatusOK
    }

    stored := cfg.PanelPass
    if strings.TrimSpace(stored) == "" {
        newSession := a.sessions.Regenerate(w, r, session)
        newSession.IsLoggedIn = true
        newSession.LoginFailures = nil
        return map[string]any{"success": true, "message": "登录成功"}, http.StatusOK
    }

    if strings.HasPrefix(stored, "$2") {
        if bcrypt.CompareHashAndPassword([]byte(stored), []byte(pass)) == nil {
            newSession := a.sessions.Regenerate(w, r, session)
            newSession.IsLoggedIn = true
            newSession.LoginFailures = nil
            return map[string]any{"success": true, "message": "登录成功"}, http.StatusOK
        }
    } else {
        if pass == stored {
            _, _ = a.cfgStore.Modify(func(c *Config) error {
                hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
                if err != nil {
                    return err
                }
                c.PanelPass = string(hash)
                return nil
            })

            newSession := a.sessions.Regenerate(w, r, session)
            newSession.IsLoggedIn = true
            newSession.LoginFailures = nil
            return map[string]any{"success": true, "message": "登录成功"}, http.StatusOK
        }
    }

    session.LoginFailures = append(session.LoginFailures, now)
    return map[string]any{"success": false, "message": "密码错误"}, http.StatusOK
}

func (a *App) dispatchAdminAction(w http.ResponseWriter, r *http.Request, action string, cfg *Config, ctx *ServerContext, session *Session) (res map[string]any, code int) {
    defer func() {
        if rec := recover(); rec != nil {
            log.Printf("panic in action %s: %v", action, rec)
            res = map[string]any{"success": false, "message": "系统错误"}
            code = http.StatusInternalServerError
        }
    }()

    switch action {
    case "charge":
        return a.ajaxCharge(r, ctx)
    case "create":
        return a.ajaxCreate(r, cfg, ctx)
    case "save_edit":
        return a.ajaxSaveEdit(r, ctx)
    case "delete":
        return a.ajaxDelete(r, ctx)
    case "batch":
        return a.ajaxBatch(r, ctx)
    case "refresh_cache":
        return a.ajaxRefreshCache(ctx)
    case "server_op":
        return a.ajaxServerOp(r, cfg)
    case "settings_op":
        return a.ajaxSettingsOp(r, cfg)
    case "test_email":
        return a.ajaxTestEmail(r, cfg, ctx)
    case "restore":
        return a.ajaxRestore(r, ctx)
    case "get_users":
        return a.ajaxGetUsers(ctx)
    case "get_logs":
        return a.ajaxGetLogs(r, ctx)
    case "run_auto_check":
        return a.ajaxRunAutoCheck(ctx)
    default:
        return map[string]any{"success": false, "message": "未知操作"}, http.StatusBadRequest
    }
}

var invalidNameChars = regexp.MustCompile(`[<>'"/\\]`)
var dateInputPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var hhmmPattern = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

type updateUserData struct {
    NewName       string
    NewGroup      *string
    NewPass       string
    Email         *string
    ExpDate       string
    ExpDateAction string
    OpenDate      string
    LastRecharge  string
    Disabled      *bool
    Note          string
}

func (a *App) ajaxCharge(r *http.Request, ctx *ServerContext) (map[string]any, int) {
    uid := strings.TrimSpace(r.FormValue("charge_uid"))
    days := maxInt(parseInt(r.FormValue("charge_days"), 0), 1)
    note := strings.TrimSpace(r.FormValue("charge_note"))
    if uid == "" {
        return map[string]any{"success": false, "message": "缺少用户ID"}, http.StatusOK
    }

    ok, err := a.rechargeUser(ctx, nil, uid, days, note, false)
    if err != nil || !ok {
        return map[string]any{"success": false, "message": "充值失败"}, http.StatusOK
    }

    user, _ := a.getFrontendUser(ctx, uid)
    return map[string]any{
        "success": true,
        "message": fmt.Sprintf("充值成功 +%d天", days),
        "user":    user,
    }, http.StatusOK
}

func (a *App) ajaxCreate(r *http.Request, cfg *Config, ctx *ServerContext) (map[string]any, int) {
    name := strings.TrimSpace(r.FormValue("name"))
    pass := r.FormValue("pass")
    group := strings.TrimSpace(r.FormValue("group"))
    email := strings.TrimSpace(r.FormValue("email"))
    days := parseInt(r.FormValue("days"), 0)
    note := strings.TrimSpace(r.FormValue("note"))
    copyFrom := strings.TrimSpace(r.FormValue("copyFrom"))

    if name == "" {
        return map[string]any{"success": false, "message": "用户名不能为空"}, http.StatusOK
    }
    if utf8.RuneCountInString(name) > 64 {
        return map[string]any{"success": false, "message": "用户名不能超过64个字符"}, http.StatusOK
    }
    if utf8.RuneCountInString(pass) > 128 {
        return map[string]any{"success": false, "message": "密码不能超过128个字符"}, http.StatusOK
    }
    if utf8.RuneCountInString(group) > 32 {
        return map[string]any{"success": false, "message": "分组名不能超过32个字符"}, http.StatusOK
    }
    if email != "" && (!looksLikeEmail(email) || utf8.RuneCountInString(email) > 128) {
        return map[string]any{"success": false, "message": "邮箱格式无效"}, http.StatusOK
    }
    if utf8.RuneCountInString(note) > 256 {
        return map[string]any{"success": false, "message": "备注不能超过256个字符"}, http.StatusOK
    }
    if invalidNameChars.MatchString(name) {
        return map[string]any{"success": false, "message": "用户名包含非法字符"}, http.StatusOK
    }

    id, err := a.createUser(ctx, name, pass, group, email, days, note, copyFrom)
    if err != nil || id == "" {
        return map[string]any{"success": false, "message": "创建失败，用户名可能已存在"}, http.StatusOK
    }

    if ctx.ServerID != "" {
        _, _ = a.cfgStore.Modify(func(c *Config) error {
            if c.ServerSettings == nil {
                c.ServerSettings = map[string]ServerSettings{}
            }
            ss := c.ServerSettings[ctx.ServerID]
            ss.DefaultTemplateUser = strPtr(copyFrom)
            c.ServerSettings[ctx.ServerID] = ss
            return nil
        })
    }

    user, _ := a.getFrontendUser(ctx, id)
    return map[string]any{"success": true, "message": "用户创建成功", "user": user}, http.StatusOK
}

func (a *App) ajaxSaveEdit(r *http.Request, ctx *ServerContext) (map[string]any, int) {
    uid := strings.TrimSpace(r.FormValue("uid"))
    if uid == "" {
        return map[string]any{"success": false, "message": "缺少用户ID"}, http.StatusOK
    }

    data := updateUserData{
        NewName:       strings.TrimSpace(r.FormValue("newname")),
        NewGroup:      strPtr(r.FormValue("newgroup")),
        NewPass:       r.FormValue("newpass"),
        Email:         strPtr(strings.TrimSpace(r.FormValue("email"))),
        ExpDate:       strings.TrimSpace(r.FormValue("expdate")),
        ExpDateAction: strings.TrimSpace(r.FormValue("expdate_action")),
        OpenDate:      strings.TrimSpace(r.FormValue("opendate")),
        LastRecharge:  strings.TrimSpace(r.FormValue("lastrecharge")),
        Note:          strings.TrimSpace(r.FormValue("note")),
    }

    if v, ok := r.Form["disabled"]; ok && len(v) > 0 {
        b := parseBool(v[0], false)
        data.Disabled = &b
    }

    if data.ExpDateAction != "" && data.ExpDateAction != "permanent" && data.ExpDateAction != "clear" {
        return map[string]any{"success": false, "message": "到期操作无效"}, http.StatusOK
    }
    if data.NewName != "" {
        if utf8.RuneCountInString(data.NewName) > 64 || invalidNameChars.MatchString(data.NewName) {
            return map[string]any{"success": false, "message": "用户名无效"}, http.StatusOK
        }
    }
    if data.NewPass != "" && utf8.RuneCountInString(data.NewPass) > 128 {
        return map[string]any{"success": false, "message": "密码不能超过128个字符"}, http.StatusOK
    }
    if data.NewGroup != nil && utf8.RuneCountInString(*data.NewGroup) > 32 {
        return map[string]any{"success": false, "message": "分组名不能超过32个字符"}, http.StatusOK
    }
    if data.Email != nil && *data.Email != "" {
        if !looksLikeEmail(*data.Email) || utf8.RuneCountInString(*data.Email) > 128 {
            return map[string]any{"success": false, "message": "邮箱格式无效"}, http.StatusOK
        }
    }
    if data.Note != "" && utf8.RuneCountInString(data.Note) > 256 {
        return map[string]any{"success": false, "message": "备注不能超过256个字符"}, http.StatusOK
    }
    if data.OpenDate != "" && !dateInputPattern.MatchString(data.OpenDate) {
        return map[string]any{"success": false, "message": "开通日期格式错误"}, http.StatusOK
    }
    if data.LastRecharge != "" && !dateInputPattern.MatchString(data.LastRecharge) {
        return map[string]any{"success": false, "message": "最后充值日期格式错误"}, http.StatusOK
    }
    if data.ExpDateAction == "" && data.ExpDate != "" && !dateInputPattern.MatchString(data.ExpDate) {
        return map[string]any{"success": false, "message": "到期日期格式错误"}, http.StatusOK
    }

    ok, err := a.updateUserProfile(ctx, nil, uid, data)
    if err != nil || !ok {
        return map[string]any{"success": false, "message": "修改失败"}, http.StatusOK
    }

    user := a.buildFrontendUserForAction(ctx, uid, data.Disabled)
    return map[string]any{"success": true, "message": "修改成功", "user": user}, http.StatusOK
}

func (a *App) ajaxDelete(r *http.Request, ctx *ServerContext) (map[string]any, int) {
    uid := strings.TrimSpace(r.FormValue("uid"))
    note := strings.TrimSpace(r.FormValue("note"))
    if uid == "" {
        return map[string]any{"success": false, "message": "缺少用户ID"}, http.StatusOK
    }

    ok, err := a.deleteUser(ctx, nil, uid, note)
    if err != nil || !ok {
        return map[string]any{"success": false, "message": "删除失败"}, http.StatusOK
    }

    return map[string]any{"success": true, "message": "用户已永久删除", "deletedId": uid}, http.StatusOK
}

func (a *App) ajaxBatch(r *http.Request, ctx *ServerContext) (map[string]any, int) {
    typeOp := strings.TrimSpace(r.FormValue("type"))
    note := strings.TrimSpace(r.FormValue("note"))
    uidsRaw := strings.TrimSpace(r.FormValue("uids"))
    if uidsRaw == "" {
        return map[string]any{"success": false, "message": "未选择用户"}, http.StatusOK
    }

    rawIDs := strings.Split(uidsRaw, ",")
    uids := make([]string, 0, len(rawIDs))
    for _, id := range rawIDs {
        id = strings.TrimSpace(id)
        if id != "" {
            uids = append(uids, id)
        }
    }

    successCount := 0
    updatedUsers := make([]LocalUser, 0)

    var editData updateUserData
    if typeOp == "edit" {
        editData = updateUserData{
            NewGroup:      strPtr(r.FormValue("newgroup")),
            OpenDate:      strings.TrimSpace(r.FormValue("opendate")),
            LastRecharge:  strings.TrimSpace(r.FormValue("lastrecharge")),
            ExpDate:       strings.TrimSpace(r.FormValue("expdate")),
            ExpDateAction: strings.TrimSpace(r.FormValue("expdate_action")),
            Note:          note,
        }
        if editData.ExpDateAction != "" && editData.ExpDateAction != "permanent" && editData.ExpDateAction != "clear" {
            return map[string]any{"success": false, "message": "到期操作无效"}, http.StatusOK
        }
        if editData.NewGroup != nil && utf8.RuneCountInString(*editData.NewGroup) > 32 {
            return map[string]any{"success": false, "message": "分组名不能超过32个字符"}, http.StatusOK
        }
        if editData.OpenDate != "" && !dateInputPattern.MatchString(editData.OpenDate) {
            return map[string]any{"success": false, "message": "开通日期格式错误"}, http.StatusOK
        }
        if editData.LastRecharge != "" && !dateInputPattern.MatchString(editData.LastRecharge) {
            return map[string]any{"success": false, "message": "最后充值日期格式错误"}, http.StatusOK
        }
        if editData.ExpDateAction == "" && editData.ExpDate != "" && !dateInputPattern.MatchString(editData.ExpDate) {
            return map[string]any{"success": false, "message": "到期日期格式错误"}, http.StatusOK
        }
    }

    var days int
    if typeOp == "charge" {
        days = maxInt(parseInt(r.FormValue("days"), 0), 1)
    }

    preFetched := map[string]EmbyUser{}
    if typeOp != "delete" && len(uids) > 0 {
        chunks := chunkStrings(uids, 50)
        for _, ids := range chunks {
            users, err := a.listEmbyUsersByIDs(ctx, ids, "Policy,Configuration")
            if err != nil {
                continue
            }
            for _, u := range users {
                preFetched[u.ID] = u
            }
        }
    }

    for _, uid := range uids {
        var ok bool
        var err error
        var user *EmbyUser
        if u, exists := preFetched[uid]; exists {
            user = &u
        }

        switch typeOp {
        case "charge":
            ok, err = a.rechargeUser(ctx, user, uid, days, note, false)
        case "enable":
            enabled := false
            ok, err = a.updateUserProfile(ctx, user, uid, updateUserData{Disabled: &enabled, Note: note})
        case "disable":
            disabled := true
            ok, err = a.updateUserProfile(ctx, user, uid, updateUserData{Disabled: &disabled, Note: note})
        case "delete":
            ok, err = a.deleteUser(ctx, user, uid, note)
        case "edit":
            ok, err = a.updateUserProfile(ctx, user, uid, editData)
        default:
            return map[string]any{"success": false, "message": "未知批量操作"}, http.StatusOK
        }

        if err == nil && ok {
            successCount++
            if typeOp != "delete" {
                var disabledOverride *bool
                if typeOp == "enable" {
                    v := false
                    disabledOverride = &v
                } else if typeOp == "disable" {
                    v := true
                    disabledOverride = &v
                }
                if fu := a.buildFrontendUserForAction(ctx, uid, disabledOverride); fu != nil {
                    updatedUsers = append(updatedUsers, *fu)
                }
            }
        }
    }

    return map[string]any{
        "success":      true,
        "message":      fmt.Sprintf("批量操作完成，成功 %d 个", successCount),
        "updatedUsers": updatedUsers,
    }, http.StatusOK
}

func (a *App) ajaxGetUsers(ctx *ServerContext) (map[string]any, int) {
    users, err := a.getUserList(ctx)
    if err != nil {
        return map[string]any{"success": false, "message": "获取用户失败"}, http.StatusOK
    }
    return map[string]any{"success": true, "users": users}, http.StatusOK
}

func (a *App) ajaxRefreshCache(ctx *ServerContext) (map[string]any, int) {
    a.invalidateUserCache(ctx)
    return map[string]any{"success": true, "message": "缓存已清除，正在刷新..."}, http.StatusOK
}

func (a *App) ajaxServerOp(r *http.Request, cfg *Config) (map[string]any, int) {
    subAction := strings.TrimSpace(r.FormValue("sub_action"))

    switch subAction {
    case "save":
        id := strings.TrimSpace(r.FormValue("id"))
        if id == "" {
            id = randomHex(8)
        }

        urlVal := strings.TrimSpace(r.FormValue("url"))
        key := strings.TrimSpace(r.FormValue("key"))
        name := strings.TrimSpace(r.FormValue("name"))

        if urlVal == "" || key == "" {
            return map[string]any{"success": false, "message": "地址和API Key不能为空"}, http.StatusOK
        }

        if !strings.HasPrefix(urlVal, "http://") && !strings.HasPrefix(urlVal, "https://") {
            urlVal = "http://" + urlVal
        }
        urlVal = strings.TrimRight(urlVal, "/")

        var oldName string
        var oldKey string
        for _, s := range cfg.Servers {
            if s.ID == id {
                oldName = s.Name
                oldKey = s.Key
                break
            }
        }

        if key == "******" {
            key = oldKey
        }

        if name == "" {
            realKey := key
            if realKey == "" {
                realKey = oldKey
            }
            fetched, _ := a.fetchEmbyServerName(urlVal, realKey)
            name = fallback(fetched, "未命名服务器")
        }

        _, err := a.cfgStore.Modify(func(c *Config) error {
            found := false
            for i := range c.Servers {
                if c.Servers[i].ID == id {
                    c.Servers[i] = ServerConfig{ID: id, Name: name, URL: urlVal, Key: key}
                    found = true
                    break
                }
            }
            if !found {
                c.Servers = append(c.Servers, ServerConfig{ID: id, Name: name, URL: urlVal, Key: key})
            }

            if len(c.Servers) == 1 {
                c.CurrentServerID = id
            }

            if oldName != "" && oldName != name {
                oldSafe := safeFileName(oldName)
                newSafe := safeFileName(name)
                if oldSafe != newSafe {
                    _ = os.Rename(filepath.Join(a.dataDir, "users", oldSafe), filepath.Join(a.dataDir, "users", newSafe))
                    _ = os.Rename(filepath.Join(a.dataDir, "log", oldSafe), filepath.Join(a.dataDir, "log", newSafe))
                }
            }
            return nil
        })
        if err != nil {
            return map[string]any{"success": false, "message": "服务保存失败"}, http.StatusOK
        }

        newCfg, _ := a.cfgStore.Load()
        return map[string]any{"success": true, "message": "服务器保存成功", "servers": newCfg.Servers, "currentServerId": newCfg.CurrentServerID}, http.StatusOK

    case "delete":
        id := strings.TrimSpace(r.FormValue("id"))
        if id == "" {
            return map[string]any{"success": false, "message": "缺少服务器ID"}, http.StatusOK
        }

        if len(cfg.Servers) <= 1 {
            return map[string]any{"success": false, "message": "无法删除唯一服务器"}, http.StatusOK
        }

        var deleted *ServerConfig
        _, err := a.cfgStore.Modify(func(c *Config) error {
            next := make([]ServerConfig, 0, len(c.Servers))
            for i := range c.Servers {
                if c.Servers[i].ID == id {
                    tmp := c.Servers[i]
                    deleted = &tmp
                    continue
                }
                next = append(next, c.Servers[i])
            }
            c.Servers = next
            if c.CurrentServerID == id {
                if len(c.Servers) > 0 {
                    c.CurrentServerID = c.Servers[0].ID
                } else {
                    c.CurrentServerID = ""
                }
            }
            return nil
        })
        if err != nil {
            return map[string]any{"success": false, "message": "删除失败"}, http.StatusOK
        }

        if deleted != nil {
            safe := safeFileName(deleted.Name)
            _ = deleteDir(filepath.Join(a.dataDir, "users", safe))
            _ = deleteDir(filepath.Join(a.dataDir, "log", safe))
        }

        newCfg, _ := a.cfgStore.Load()
        return map[string]any{"success": true, "message": "服务器及相关数据已删除", "servers": newCfg.Servers, "currentServerId": newCfg.CurrentServerID}, http.StatusOK

    case "switch":
        id := strings.TrimSpace(r.FormValue("id"))
        if id == "" {
            return map[string]any{"success": false, "message": "缺少服务器ID"}, http.StatusOK
        }

        exists := false
        for _, s := range cfg.Servers {
            if s.ID == id {
                exists = true
                break
            }
        }
        if !exists {
            return map[string]any{"success": false, "message": "服务器不存在"}, http.StatusOK
        }

        _, err := a.cfgStore.Modify(func(c *Config) error {
            c.CurrentServerID = id
            return nil
        })
        if err != nil {
            return map[string]any{"success": false, "message": "切换失败"}, http.StatusOK
        }

        return map[string]any{"success": true, "message": "切换成功，正在刷新...", "currentServerId": id}, http.StatusOK

    default:
        return map[string]any{"success": false, "message": "未知服务器操作"}, http.StatusBadRequest
    }
}

func (a *App) ajaxSettingsOp(r *http.Request, cfg *Config) (map[string]any, int) {
    serverID := strings.TrimSpace(r.FormValue("serverId"))
    if serverID == "" {
        serverID = strings.TrimSpace(cfg.CurrentServerID)
    }
    if serverID == "" && len(cfg.Servers) > 0 {
        serverID = cfg.Servers[0].ID
    }

    newCfg, err := a.cfgStore.Modify(func(c *Config) error {
        if c.ServerSettings == nil {
            c.ServerSettings = map[string]ServerSettings{}
        }

        ss := c.ServerSettings[serverID]

        if checkTime, ok := r.Form["checkTime"]; ok {
            val := strings.TrimSpace(firstOrEmpty(checkTime))
            if val != "" && !hhmmPattern.MatchString(val) {
                return errors.New("时间格式错误，请使用 HH:mm")
            }
            ss.CheckTime = strPtr(val)
        }

        if v, ok := r.Form["autoTaskEnabled"]; ok {
            b := parseBool(firstOrEmpty(v), true)
            ss.AutoTaskEnabled = &b
        }

        if v, ok := r.Form["logRetentionDays"]; ok {
            days := parseInt(firstOrEmpty(v), 30)
            if days < 1 {
                days = 30
            }
            ss.LogRetentionDays = intPtr(days)
        }

        if v, ok := r.Form["panelPass"]; ok {
            p := strings.TrimSpace(firstOrEmpty(v))
            if p == "******" {
            } else if p == "" {
                c.PanelPass = ""
            } else {
                hash, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
                if err != nil {
                    return err
                }
                c.PanelPass = string(hash)
            }
        }

        if v, ok := r.Form["smtp_host"]; ok {
            ss.SMTPHost = strPtr(strings.TrimSpace(firstOrEmpty(v)))
        }
        if v, ok := r.Form["smtp_port"]; ok {
            p := parseInt(strings.TrimSpace(firstOrEmpty(v)), 0)
            ss.SMTPPort = intPtr(p)
        }
        if v, ok := r.Form["smtp_user"]; ok {
            ss.SMTPUser = strPtr(strings.TrimSpace(firstOrEmpty(v)))
        }
        if v, ok := r.Form["smtp_pass"]; ok {
            p := strings.TrimSpace(firstOrEmpty(v))
            if p != "******" {
                ss.SMTPPass = strPtr(p)
            }
        }
        if v, ok := r.Form["smtp_from"]; ok {
            ss.SMTPFrom = strPtr(strings.TrimSpace(firstOrEmpty(v)))
        }
        if v, ok := r.Form["smtp_secure"]; ok {
            ss.SMTPSecure = strPtr(strings.TrimSpace(firstOrEmpty(v)))
        }

        if v, ok := r.Form["notify_before_days"]; ok {
            days := parseInt(strings.TrimSpace(firstOrEmpty(v)), 3)
            if days < 0 {
                days = 0
            }
            ss.NotifyBeforeDays = intPtr(days)
        }

        if v, ok := r.Form["notify_on_operation"]; ok {
            b := parseBool(firstOrEmpty(v), true)
            ss.NotifyOnOperation = &b
        }

        if v, ok := r.Form["expireAction"]; ok {
            ea := strings.TrimSpace(firstOrEmpty(v))
            if ea != "delete" {
                ea = "disable"
            }
            ss.ExpireAction = strPtr(ea)
        }

        if v, ok := r.Form["restoreTemplateUser"]; ok {
            ss.RestoreTemplateUser = strPtr(strings.TrimSpace(firstOrEmpty(v)))
        }

        if v, ok := r.Form["hiddenUsers"]; ok {
            lines := strings.Split(firstOrEmpty(v), "\n")
            list := make([]string, 0)
            for _, item := range lines {
                item = strings.TrimSpace(item)
                if item != "" {
                    list = append(list, item)
                }
            }
            hiddenMap := normalizeHiddenUsers(c.HiddenUsers)
            if serverID != "" {
                hiddenMap[serverID] = list
            }
            c.HiddenUsers = hiddenMap
        }

        c.ServerSettings[serverID] = ss
        return nil
    })
    if err != nil {
        return map[string]any{"success": false, "message": err.Error()}, http.StatusOK
    }

    serverSettings := map[string]any{}
    if ss, ok := newCfg.ServerSettings[serverID]; ok {
        serverSettings = serverSettingsToMap(ss)
    }

    return map[string]any{
        "success":        true,
        "message":        "系统设置已保存(当前服务器)",
        "serverSettings": serverSettings,
        "hiddenUsers":    normalizeHiddenUsers(newCfg.HiddenUsers),
    }, http.StatusOK
}

func (a *App) ajaxTestEmail(r *http.Request, cfg *Config, ctx *ServerContext) (map[string]any, int) {
    to := strings.TrimSpace(r.FormValue("test_to"))
    host := strings.TrimSpace(r.FormValue("smtp_host"))
    port := parseInt(strings.TrimSpace(r.FormValue("smtp_port")), 0)
    user := strings.TrimSpace(r.FormValue("smtp_user"))
    pass := strings.TrimSpace(r.FormValue("smtp_pass"))
    from := strings.TrimSpace(r.FormValue("smtp_from"))
    secure := strings.TrimSpace(r.FormValue("smtp_secure"))

    if host == "" || user == "" || pass == "" || to == "" {
        return map[string]any{"success": false, "message": "请填写完整的 SMTP 信息和接收邮箱"}, http.StatusOK
    }

    if pass == "******" {
        sid := strings.TrimSpace(r.FormValue("serverId"))
        if sid == "" {
            sid = ctx.ServerID
        }
        if ss, ok := cfg.ServerSettings[sid]; ok && ss.SMTPPass != nil {
            pass = *ss.SMTPPass
        }
    }

    settings := ctx.Settings
    settings.SMTPHost = host
    settings.SMTPPort = port
    settings.SMTPUser = user
    settings.SMTPPass = pass
    settings.SMTPFrom = from
    settings.SMTPSecure = fallback(secure, "ssl")

    subject := "Emby Panel 邮件测试"
    body := fmt.Sprintf("<h1>邮件发送测试</h1><p>发送时间: %s</p>", time.Now().Format("2006-01-02 15:04:05"))
    if err := sendEmail(settings, to, subject, body); err != nil {
        return map[string]any{"success": false, "message": "发送失败: " + err.Error()}, http.StatusOK
    }
    return map[string]any{"success": true, "message": "测试邮件已发送至 " + to}, http.StatusOK
}

func (a *App) ajaxRestore(r *http.Request, ctx *ServerContext) (map[string]any, int) {
    file, header, err := r.FormFile("backup_file")
    if err != nil {
        return map[string]any{"success": false, "message": "未上传文件"}, http.StatusOK
    }
    defer file.Close()

    ext := strings.ToLower(filepath.Ext(header.Filename))
    if ext != ".json" {
        return map[string]any{"success": false, "message": "仅支持 JSON 备份文件"}, http.StatusOK
    }

    payloadBytes, err := io.ReadAll(file)
    if err != nil {
        return map[string]any{"success": false, "message": "读取备份失败"}, http.StatusOK
    }

    var payload struct {
        Users map[string]LocalUser `json:"users"`
    }
    if err := json.Unmarshal(payloadBytes, &payload); err != nil {
        return map[string]any{"success": false, "message": "备份文件格式错误"}, http.StatusOK
    }
    if len(payload.Users) == 0 {
        return map[string]any{"success": false, "message": "备份文件中没有 users 数据"}, http.StatusOK
    }

    for id, user := range payload.Users {
        if strings.TrimSpace(id) == "" {
            continue
        }
        if err := a.db.Save(ctx.DBFile, id, user); err != nil {
            return map[string]any{"success": false, "message": "恢复失败: " + err.Error()}, http.StatusOK
        }
    }

    syncMsg := a.syncRestoredUsersToEmby(ctx)
    msg := "恢复成功"
    if syncMsg != "" {
        msg += " (" + syncMsg + ")"
    }
    msg += "，正在刷新..."

    return map[string]any{"success": true, "message": msg}, http.StatusOK
}

func (a *App) ajaxGetLogs(r *http.Request, ctx *ServerContext) (map[string]any, int) {
    fileMap, err := getLogFileMap(ctx.LogFile)
    if err != nil {
        return map[string]any{"success": false, "message": err.Error()}, http.StatusOK
    }

    if len(fileMap) == 0 {
        return map[string]any{"success": true, "files": []any{}, "content": "", "selected": ""}, http.StatusOK
    }

    selected := strings.TrimSpace(r.FormValue("file"))
    selectedPath := fileMap[selected]
    if selectedPath == "" {
        newestTime := int64(0)
        for _, p := range fileMap {
            if fi, err := os.Stat(p); err == nil {
                if fi.ModTime().Unix() > newestTime {
                    newestTime = fi.ModTime().Unix()
                    selectedPath = p
                }
            }
        }
    }

    content := ""
    if selectedPath != "" {
        content, _ = readTail(selectedPath, 256*1024)
        if fi, err := os.Stat(selectedPath); err == nil && fi.Size() > 256*1024 {
            if idx := strings.Index(content, "\n"); idx >= 0 {
                content = content[idx+1:]
            }
            content = "...(仅显示最近 262144 字节)\n" + content
        }
    }

    files := make([]map[string]any, 0, len(fileMap))
    for name, path := range fileMap {
        fi, err := os.Stat(path)
        if err != nil {
            continue
        }
        files = append(files, map[string]any{
            "name":  name,
            "mtime": fi.ModTime().Unix(),
            "size":  fi.Size(),
        })
    }
    sort.Slice(files, func(i, j int) bool {
        return toInt64(files[i]["mtime"]) > toInt64(files[j]["mtime"])
    })

    return map[string]any{
        "success":  true,
        "files":    files,
        "selected": filepath.Base(selectedPath),
        "content":  content,
    }, http.StatusOK
}

func (a *App) ajaxRunAutoCheck(ctx *ServerContext) (map[string]any, int) {
    if !ctx.Settings.AutoTaskEnabled {
        return map[string]any{"success": false, "message": "自动任务已关闭"}, http.StatusOK
    }

    if !a.autoCheckRunning.CompareAndSwap(false, true) {
        return map[string]any{"success": false, "message": "自动任务正在执行，请稍后重试"}, http.StatusOK
    }
    defer a.autoCheckRunning.Store(false)

    summary, err := a.runAutoCheckCore(ctx)
    if err != nil {
        return map[string]any{"success": false, "message": err.Error()}, http.StatusOK
    }

    return map[string]any{"success": true, "message": summary.Message}, http.StatusOK
}

func (a *App) rechargeUser(ctx *ServerContext, preFetched *EmbyUser, id string, days int, note string, forceSetExpire bool) (bool, error) {
    u := preFetched
    if u == nil {
        uu, err := a.getEmbyUser(ctx, id)
        if err != nil {
            return false, err
        }
        u = uu
    }

    if u == nil {
        return false, errors.New("emby user not found")
    }

    policy := cloneMap(u.Policy)
    policy["IsDisabled"] = false
    if err := a.embyPost(ctx, "/Users/"+id+"/Policy", policy); err != nil {
        return false, err
    }

    local, err := a.getUserData(ctx, id)
    if err != nil {
        return false, err
    }

    isPermanent := !forceSetExpire && isPermanentValue(local.ExpireDate, local.DaysLeft)
    newExpire := ""
    if !isPermanent {
        currentExpireTs := int64(0)
        if strings.TrimSpace(local.ExpireDate) != "" {
            if t, err := time.ParseInLocation("2006-01-02 15:04:05", local.ExpireDate+" 23:59:59", time.Local); err == nil {
                currentExpireTs = t.Unix()
            }
        }

        var ts int64
        if currentExpireTs > time.Now().Unix() {
            ts = currentExpireTs + int64(days*86400)
        } else {
            ts = time.Now().Unix() + int64(days*86400)
        }
        newExpire = time.Unix(ts, 0).In(time.Local).Format("2006-01-02")
    }

    nowText := time.Now().In(time.Local).Format("2006-01-02 15:04:05")
    local.LastRecharge = nowText
    local.ExpireDate = newExpire
    if isPermanentValue(newExpire, local.DaysLeft) {
        local.DaysLeft = "永久"
    } else {
        local.DaysLeft = calcDaysLeft(newExpire)
    }
    local.Status = statusEnabled
    local.ChargeHistory = append(local.ChargeHistory, ChargeRecord{
        Date: nowText,
        Days: days,
        Note: fallback(strings.TrimSpace(note), "管理员充值"),
    })

    if strings.TrimSpace(local.OpenDate) == "" && strings.TrimSpace(u.DateCreated) != "" {
        local.OpenDate = formatEmbyTime(u.DateCreated)
    }
    if strings.TrimSpace(local.Name) == "" {
        local.Name = u.Name
    }

    if err := a.db.Save(ctx.DBFile, id, local); err != nil {
        return false, err
    }
    a.invalidateUserCache(ctx)

    a.writeLog(ctx, nil, fmt.Sprintf("用户充值: %s (ID: %s), 增加天数: %d, 备注: %s", local.Name, id, days, note))
    a.notifyUserOperation(ctx, local, "recharge", map[string]any{
        "days":       days,
        "expireDate": newExpire,
        "note":       note,
    })

    return true, nil
}

func (a *App) createUser(ctx *ServerContext, name, pass, group, email string, days int, note, copyFromID string) (string, error) {
    var sourceUser *EmbyUser
    if strings.TrimSpace(copyFromID) != "" {
        sourceUser, _ = a.getEmbyUser(ctx, copyFromID)
    }

    var createRes struct {
        ID string `json:"Id"`
    }
    if err := a.embyRequestJSON(ctx, http.MethodPost, "/Users/New", map[string]any{"Name": name}, &createRes); err != nil {
        return "", err
    }
    if strings.TrimSpace(createRes.ID) == "" {
        return "", errors.New("emby create failed")
    }

    id := createRes.ID

    if sourceUser != nil {
        if len(sourceUser.Policy) > 0 {
            p := cloneMap(sourceUser.Policy)
            p["IsDisabled"] = false
            _ = a.embyPost(ctx, "/Users/"+id+"/Policy", p)
        }
        if len(sourceUser.Configuration) > 0 {
            _ = a.embyPost(ctx, "/Users/"+id+"/Configuration", sourceUser.Configuration)
        }
    }

    if strings.TrimSpace(pass) != "" {
        _ = a.embyPost(ctx, "/Users/"+id+"/Password", map[string]any{
            "Id":        id,
            "CurrentPw": "",
            "NewPw":     pass,
        })
    }

    local := getDefaultUserData()
    local.Name = name
    local.Group = group
    local.Email = email
    local.OpenDate = time.Now().In(time.Local).Format("2006-01-02 15:04:05")
    local.Status = statusEnabled
    if err := a.db.Save(ctx.DBFile, id, local); err != nil {
        return "", err
    }
    a.invalidateUserCache(ctx)

    a.writeLog(ctx, nil, fmt.Sprintf("创建用户: %s (ID: %s), 分组: %s, 邮箱: %s, 初始天数: %d, 备注: %s", name, id, group, email, days, note))

    if days > 0 {
        _, _ = a.rechargeUser(ctx, nil, id, days, fallback(note, "初始开通"), true)
    }

    return id, nil
}

func (a *App) updateUserProfile(ctx *ServerContext, preFetched *EmbyUser, id string, d updateUserData) (bool, error) {
    u := preFetched
    if u == nil {
        uu, err := a.getEmbyUser(ctx, id)
        if err != nil {
            return false, err
        }
        u = uu
    }
    if u == nil {
        return false, errors.New("emby user not found")
    }

    if d.NewName != "" && d.NewName != u.Name {
        if err := a.embyPost(ctx, "/Users/"+id, map[string]any{"Name": d.NewName}); err != nil {
            return false, err
        }
    }
    if d.NewPass != "" {
        if err := a.embyPost(ctx, "/Users/"+id+"/Password", map[string]any{"Id": id, "CurrentPw": "", "NewPw": d.NewPass}); err != nil {
            return false, err
        }
    }

    if d.Disabled != nil {
        policy := cloneMap(u.Policy)
        policy["IsDisabled"] = *d.Disabled
        if err := a.embyPost(ctx, "/Users/"+id+"/Policy", policy); err != nil {
            return false, err
        }
    }

    local, err := a.getUserData(ctx, id)
    if err != nil {
        return false, err
    }

    if d.NewName != "" {
        local.Name = d.NewName
    }
    if d.NewGroup != nil {
        local.Group = *d.NewGroup
    }
    if d.OpenDate != "" {
        local.OpenDate = d.OpenDate
    }
    if d.LastRecharge != "" {
        local.LastRecharge = d.LastRecharge
    }
    if d.Email != nil {
        local.Email = *d.Email
    }

    if d.ExpDateAction == "permanent" || d.ExpDateAction == "clear" {
        local.ExpireDate = ""
        local.DaysLeft = "永久"
    } else if d.ExpDate != "" {
        expDate := d.ExpDate
        if expDate == appPermanentDate {
            expDate = ""
        }
        local.ExpireDate = expDate
        if isPermanentValue(expDate, local.DaysLeft) {
            local.DaysLeft = "永久"
        } else {
            local.DaysLeft = calcDaysLeft(expDate)
        }
    }

    isAdmin := mapBool(u.Policy, "IsAdministrator")
    oldDisabled := mapBool(u.Policy, "IsDisabled")

    if isAdmin {
        local.Status = statusAdmin
        local.DaysLeft = "永久"
        local.ExpireDate = ""
    } else if d.Disabled != nil {
        if *d.Disabled {
            local.Status = statusDisabled
        } else {
            local.Status = statusEnabled
        }
    }

    if err := a.db.Save(ctx.DBFile, id, local); err != nil {
        return false, err
    }
    a.invalidateUserCache(ctx)

    details := make([]string, 0)
    if d.NewName != "" {
        details = append(details, "新用户名:"+d.NewName)
    }
    if d.NewGroup != nil {
        details = append(details, "分组:"+*d.NewGroup)
    }
    if d.NewPass != "" {
        details = append(details, "修改密码")
    }
    if d.Email != nil {
        details = append(details, "邮箱:"+*d.Email)
    }
    if d.ExpDateAction == "permanent" {
        details = append(details, "有效期:永久")
    } else if d.ExpDateAction == "clear" {
        details = append(details, "有效期:未设置")
    } else if d.ExpDate != "" {
        details = append(details, "有效期:"+d.ExpDate)
    }
    if d.Disabled != nil {
        if *d.Disabled {
            details = append(details, "状态:禁用")
        } else {
            details = append(details, "状态:启用")
        }
    }
    if d.Note != "" {
        details = append(details, "备注:"+d.Note)
    }
    a.writeLog(ctx, nil, fmt.Sprintf("修改资料: %s (ID: %s), %s", fallback(u.Name, local.Name), id, strings.Join(details, ", ")))

    statusChanged := d.Disabled != nil && (*d.Disabled != oldDisabled)
    if statusChanged && !isAdmin {
        if *d.Disabled {
            a.notifyUserOperation(ctx, local, "disable", nil)
        } else {
            a.notifyUserOperation(ctx, local, "enable", nil)
        }
    }

    return true, nil
}

func (a *App) deleteUser(ctx *ServerContext, preFetched *EmbyUser, id, note string) (bool, error) {
    local, _ := a.getUserData(ctx, id)
    name := fallback(local.Name, id)

    if err := a.embyDelete(ctx, "/Users/"+id); err != nil {
        return false, err
    }

    if err := a.db.Delete(ctx.DBFile, id); err != nil {
        return false, err
    }

    allLocal, _ := a.db.GetAll(ctx.DBFile)
    for uid, u := range allLocal {
        if uid == id {
            continue
        }
        if strings.EqualFold(strings.TrimSpace(u.Name), strings.TrimSpace(name)) {
            _ = a.db.Delete(ctx.DBFile, uid)
        }
    }

    a.invalidateUserCache(ctx)

    a.writeLog(ctx, nil, fmt.Sprintf("删除用户: %s (ID: %s), 备注: %s", name, id, note))
    if local.Email != "" {
        a.notifyUserOperation(ctx, local, "delete", nil)
    }

    return true, nil
}

func (a *App) getUserData(ctx *ServerContext, uid string) (LocalUser, error) {
    u, err := a.db.Get(ctx.DBFile, uid)
    if err != nil {
        return getDefaultUserData(), err
    }
    if u == nil {
        return getDefaultUserData(), nil
    }

    base := getDefaultUserData()
    merged := *u
    if merged.Name == "" {
        merged.Name = base.Name
    }
    if merged.Status == "" {
        merged.Status = base.Status
    }
    if merged.ChargeHistory == nil {
        merged.ChargeHistory = []ChargeRecord{}
    }
    return merged, nil
}

func getDefaultUserData() LocalUser {
    return LocalUser{
        Name:           "",
        OpenDate:       "",
        LastRecharge:   "",
        ExpireDate:     "",
        DaysLeft:       "",
        Status:         statusEnabled,
        Group:          "",
        Email:          "",
        LastNotifyDate: "",
        ChargeHistory:  []ChargeRecord{},
    }
}

func (a *App) processUserData(u EmbyUser, local LocalUser, usePolicyStatus bool) LocalUser {
    if local.ExpireDate == appPermanentDate {
        local.ExpireDate = ""
    }

    if local.ExpireDate != "" {
        local.DaysLeft = calcDaysLeft(local.ExpireDate)
    } else {
        local.DaysLeft = "永久"
    }

    if strings.TrimSpace(local.Name) == "" {
        local.Name = strings.TrimSpace(u.Name)
    }

    if strings.TrimSpace(local.OpenDate) == "" && strings.TrimSpace(u.DateCreated) != "" {
        local.OpenDate = formatEmbyTime(u.DateCreated)
    }

    isAdmin := mapBool(u.Policy, "IsAdministrator")
    disabled := mapBool(u.Policy, "IsDisabled")

    if isAdmin {
        local.Status = statusAdmin
        local.DaysLeft = "永久"
        local.ExpireDate = ""
    } else if usePolicyStatus {
        if disabled && local.Status != statusAdmin {
            local.Status = statusDisabled
        } else if !disabled && local.Status == statusDisabled {
            local.Status = statusEnabled
        }
    }

    return local
}

func formatFrontendUser(u EmbyUser, local LocalUser) LocalUser {
    local.ID = u.ID
    local.LastActivityDate = u.LastActivityDate
    local.Disabled = mapBool(u.Policy, "IsDisabled")
    local.IsAdministrator = mapBool(u.Policy, "IsAdministrator")

    if strings.TrimSpace(local.Name) == "" {
        local.Name = u.Name
    }

    if isPermanentValue(local.ExpireDate, local.DaysLeft) {
        local.DaysLeft = "永久"
        local.ExpireDate = ""
        local.SortDays = 99999
    } else {
        local.SortDays = parseInt(local.DaysLeft, 0)
    }

    return local
}

func (a *App) getUserList(ctx *ServerContext) ([]LocalUser, error) {
    if strings.TrimSpace(ctx.EmbyServerURL) == "" || strings.TrimSpace(ctx.APIKey) == "" {
        return []LocalUser{}, nil
    }

    cacheTTL := 5 * time.Minute
    embyUsers := []EmbyUser{}
    usedCache := false

    if fi, err := os.Stat(ctx.CacheFile); err == nil {
        if time.Since(fi.ModTime()) < cacheTTL {
            if b, err := os.ReadFile(ctx.CacheFile); err == nil {
                if err := json.Unmarshal(b, &embyUsers); err == nil {
                    usedCache = true
                }
            }
        }
    }

    if len(embyUsers) == 0 {
        users, err := a.listEmbyUsers(ctx, "/Users?Fields=Policy,Configuration,DateCreated,LastActivityDate")
        if err == nil {
            embyUsers = users
            if len(users) > 0 {
                if b, err := json.Marshal(users); err == nil {
                    _ = os.WriteFile(ctx.CacheFile, b, 0o644)
                }
            }
        }
    }

    if len(embyUsers) == 0 {
        return []LocalUser{}, nil
    }

    allLocal, err := a.db.GetAll(ctx.DBFile)
    if err != nil {
        return nil, err
    }

    list := make([]LocalUser, 0, len(embyUsers))
    active := map[string]struct{}{}

    for _, u := range embyUsers {
        if strings.TrimSpace(u.ID) == "" {
            continue
        }
        active[u.ID] = struct{}{}

        local, ok := allLocal[u.ID]
        if !ok {
            local = getDefaultUserData()
        }
        local = a.processUserData(u, local, !usedCache || !ok)

        if !ok {
            _ = a.db.Save(ctx.DBFile, u.ID, local)
            allLocal[u.ID] = local
        }

        frontend := formatFrontendUser(u, local)
        if usedCache && !frontend.IsAdministrator {
            frontend.Disabled = local.Status == statusDisabled
            if frontend.Disabled {
                frontend.Status = statusDisabled
            } else {
                frontend.Status = statusEnabled
            }
        }
        list = append(list, frontend)
    }

    if !usedCache {
        for id := range allLocal {
            if _, ok := active[id]; !ok {
                _ = a.db.Delete(ctx.DBFile, id)
            }
        }
    }

    return list, nil
}

func (a *App) invalidateUserCache(ctx *ServerContext) {
    if ctx == nil {
        return
    }
    if strings.TrimSpace(ctx.CacheFile) != "" {
        _ = os.Remove(ctx.CacheFile)
    }
    if strings.TrimSpace(ctx.UserServerDir) == "" {
        return
    }
    matches, err := filepath.Glob(filepath.Join(ctx.UserServerDir, "emby_users_cache*.json"))
    if err != nil {
        return
    }
    for _, file := range matches {
        _ = os.Remove(file)
    }
}

func (a *App) getFrontendUser(ctx *ServerContext, id string) (*LocalUser, error) {
    u, err := a.getEmbyUser(ctx, id)
    if err != nil {
        return nil, err
    }
    if u == nil {
        return nil, errors.New("emby user not found")
    }

    local, err := a.getUserData(ctx, id)
    if err != nil {
        return nil, err
    }
    local = a.processUserData(*u, local, true)
    frontend := formatFrontendUser(*u, local)
    return &frontend, nil
}

func (a *App) getLocalUserForResponse(ctx *ServerContext, id string) (*LocalUser, error) {
    local, err := a.getUserData(ctx, id)
    if err != nil {
        return nil, err
    }
    local.ID = id
    if strings.TrimSpace(local.DaysLeft) == "" {
        local.DaysLeft = calcDaysLeft(local.ExpireDate)
    }
    if isPermanentValue(local.ExpireDate, local.DaysLeft) {
        local.ExpireDate = ""
        local.SortDays = 99999
    } else {
        local.SortDays = parseInt(local.DaysLeft, 0)
    }
    return &local, nil
}

func (a *App) buildFrontendUserForAction(ctx *ServerContext, id string, disabledOverride *bool) *LocalUser {
    user, err := a.getFrontendUser(ctx, id)
    if err != nil || user == nil {
        user, _ = a.getLocalUserForResponse(ctx, id)
    }
    if user != nil && disabledOverride != nil && !user.IsAdministrator && user.Status != statusAdmin {
        user.Disabled = *disabledOverride
        if *disabledOverride {
            user.Status = statusDisabled
        } else {
            user.Status = statusEnabled
        }
    }
    return user
}

func (a *App) syncRestoredUsersToEmby(ctx *ServerContext) string {
    localData, err := a.db.GetAll(ctx.DBFile)
    if err != nil || len(localData) == 0 {
        return ""
    }

    embyUsers, err := a.listEmbyUsers(ctx, "/Users")
    if err != nil {
        return "无法连接 Emby，仅恢复本地数据"
    }

    byID := map[string]EmbyUser{}
    byName := map[string]EmbyUser{}
    for _, u := range embyUsers {
        byID[u.ID] = u
        byName[strings.ToLower(strings.TrimSpace(u.Name))] = u
    }

    var templateUser *EmbyUser
    tplName := strings.TrimSpace(ctx.Settings.RestoreTemplateUser)
    if tplName != "" {
        if u, ok := byName[strings.ToLower(tplName)]; ok {
            tpl, err := a.getEmbyUser(ctx, u.ID)
            if err == nil && tpl != nil {
                templateUser = tpl
            }
        }
    }

    createdCount := 0
    remappedCount := 0

    for oldID, user := range localData {
        name := strings.TrimSpace(user.Name)
        if name == "" {
            continue
        }

        if _, ok := byID[oldID]; ok {
            continue
        }

        if existing, ok := byName[strings.ToLower(name)]; ok {
            newID := existing.ID
            if _, exists := localData[newID]; !exists {
                _ = a.db.Save(ctx.DBFile, newID, user)
                _ = a.db.Delete(ctx.DBFile, oldID)
                remappedCount++
            }
            continue
        }

        var createRes struct { ID string `json:"Id"` }
        if err := a.embyRequestJSON(ctx, http.MethodPost, "/Users/New", map[string]any{"Name": name}, &createRes); err != nil {
            continue
        }
        if strings.TrimSpace(createRes.ID) == "" {
            continue
        }

        newID := createRes.ID
        _ = a.db.Save(ctx.DBFile, newID, user)
        _ = a.db.Delete(ctx.DBFile, oldID)
        createdCount++

        if templateUser != nil {
            if len(templateUser.Policy) > 0 {
                p := cloneMap(templateUser.Policy)
                p["IsDisabled"] = false
                _ = a.embyPost(ctx, "/Users/"+newID+"/Policy", p)
            }
            if len(templateUser.Configuration) > 0 {
                _ = a.embyPost(ctx, "/Users/"+newID+"/Configuration", templateUser.Configuration)
            }
        } else {
            _ = a.embyPost(ctx, "/Users/"+newID+"/Policy", map[string]any{"IsDisabled": false})
        }
    }

    msgs := make([]string, 0)
    if createdCount > 0 {
        if templateUser != nil {
            msgs = append(msgs, fmt.Sprintf("创建 %d 个用户(已应用模板)", createdCount))
        } else {
            msgs = append(msgs, fmt.Sprintf("创建 %d 个用户", createdCount))
        }
    }
    if remappedCount > 0 {
        msgs = append(msgs, fmt.Sprintf("关联 %d 个现有用户", remappedCount))
    }
    return strings.Join(msgs, ", ")
}

func (a *App) fetchEmbyServerName(urlVal, key string) (string, error) {
    if strings.TrimSpace(urlVal) == "" || strings.TrimSpace(key) == "" {
        return "", errors.New("missing url or key")
    }
    endpoint := strings.TrimRight(urlVal, "/") + "/System/Info?api_key=" + url.QueryEscape(key)
    req, err := http.NewRequest(http.MethodGet, endpoint, nil)
    if err != nil {
        return "", err
    }
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return "", fmt.Errorf("http %d", resp.StatusCode)
    }

    var res struct {
        ServerName string `json:"ServerName"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
        return "", err
    }

    return strings.TrimSpace(res.ServerName), nil
}

func (a *App) embyRequest(ctx *ServerContext, method, path string, payload any) ([]byte, error) {
    if strings.TrimSpace(ctx.EmbyServerURL) == "" || strings.TrimSpace(ctx.APIKey) == "" {
        return nil, errors.New("当前服务器未配置 API")
    }

    fullURL := strings.TrimRight(ctx.EmbyServerURL, "/") + path
    if strings.Contains(fullURL, "?") {
        fullURL += "&api_key=" + url.QueryEscape(ctx.APIKey)
    } else {
        fullURL += "?api_key=" + url.QueryEscape(ctx.APIKey)
    }

    const maxRetries = 3
    var lastErr error

    for attempt := 1; attempt <= maxRetries; attempt++ {
        var body io.Reader
        var raw []byte
        if payload != nil {
            b, err := json.Marshal(payload)
            if err != nil {
                return nil, err
            }
            raw = b
            body = bytes.NewReader(b)
        }

        req, err := http.NewRequest(method, fullURL, body)
        if err != nil {
            return nil, err
        }
        req.Header.Set("Accept", "application/json")
        req.Header.Set("Content-Type", "application/json")
        if payload != nil {
            req.ContentLength = int64(len(raw))
        }

        resp, err := a.httpClient.Do(req)
        if err != nil {
            lastErr = err
            if attempt < maxRetries {
                time.Sleep(1 * time.Second)
                continue
            }
            return nil, err
        }

        respBody, _ := io.ReadAll(resp.Body)
        _ = resp.Body.Close()

        if resp.StatusCode >= 500 {
            lastErr = fmt.Errorf("http %d", resp.StatusCode)
            if attempt < maxRetries {
                time.Sleep(1 * time.Second)
                continue
            }
            return nil, lastErr
        }
        if resp.StatusCode >= 400 {
            return nil, fmt.Errorf("http %d", resp.StatusCode)
        }

        return respBody, nil
    }

    if lastErr == nil {
        lastErr = errors.New("emby request failed")
    }
    return nil, lastErr
}

func (a *App) embyRequestJSON(ctx *ServerContext, method, path string, payload any, out any) error {
    b, err := a.embyRequest(ctx, method, path, payload)
    if err != nil {
        return err
    }
    if len(bytes.TrimSpace(b)) == 0 || out == nil {
        return nil
    }
    return json.Unmarshal(b, out)
}

func (a *App) embyPost(ctx *ServerContext, path string, payload any) error {
    _, err := a.embyRequest(ctx, http.MethodPost, path, payload)
    return err
}

func (a *App) embyDelete(ctx *ServerContext, path string) error {
    _, err := a.embyRequest(ctx, http.MethodDelete, path, nil)
    return err
}

func (a *App) listEmbyUsers(ctx *ServerContext, path string) ([]EmbyUser, error) {
    var users []EmbyUser
    if err := a.embyRequestJSON(ctx, http.MethodGet, path, nil, &users); err != nil {
        return nil, err
    }
    for i := range users {
        if users[i].Policy == nil {
            users[i].Policy = map[string]any{}
        }
        if users[i].Configuration == nil {
            users[i].Configuration = map[string]any{}
        }
    }
    return users, nil
}

func (a *App) getEmbyUser(ctx *ServerContext, id string) (*EmbyUser, error) {
    var user EmbyUser
    if err := a.embyRequestJSON(ctx, http.MethodGet, "/Users/"+id, nil, &user); err != nil {
        return nil, err
    }
    if strings.TrimSpace(user.ID) == "" {
        return nil, nil
    }
    if user.Policy == nil {
        user.Policy = map[string]any{}
    }
    if user.Configuration == nil {
        user.Configuration = map[string]any{}
    }
    return &user, nil
}

func (a *App) listEmbyUsersByName(ctx *ServerContext, name string) ([]EmbyUser, error) {
    q := url.QueryEscape(name)
    return a.listEmbyUsers(ctx, "/Users?Name="+q)
}

func (a *App) listEmbyUsersByIDs(ctx *ServerContext, ids []string, fields string) ([]EmbyUser, error) {
    if len(ids) == 0 {
        return []EmbyUser{}, nil
    }
    path := "/Users?Ids=" + url.QueryEscape(strings.Join(ids, ","))
    if strings.TrimSpace(fields) != "" {
        path += "&Fields=" + url.QueryEscape(fields)
    }
    return a.listEmbyUsers(ctx, path)
}

func (a *App) runAutoCheckCore(ctx *ServerContext) (AutoCheckSummary, error) {
    summary := AutoCheckSummary{}

    if strings.TrimSpace(ctx.EmbyServerURL) == "" || strings.TrimSpace(ctx.APIKey) == "" {
        return summary, errors.New("当前服务器未配置 API")
    }

    localData, err := a.db.GetAll(ctx.DBFile)
    if err != nil {
        return summary, err
    }
    if len(localData) == 0 {
        summary.Message = "本地数据为空，无需检查"
        return summary, nil
    }

    expiredIDs := make([]string, 0)
    now := time.Now().In(time.Local)
    today := now.Format("2006-01-02")

    for id, local := range localData {
        if strings.TrimSpace(local.ExpireDate) == "" || local.ExpireDate == appPermanentDate {
            continue
        }

        summary.Checked++

        daysLeft, ok := calcDaysLeftValue(local.ExpireDate, now)
        if !ok {
            continue
        }

        userDirty := false
        if strings.TrimSpace(local.Email) != "" {
            shouldNotify := false
            subject := ""
            body := ""
            mark := today

            if daysLeft <= ctx.Settings.NotifyBeforeDays && daysLeft >= 0 && local.LastNotifyDate != today {
                shouldNotify = true
                subject = "【提醒】您的账号即将过期"
                body = fmt.Sprintf("<p>亲爱的用户 %s：</p><p>您的账号将于 %s 到期，剩余 %d 天。</p>", local.Name, local.ExpireDate, daysLeft)
            } else if daysLeft < 0 && local.LastNotifyDate != "expired" {
                shouldNotify = true
                mark = "expired"
                subject = "【通知】您的账号已过期"
                body = fmt.Sprintf("<p>亲爱的用户 %s：</p><p>您的账号已于 %s 过期并被禁用。</p>", local.Name, local.ExpireDate)
            }

            if shouldNotify {
                if err := sendEmail(ctx.Settings, local.Email, subject, body); err == nil {
                    local.LastNotifyDate = mark
                    userDirty = true
                    summary.Notify++
                }
            }
        }

        if daysLeft < 0 {
            expiredIDs = append(expiredIDs, id)
        }

        if userDirty {
            _ = a.db.Save(ctx.DBFile, id, local)
        }
    }

    embyMap := map[string]EmbyUser{}
    if len(expiredIDs) > 0 {
        chunks := chunkStrings(uniqueStrings(expiredIDs), 50)
        for _, ids := range chunks {
            users, err := a.listEmbyUsersByIDs(ctx, ids, "Policy")
            if err != nil {
                continue
            }
            for _, u := range users {
                embyMap[u.ID] = u
            }
        }
    }

    for _, id := range expiredIDs {
        local, ok := localData[id]
        if !ok {
            continue
        }

        u, ok := embyMap[id]
        if !ok {
            summary.Errors++
            continue
        }

        isAdmin := mapBool(u.Policy, "IsAdministrator")
        isDisabled := mapBool(u.Policy, "IsDisabled")
        if isAdmin {
            continue
        }

        if ctx.Settings.ExpireAction == "delete" {
            ok, err := a.deleteUser(ctx, &u, id, "自动检查: 已过期")
            if err == nil && ok {
                summary.Deleted++
            } else {
                summary.Errors++
            }
            continue
        }

        if !isDisabled {
            policy := cloneMap(u.Policy)
            policy["IsDisabled"] = true
            if err := a.embyPost(ctx, "/Users/"+id+"/Policy", policy); err != nil {
                summary.Errors++
                continue
            }
            local.Status = statusDisabled
            _ = a.db.Save(ctx.DBFile, id, local)
            summary.Disabled++
            a.writeLog(ctx, nil, fmt.Sprintf("自动检查: 禁用用户 %s (ID: %s), 原因: 已过期", local.Name, id))
        }
    }

    parts := []string{fmt.Sprintf("检查完成，检查 %d 个", summary.Checked)}
    if summary.Notify > 0 {
        parts = append(parts, fmt.Sprintf("通知 %d 个", summary.Notify))
    }
    if summary.Disabled > 0 {
        parts = append(parts, fmt.Sprintf("禁用 %d 个", summary.Disabled))
    }
    if summary.Deleted > 0 {
        parts = append(parts, fmt.Sprintf("删除 %d 个", summary.Deleted))
    }
    if summary.Errors > 0 {
        parts = append(parts, fmt.Sprintf("失败 %d 个", summary.Errors))
    }

    summary.Message = strings.Join(parts, "，")
    return summary, nil
}

func (a *App) startAutoCheckScheduler() {
    go func() {
        ticker := time.NewTicker(1 * time.Minute)
        defer ticker.Stop()

        for range ticker.C {
            a.runCronAutoChecks()
        }
    }()
}

func (a *App) runCronAutoChecks() {
    if !a.autoCheckRunning.CompareAndSwap(false, true) {
        return
    }
    defer a.autoCheckRunning.Store(false)

    cfg, err := a.cfgStore.Load()
    if err != nil {
        return
    }

    nowHM := time.Now().In(time.Local).Format("15:04")

    for _, server := range cfg.Servers {
        ctx, err := a.buildServerContext(cfg, server.ID)
        if err != nil {
            continue
        }
        if !ctx.Settings.AutoTaskEnabled {
            continue
        }
        checkTime := fallback(ctx.Settings.CheckTime, "00:00")
        if nowHM != checkTime {
            continue
        }

        summary, err := a.runAutoCheckCore(ctx)
        if err != nil {
            log.Printf("[cron] server=%s err=%v", server.Name, err)
            continue
        }
        log.Printf("[cron] server=%s %s", server.Name, summary.Message)
    }
}

func (a *App) handleQueryRoot(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }
    http.Redirect(w, r, "/user/user.html", http.StatusFound)
}

func (a *App) handleUserPage(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    http.ServeFile(w, r, filepath.Join(a.projectDir, "public", "user", "user.html"))
}

func (a *App) handleUserQuery(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "Method Not Allowed"})
        return
    }

    if err := parseRequestForm(r); err != nil {
        writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "请求格式错误"})
        return
    }

    if !a.passRateLimit(r) {
        writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "请求过于频繁，请稍后再试"})
        return
    }

    username := strings.TrimSpace(r.FormValue("username"))
    if username == "" {
        writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "请输入用户名"})
        return
    }
    if utf8.RuneCountInString(username) > 64 {
        writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "用户名过长"})
        return
    }

    cfg, err := a.cfgStore.Load()
    if err != nil {
        writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "配置读取失败"})
        return
    }

    requireToken := cfg.QueryRequireTok
    queryToken := strings.TrimSpace(cfg.QueryToken)
    if requireToken && queryToken != "" {
        clientToken := strings.TrimSpace(r.FormValue("token"))
        if clientToken == "" {
            clientToken = strings.TrimSpace(r.Header.Get("X-Query-Token"))
        }
        if clientToken == "" || !secureCompare(clientToken, queryToken) {
            writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "访问令牌无效"})
            return
        }
    }

    if len(cfg.Servers) == 0 {
        writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "后台未配置任何服务器"})
        return
    }

    allowAPIFallback := cfg.QueryAPIFallback
    results := make([]map[string]any, 0)

    for _, server := range cfg.Servers {
        safeName := safeFileName(server.Name)
        dbFile := filepath.Join(a.dataDir, "users", safeName, "users.db")

        if _, err := os.Stat(dbFile); err != nil {
            if strings.TrimSpace(server.URL) == "" || strings.TrimSpace(server.Key) == "" {
                continue
            }
        }

        localUser, _ := a.db.FindByName(dbFile, username)
        skipAPIFallback := false

        serverCtx, errCtx := a.buildServerContext(cfg, server.ID)
        if errCtx != nil || serverCtx == nil {
            continue
        }

        if localUser != nil && strings.TrimSpace(server.URL) != "" && strings.TrimSpace(server.Key) != "" {
            apiUser := (*EmbyUser)(nil)

            if strings.TrimSpace(localUser.ID) != "" {
                apiUser, _ = a.getEmbyUser(serverCtx, localUser.ID)
            }

            if apiUser == nil {
                usersByName, err := a.listEmbyUsersByName(serverCtx, username)
                if err == nil {
                    for _, u := range usersByName {
                        if strings.EqualFold(strings.TrimSpace(u.Name), username) {
                            tmp := u
                            apiUser = &tmp
                            break
                        }
                    }

                    if apiUser == nil {
                        if strings.TrimSpace(localUser.ID) != "" {
                            _ = a.db.Delete(dbFile, localUser.ID)
                        }
                        _ = a.deleteLocalUsersByName(dbFile, username)
                        localUser = nil
                        skipAPIFallback = true
                    }
                }
            }
        }

        if allowAPIFallback && localUser == nil && !skipAPIFallback && strings.TrimSpace(server.URL) != "" && strings.TrimSpace(server.Key) != "" {
            usersByName, err := a.listEmbyUsersByName(serverCtx, username)
            if err == nil {
                for _, u := range usersByName {
                    if strings.EqualFold(strings.TrimSpace(u.Name), username) {
                        status := statusEnabled
                        if mapBool(u.Policy, "IsDisabled") {
                            status = statusDisabled
                        }
                        if mapBool(u.Policy, "IsAdministrator") {
                            status = statusAdmin
                        }

                        localUser = &LocalUser{
                            Name:         u.Name,
                            Status:       status,
                            OpenDate:     formatEmbyDateOnly(u.DateCreated),
                            ExpireDate:   "",
                            DaysLeft:     "永久",
                            ChargeHistory: []ChargeRecord{},
                        }
                        break
                    }
                }
            }
        }

        if localUser != nil {
            expireDate := strings.TrimSpace(localUser.ExpireDate)
            daysLeft := "永久"
            expireDisplay := "永久"
            if expireDate != "" && expireDate != appPermanentDate {
                daysLeft = calcDaysLeft(expireDate)
                expireDisplay = expireDate
            }

            history := localUser.ChargeHistory
            if len(history) > 20 {
                history = history[len(history)-20:]
            }
            reverseChargeHistory(history)

            results = append(results, map[string]any{
                "serverName": server.Name,
                "name":       localUser.Name,
                "status":     fallback(localUser.Status, statusEnabled),
                "openDate":   fallback(localUser.OpenDate, "未知"),
                "expireDate": expireDisplay,
                "daysLeft":   daysLeft,
                "history":    history,
            })
        }
    }

    if len(results) > 0 {
        writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": results})
        return
    }

    writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "未找到该用户，请检查拼写"})
}

func (a *App) deleteLocalUsersByName(dbFile, name string) error {
    all, err := a.db.GetAll(dbFile)
    if err != nil {
        return err
    }
    for id, u := range all {
        if strings.EqualFold(strings.TrimSpace(u.Name), strings.TrimSpace(name)) {
            _ = a.db.Delete(dbFile, id)
        }
    }
    return nil
}

func (a *App) passRateLimit(r *http.Request) bool {
    rateDir := filepath.Join(a.dataDir, "rate_limit")
    _ = os.MkdirAll(rateDir, 0o775)

    ip := getClientIP(r)
    if ip == "" {
        ip = "unknown"
    }

    file := filepath.Join(rateDir, md5Hex(ip)+".txt")

    maxRequests := 10
    window := int64(60)
    now := time.Now().Unix()

    reqs := make([]int64, 0)

    if b, err := os.ReadFile(file); err == nil {
        scanner := bufio.NewScanner(bytes.NewReader(b))
        for scanner.Scan() {
            ts := parseInt64(strings.TrimSpace(scanner.Text()), 0)
            if now-ts < window {
                reqs = append(reqs, ts)
            }
        }
    }

    if len(reqs) >= maxRequests {
        return false
    }

    reqs = append(reqs, now)
    lines := make([]string, 0, len(reqs))
    for _, ts := range reqs {
        lines = append(lines, strconv.FormatInt(ts, 10))
    }
    _ = os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0o644)

    if randomIntn(100) == 1 {
        entries, _ := os.ReadDir(rateDir)
        for _, e := range entries {
            if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
                continue
            }
            p := filepath.Join(rateDir, e.Name())
            if fi, err := os.Stat(p); err == nil {
                if now-fi.ModTime().Unix() > 3600 {
                    _ = os.Remove(p)
                }
            }
        }
    }

    return true
}

func (a *App) downloadBackup(w http.ResponseWriter, ctx *ServerContext) error {
    users, err := a.db.GetAll(ctx.DBFile)
    if err != nil {
        return err
    }

    payload := map[string]any{
        "meta": map[string]any{
            "exportedAt": time.Now().In(time.Local).Format("2006-01-02 15:04:05"),
            "serverId":   ctx.ServerID,
            "serverName": ctx.ServerName,
            "version":    "1.0",
        },
        "users": users,
    }

    safeName := safeFileName(fallback(ctx.ServerName, "emby_backup"))
    fileName := fmt.Sprintf("%s_backup_%s.json", safeName, time.Now().In(time.Local).Format("20060102_150405"))

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
    enc := json.NewEncoder(w)
    enc.SetEscapeHTML(false)
    enc.SetIndent("", "    ")
    return enc.Encode(payload)
}

func getLogFileMap(logFile string) (map[string]string, error) {
    if strings.TrimSpace(logFile) == "" {
        return nil, errors.New("日志目录未配置")
    }

    logDir := filepath.Dir(logFile)
    if fi, err := os.Stat(logDir); err != nil || !fi.IsDir() {
        return nil, errors.New("日志目录不存在")
    }

    fileMap := map[string]string{}

    if fi, err := os.Stat(logFile); err == nil && !fi.IsDir() {
        fileMap[filepath.Base(logFile)] = logFile
    }

    pattern := logFile + ".*"
    matches, _ := filepath.Glob(pattern)
    for _, p := range matches {
        base := filepath.Base(p)
        if regexp.MustCompile(`\.\d+$`).MatchString(base) {
            if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
                fileMap[base] = p
            }
        }
    }

    return fileMap, nil
}

func (a *App) downloadLogFile(w http.ResponseWriter, ctx *ServerContext, fileName string, tail bool, maxBytes int64) error {
    fileName = strings.TrimSpace(fileName)
    if fileName == "" {
        return errors.New("missing log file")
    }

    fileMap, err := getLogFileMap(ctx.LogFile)
    if err != nil {
        return err
    }

    path := fileMap[filepath.Base(fileName)]
    if path == "" {
        return errors.New("log file not found")
    }

    fi, err := os.Stat(path)
    if err != nil {
        return err
    }

    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(path)))

    if !tail {
        w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
        f, err := os.Open(path)
        if err != nil {
            return err
        }
        defer f.Close()
        _, err = io.Copy(w, f)
        return err
    }

    if maxBytes <= 0 {
        maxBytes = 128 * 1024
    }

    offset := int64(0)
    if fi.Size() > maxBytes {
        offset = fi.Size() - maxBytes
    }

    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()

    if offset > 0 {
        if _, err := f.Seek(offset, io.SeekStart); err != nil {
            return err
        }
    }

    readSize := fi.Size() - offset
    w.Header().Set("Content-Length", strconv.FormatInt(readSize, 10))
    _, err = io.Copy(w, f)
    return err
}

func (a *App) writeLog(ctx *ServerContext, r *http.Request, msg string) {
    a.logMu.Lock()
    defer a.logMu.Unlock()

    logFile := strings.TrimSpace(ctx.LogFile)
    if logFile == "" {
        logFile = filepath.Join(a.dataDir, "log", "default", "operation_log.txt")
    }

    if err := os.MkdirAll(filepath.Dir(logFile), 0o775); err != nil {
        return
    }

    rotateLogFile(logFile, 10*1024*1024, 5)

    ip := "CLI"
    if r != nil {
        ip = getClientIP(r)
        if strings.TrimSpace(ip) == "" {
            ip = "Unknown"
        }
    }

    line := fmt.Sprintf("[%s] [%s] %s\n", time.Now().In(time.Local).Format("2006-01-02 15:04:05"), ip, msg)
    _ = appendFile(logFile, []byte(line))

    cleanupOldLogs(logFile, ctx.Settings.LogRetentionDays)
}

func (a *App) notifyUserOperation(ctx *ServerContext, local LocalUser, action string, contextData map[string]any) {
    if !ctx.Settings.NotifyOnOperation {
        return
    }
    if strings.TrimSpace(local.Email) == "" {
        return
    }

    now := time.Now().In(time.Local).Format("2006-01-02 15:04:05")

    subject := ""
    body := ""

    switch action {
    case "recharge":
        days := toInt(contextData["days"])
        expireDate := fmt.Sprint(contextData["expireDate"])
        note := fmt.Sprint(contextData["note"])
        subject = "账号充值通知"
        body = fmt.Sprintf("<p>亲爱的用户 %s：</p><p>您的账号已充值 %d 天。</p>", local.Name, days)
        if strings.TrimSpace(expireDate) != "" {
            body += fmt.Sprintf("<p>到期时间：%s</p>", expireDate)
        }
        if strings.TrimSpace(note) != "" {
            body += fmt.Sprintf("<p>备注：%s</p>", note)
        }
        body += fmt.Sprintf("<p>操作时间：%s</p>", now)
    case "enable":
        subject = "账号已启用"
        body = fmt.Sprintf("<p>亲爱的用户 %s：</p><p>您的账号已被管理员启用。</p><p>操作时间：%s</p>", local.Name, now)
    case "disable":
        subject = "账号已禁用"
        body = fmt.Sprintf("<p>亲爱的用户 %s：</p><p>您的账号已被管理员禁用。</p><p>操作时间：%s</p>", local.Name, now)
    case "delete":
        subject = "账号已删除"
        body = fmt.Sprintf("<p>亲爱的用户 %s：</p><p>您的账号已被管理员删除。</p><p>操作时间：%s</p>", local.Name, now)
    default:
        return
    }

    if strings.TrimSpace(ctx.ServerName) != "" {
        body += fmt.Sprintf("<p>服务器：%s</p>", ctx.ServerName)
    }
    body += "<p>如非本人操作，请及时联系管理员。</p>"

    _ = sendEmail(ctx.Settings, local.Email, subject, body)
}

func sendEmail(settings EffectiveSettings, to, subject, body string) error {
    host := strings.TrimSpace(settings.SMTPHost)
    user := strings.TrimSpace(settings.SMTPUser)
    pass := settings.SMTPPass
    if host == "" || user == "" || strings.TrimSpace(pass) == "" {
        return errors.New("未配置 SMTP 信息")
    }

    secure := fallback(settings.SMTPSecure, "ssl")
    port := settings.SMTPPort
    if port <= 0 {
        if secure == "ssl" {
            port = 465
        } else {
            port = 587
        }
    }

    from := strings.TrimSpace(settings.SMTPFrom)
    if from == "" {
        from = user
    }

    addr := net.JoinHostPort(host, strconv.Itoa(port))

    var c *smtp.Client
    var conn net.Conn
    var err error

    if secure == "ssl" {
        tlsConn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
        if err != nil {
            return err
        }
        conn = tlsConn
        c, err = smtp.NewClient(tlsConn, host)
        if err != nil {
            _ = conn.Close()
            return err
        }
    } else {
        conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
        if err != nil {
            return err
        }
        c, err = smtp.NewClient(conn, host)
        if err != nil {
            _ = conn.Close()
            return err
        }
        if secure == "tls" {
            if ok, _ := c.Extension("STARTTLS"); ok {
                if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
                    _ = c.Close()
                    _ = conn.Close()
                    return err
                }
            }
        }
    }
    defer c.Close()
    defer conn.Close()

    auth := smtp.PlainAuth("", user, pass, host)
    if ok, _ := c.Extension("AUTH"); ok {
        if err := c.Auth(auth); err != nil {
            return err
        }
    }

    if err := c.Mail(from); err != nil {
        return err
    }
    if err := c.Rcpt(to); err != nil {
        return err
    }

    wc, err := c.Data()
    if err != nil {
        return err
    }

    msg := buildEmailMessage(from, to, subject, body)
    if _, err := wc.Write([]byte(msg)); err != nil {
        _ = wc.Close()
        return err
    }
    if err := wc.Close(); err != nil {
        return err
    }

    return c.Quit()
}

func buildEmailMessage(from, to, subject, body string) string {
    encodedSubject := mime.QEncoding.Encode("UTF-8", subject)
    headers := []string{
        "MIME-Version: 1.0",
        "Content-Type: text/html; charset=UTF-8",
        "From: " + from,
        "To: " + to,
        "Subject: " + encodedSubject,
        "",
        body,
    }
    return strings.Join(headers, "\r\n")
}

func addSecurityHeaders(w http.ResponseWriter) {
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("Referrer-Policy", "same-origin")
}

func parseRequestForm(r *http.Request) error {
    ct := strings.ToLower(r.Header.Get("Content-Type"))
    if strings.HasPrefix(ct, "multipart/form-data") {
        return r.ParseMultipartForm(64 << 20)
    }
    return r.ParseForm()
}

func readTail(path string, maxBytes int64) (string, error) {
    f, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer f.Close()

    fi, err := f.Stat()
    if err != nil {
        return "", err
    }

    offset := int64(0)
    if fi.Size() > maxBytes {
        offset = fi.Size() - maxBytes
    }
    if _, err := f.Seek(offset, io.SeekStart); err != nil {
        return "", err
    }

    b, err := io.ReadAll(f)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

func rotateLogFile(logFile string, maxBytes int64, maxFiles int) {
    fi, err := os.Stat(logFile)
    if err != nil || fi.IsDir() {
        return
    }
    if fi.Size() < maxBytes {
        return
    }

    if maxFiles < 1 {
        maxFiles = 1
    }

    last := fmt.Sprintf("%s.%d", logFile, maxFiles)
    _ = os.Remove(last)

    for i := maxFiles - 1; i >= 1; i-- {
        src := fmt.Sprintf("%s.%d", logFile, i)
        dst := fmt.Sprintf("%s.%d", logFile, i+1)
        if _, err := os.Stat(src); err == nil {
            _ = os.Rename(src, dst)
        }
    }

    _ = os.Rename(logFile, logFile+".1")
}

func cleanupOldLogs(logFile string, retentionDays int) {
    if retentionDays < 1 {
        retentionDays = 30
    }

    dir := filepath.Dir(logFile)
    base := filepath.Base(logFile)

    stamp := filepath.Join(dir, ".log_cleanup_stamp_"+base)
    todayStart := time.Now().In(time.Local).Truncate(24 * time.Hour)
    if fi, err := os.Stat(stamp); err == nil {
        if fi.ModTime().After(todayStart) {
            return
        }
    }

    lockFile := filepath.Join(dir, ".log_cleanup_lock_"+base)
    lock, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o644)
    if err != nil {
        return
    }
    defer lock.Close()

    pattern := logFile + ".*"
    matches, _ := filepath.Glob(pattern)
    limit := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
    for _, p := range matches {
        if !regexp.MustCompile(`\.\d+$`).MatchString(p) {
            continue
        }
        if fi, err := os.Stat(p); err == nil {
            if fi.ModTime().Before(limit) {
                _ = os.Remove(p)
            }
        }
    }

    _ = os.WriteFile(stamp, []byte(todayStart.Format(time.RFC3339)), 0o644)
}

func appendFile(path string, data []byte) error {
    f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil {
        return err
    }
    defer f.Close()
    _, err = f.Write(data)
    return err
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(code)
    enc := json.NewEncoder(w)
    enc.SetEscapeHTML(false)
    _ = enc.Encode(payload)
}

func getClientIP(r *http.Request) string {
    remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        remoteHost = r.RemoteAddr
    }

    ip := strings.TrimSpace(remoteHost)
    fwd := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])

    if fwd != "" && isPrivateIP(ip) {
        ip = fwd
    }

    if parsed := net.ParseIP(ip); parsed == nil {
        return "unknown"
    }
    return ip
}

func isPrivateIP(ip string) bool {
    p := net.ParseIP(strings.TrimSpace(ip))
    if p == nil {
        return false
    }
    return p.IsPrivate() || p.IsLoopback() || p.IsLinkLocalUnicast() || p.IsLinkLocalMulticast()
}

func randomHex(n int) string {
    if n <= 0 {
        n = 16
    }
    b := make([]byte, n)
    if _, err := crand.Read(b); err != nil {
        return strconv.FormatInt(time.Now().UnixNano(), 16)
    }
    return hex.EncodeToString(b)
}

func secureCompare(a, b string) bool {
    if len(a) != len(b) {
        return false
    }
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func parseBool(v string, def bool) bool {
    s := strings.TrimSpace(strings.ToLower(v))
    switch s {
    case "1", "true", "yes", "on":
        return true
    case "0", "false", "no", "off":
        return false
    default:
        return def
    }
}

func parseInt(s string, def int) int {
    s = strings.TrimSpace(s)
    if s == "" {
        return def
    }
    n, err := strconv.Atoi(s)
    if err != nil {
        return def
    }
    return n
}

func parseInt64(s string, def int64) int64 {
    s = strings.TrimSpace(s)
    if s == "" {
        return def
    }
    n, err := strconv.ParseInt(s, 10, 64)
    if err != nil {
        return def
    }
    return n
}

func toInt64(v any) int64 {
    switch t := v.(type) {
    case int64:
        return t
    case int:
        return int64(t)
    case float64:
        return int64(t)
    case json.Number:
        n, _ := t.Int64()
        return n
    case string:
        n, _ := strconv.ParseInt(t, 10, 64)
        return n
    default:
        return 0
    }
}

func toInt(v any) int {
    switch t := v.(type) {
    case int:
        return t
    case int64:
        return int(t)
    case float64:
        return int(t)
    case string:
        return parseInt(t, 0)
    case json.Number:
        i, _ := t.Int64()
        return int(i)
    default:
        return 0
    }
}

func fallback(value, def string) string {
    if strings.TrimSpace(value) == "" {
        return def
    }
    return value
}

func safeFileName(name string) string {
    name = regexp.MustCompile(`[\\/:*?"<>|]`).ReplaceAllString(name, "_")
    name = strings.TrimSpace(name)
    if name == "" || name == "." || name == ".." {
        return "unnamed"
    }
    return name
}

func calcDaysLeftValue(expireDate string, now time.Time) (int, bool) {
    expireDate = strings.TrimSpace(expireDate)
    if expireDate == "" || expireDate == appPermanentDate {
        return 0, false
    }

    expireDay, err := time.ParseInLocation("2006-01-02", expireDate, time.Local)
    if err != nil {
        return 0, false
    }

    n := now.In(time.Local)
    todayStart := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.Local)
    expireStart := time.Date(expireDay.Year(), expireDay.Month(), expireDay.Day(), 0, 0, 0, 0, time.Local)
    days := int(expireStart.Sub(todayStart) / (24 * time.Hour))
    return days, true
}

func calcDaysLeft(expireDate string) string {
    if strings.TrimSpace(expireDate) == "" || expireDate == appPermanentDate {
        return "永久"
    }
    days, ok := calcDaysLeftValue(expireDate, time.Now())
    if !ok {
        return ""
    }
    return strconv.Itoa(days)
}

func isPermanentValue(expireDate, daysLeft string) bool {
    expireDate = strings.TrimSpace(expireDate)
    daysLeft = strings.TrimSpace(daysLeft)
    if expireDate != "" && expireDate != appPermanentDate {
        return false
    }
    if expireDate == "" && daysLeft != calcDaysLeft("") {
        return false
    }
    return expireDate == "" || expireDate == appPermanentDate || daysLeft == "永久"
}

func formatEmbyTime(value string) string {
    if strings.TrimSpace(value) == "" {
        return ""
    }
    layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.0000000Z", "2006-01-02T15:04:05Z"}
    for _, layout := range layouts {
        if t, err := time.Parse(layout, value); err == nil {
            return t.In(time.Local).Format("2006-01-02 15:04:05")
        }
    }
    return value
}

func formatEmbyDateOnly(value string) string {
    t := formatEmbyTime(value)
    if len(t) >= 10 {
        return t[:10]
    }
    return "未知"
}

func mapBool(m map[string]any, key string) bool {
    if m == nil {
        return false
    }
    v, ok := m[key]
    if !ok {
        return false
    }
    switch t := v.(type) {
    case bool:
        return t
    case string:
        return parseBool(t, false)
    case float64:
        return t != 0
    case int:
        return t != 0
    default:
        return false
    }
}

func cloneMap(in map[string]any) map[string]any {
    out := map[string]any{}
    for k, v := range in {
        out[k] = v
    }
    return out
}

func chunkStrings(in []string, size int) [][]string {
    if size <= 0 {
        size = 1
    }
    out := make([][]string, 0)
    for i := 0; i < len(in); i += size {
        end := i + size
        if end > len(in) {
            end = len(in)
        }
        out = append(out, in[i:end])
    }
    return out
}

func uniqueStrings(in []string) []string {
    seen := map[string]struct{}{}
    out := make([]string, 0, len(in))
    for _, v := range in {
        if _, ok := seen[v]; ok {
            continue
        }
        seen[v] = struct{}{}
        out = append(out, v)
    }
    return out
}

func reverseChargeHistory(in []ChargeRecord) {
    for i, j := 0, len(in)-1; i < j; i, j = i+1, j-1 {
        in[i], in[j] = in[j], in[i]
    }
}

func md5Hex(text string) string {
    sum := md5.Sum([]byte(text))
    return hex.EncodeToString(sum[:])
}

func strPtr(s string) *string {
    cp := s
    return &cp
}

func intPtr(i int) *int {
    cp := i
    return &cp
}

func firstOrEmpty(in []string) string {
    if len(in) == 0 {
        return ""
    }
    return in[0]
}

func randomIntn(max int) int {
    if max <= 1 {
        return 0
    }
    b := make([]byte, 1)
    if _, err := crand.Read(b); err != nil {
        return int(time.Now().UnixNano() % int64(max))
    }
    return int(b[0]) % max
}

func looksLikeEmail(email string) bool {
    email = strings.TrimSpace(email)
    if email == "" {
        return false
    }
    at := strings.Index(email, "@")
    dot := strings.LastIndex(email, ".")
    return at > 0 && dot > at+1 && dot < len(email)-1
}

func mustJSON(v any) string {
    b, err := json.Marshal(v)
    if err != nil {
        return "{}"
    }
    return string(b)
}

func deleteDir(path string) error {
    if path == "" {
        return nil
    }
    if _, err := os.Stat(path); err != nil {
        return nil
    }
    return os.RemoveAll(path)
}

func initTimezone() {
    tz := os.Getenv("TZ")
    if strings.TrimSpace(tz) == "" {
        tz = appDefaultTZ
    }
    if loc, err := time.LoadLocation(tz); err == nil {
        time.Local = loc
    }
}

func maxInt(a, b int) int {
    if a > b {
        return a
    }
    return b
}
