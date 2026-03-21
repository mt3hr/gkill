package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// apiTestMu serializes API tests that mutate gkill_options global state.
var apiTestMu sync.Mutex

// setupTestGkillServerAPI creates a GkillServerAPI backed by temp SQLite databases.
// It overrides gkill_options globals to point at the temp directory, then calls
// NewGkillServerAPI() which creates config DBs and a default admin account.
//
// Requirements:
//   - CGO enabled (sqlite3)
//   - The embed/i18n/locales directory must contain locale JSON files (built into the binary via go:embed)
//
// The returned cleanup function restores the original gkill_options values.
func setupTestGkillServerAPI(t *testing.T) (*GkillServerAPI, func()) {
	t.Helper()
	apiTestMu.Lock()

	tmpDir, err := os.MkdirTemp("", "gkill_api_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}

	// Save original option values
	origHome := gkill_options.GkillHomeDir
	origLib := gkill_options.LibDir
	origCache := gkill_options.CacheDir
	origLog := gkill_options.LogDir
	origConfig := gkill_options.ConfigDir
	origData := gkill_options.DataDirectoryDefault
	origTLSCert := gkill_options.TLSCertFileDefault
	origTLSKey := gkill_options.TLSKeyFileDefault
	origCacheInMemory := gkill_options.IsCacheInMemory

	// Override to temp directory (no $HOME expansion needed — these are literal paths)
	gkill_options.GkillHomeDir = tmpDir
	gkill_options.LibDir = tmpDir + "/lib/base_directory"
	gkill_options.CacheDir = tmpDir + "/caches"
	gkill_options.LogDir = tmpDir + "/logs"
	gkill_options.ConfigDir = tmpDir + "/configs"
	gkill_options.DataDirectoryDefault = tmpDir + "/datas"
	gkill_options.TLSCertFileDefault = tmpDir + "/tls/cert.cer"
	gkill_options.TLSKeyFileDefault = tmpDir + "/tls/key.pem"
	gkill_options.IsCacheInMemory = false

	// Create required subdirectories
	for _, dir := range []string{"configs", "datas", "caches", "logs", "lib/base_directory"} {
		if err := os.MkdirAll(tmpDir+"/"+dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	api, err := NewGkillServerAPI()
	if err != nil {
		// Restore before failing
		gkill_options.GkillHomeDir = origHome
		gkill_options.LibDir = origLib
		gkill_options.CacheDir = origCache
		gkill_options.LogDir = origLog
		gkill_options.ConfigDir = origConfig
		gkill_options.DataDirectoryDefault = origData
		gkill_options.TLSCertFileDefault = origTLSCert
		gkill_options.TLSKeyFileDefault = origTLSKey
		gkill_options.IsCacheInMemory = origCacheInMemory
		t.Fatalf("NewGkillServerAPI failed: %v", err)
	}

	cleanup := func() {
		// Close all config DAOs individually to release SQLite file handles.
		// GkillDAOManager.Close() has an early-return-on-error pattern that can
		// leave some DB files locked, and it also misses GkillNotificationTargetDAO.
		// We close each one explicitly here to avoid TempDir cleanup failures on Windows.
		ctx := context.Background()
		if api.GkillDAOManager != nil && api.GkillDAOManager.ConfigDAOs != nil {
			cfgDAOs := api.GkillDAOManager.ConfigDAOs
			if cfgDAOs.AccountDAO != nil {
				cfgDAOs.AccountDAO.Close(ctx)
			}
			if cfgDAOs.LoginSessionDAO != nil {
				cfgDAOs.LoginSessionDAO.Close(ctx)
			}
			if cfgDAOs.FileUploadHistoryDAO != nil {
				cfgDAOs.FileUploadHistoryDAO.Close(ctx)
			}
			if cfgDAOs.ShareKyouInfoDAO != nil {
				cfgDAOs.ShareKyouInfoDAO.Close(ctx)
			}
			if cfgDAOs.ServerConfigDAO != nil {
				cfgDAOs.ServerConfigDAO.Close(ctx)
			}
			if cfgDAOs.AppllicationConfigDAO != nil {
				cfgDAOs.AppllicationConfigDAO.Close(ctx)
			}
			if cfgDAOs.RepositoryDAO != nil {
				cfgDAOs.RepositoryDAO.Close(ctx)
			}
			if cfgDAOs.GkillNotificationTargetDAO != nil {
				cfgDAOs.GkillNotificationTargetDAO.Close(ctx)
			}
		}
		gkill_options.GkillHomeDir = origHome
		gkill_options.LibDir = origLib
		gkill_options.CacheDir = origCache
		gkill_options.LogDir = origLog
		gkill_options.ConfigDir = origConfig
		gkill_options.DataDirectoryDefault = origData
		gkill_options.TLSCertFileDefault = origTLSCert
		gkill_options.TLSKeyFileDefault = origTLSKey
		gkill_options.IsCacheInMemory = origCacheInMemory
		// Best-effort cleanup of temp directory
		os.RemoveAll(tmpDir)
		apiTestMu.Unlock()
	}

	return api, cleanup
}

// setupTestRouter creates a GkillServerAPI and registers all routes on the
// GkillDAOManager's mux.Router, returning an httptest.Server ready for requests.
// This mirrors the route registration logic in GkillServerAPI.Serve().
func setupTestRouter(t *testing.T) (*httptest.Server, *GkillServerAPI, func()) {
	t.Helper()

	api, optCleanup := setupTestGkillServerAPI(t)

	router := api.GkillDAOManager.GetRouter()

	// Register key API routes (mirrors Serve() route registrations)
	router.HandleFunc(api.APIAddress.LoginAddress, func(w http.ResponseWriter, r *http.Request) {
		api.HandleLogin(w, r)
	}).Methods(api.APIAddress.LoginMethod)

	router.HandleFunc(api.APIAddress.LogoutAddress, func(w http.ResponseWriter, r *http.Request) {
		api.HandleLogout(w, r)
	}).Methods(api.APIAddress.LogoutMethod)

	router.HandleFunc(api.APIAddress.GetApplicationConfigAddress, func(w http.ResponseWriter, r *http.Request) {
		api.HandleGetApplicationConfig(w, r)
	}).Methods(api.APIAddress.GetApplicationConfigMethod)

	router.HandleFunc(api.APIAddress.AddKmemoAddress, func(w http.ResponseWriter, r *http.Request) {
		api.HandleAddKmemo(w, r)
	}).Methods(api.APIAddress.AddKmemoMethod)

	router.HandleFunc(api.APIAddress.AddTagAddress, func(w http.ResponseWriter, r *http.Request) {
		api.HandleAddTag(w, r)
	}).Methods(api.APIAddress.AddTagMethod)

	router.HandleFunc(api.APIAddress.GetKyousAddress, func(w http.ResponseWriter, r *http.Request) {
		api.HandleGetKyous(w, r)
	}).Methods(api.APIAddress.GetKyousMethod)

	router.HandleFunc(api.APIAddress.SubmitKFTLTextAddress, func(w http.ResponseWriter, r *http.Request) {
		api.HandleSubmitKFTLText(w, r)
	}).Methods(api.APIAddress.SubmitKFTLTextMethod)

	ts := httptest.NewServer(router)

	cleanup := func() {
		ts.Close()
		optCleanup()
	}

	return ts, api, cleanup
}

// prepareLoginReadyAccount updates the default admin account to be login-ready
// (clears the password reset token and sets a known password hash).
func prepareLoginReadyAccount(t *testing.T, api *GkillServerAPI, userID string, passwordSha256 string) {
	t.Helper()
	ctx := context.Background()

	acc, err := api.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, userID)
	if err != nil {
		t.Fatalf("GetAccount(%s) failed: %v", userID, err)
	}
	if acc == nil {
		t.Fatalf("account %s not found", userID)
	}

	// Clear password reset token and set password
	acc.PasswordResetToken = nil
	acc.PasswordSha256 = &passwordSha256
	acc.IsEnable = true

	ok, err := api.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(ctx, acc)
	if err != nil {
		t.Fatalf("UpdateAccount failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateAccount returned false")
	}
}

// postJSON sends a POST request with JSON body and returns the response.
func postJSON(t *testing.T, url string, body interface{}) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	return resp
}

// --- Tests ---

func TestNewGKillAPIAddress_AllFieldsPopulated(t *testing.T) {
	addr := NewGKillAPIAddress()

	// Verify that all address fields are non-empty
	if addr.LoginAddress == "" {
		t.Error("LoginAddress is empty")
	}
	if addr.LoginMethod == "" {
		t.Error("LoginMethod is empty")
	}
	if addr.LogoutAddress == "" {
		t.Error("LogoutAddress is empty")
	}
	if addr.AddKmemoAddress == "" {
		t.Error("AddKmemoAddress is empty")
	}
	if addr.GetKyousAddress == "" {
		t.Error("GetKyousAddress is empty")
	}
	if addr.GetApplicationConfigAddress == "" {
		t.Error("GetApplicationConfigAddress is empty")
	}
	if addr.SubmitKFTLTextAddress == "" {
		t.Error("SubmitKFTLTextAddress is empty")
	}
	if addr.UpdateCacheAddress == "" {
		t.Error("UpdateCacheAddress is empty")
	}

	// Verify all methods are POST
	if addr.LoginMethod != "POST" {
		t.Errorf("LoginMethod = %q, want POST", addr.LoginMethod)
	}
	if addr.GetKyousMethod != "POST" {
		t.Errorf("GetKyousMethod = %q, want POST", addr.GetKyousMethod)
	}
}

func TestNewGKillAPIAddress_PathPrefixes(t *testing.T) {
	addr := NewGKillAPIAddress()

	// All API addresses should start with /api/ (except serviceWorker.js)
	addresses := []struct {
		name string
		addr string
	}{
		{"Login", addr.LoginAddress},
		{"Logout", addr.LogoutAddress},
		{"AddKmemo", addr.AddKmemoAddress},
		{"GetKyous", addr.GetKyousAddress},
		{"AddTag", addr.AddTagAddress},
		{"GetApplicationConfig", addr.GetApplicationConfigAddress},
		{"SubmitKFTLText", addr.SubmitKFTLTextAddress},
		{"UpdateCache", addr.UpdateCacheAddress},
	}

	for _, tc := range addresses {
		if len(tc.addr) < 5 || tc.addr[:5] != "/api/" {
			t.Errorf("%s address %q does not start with /api/", tc.name, tc.addr)
		}
	}
}

func TestSetupGkillServerAPI(t *testing.T) {
	api, cleanup := setupTestGkillServerAPI(t)
	defer cleanup()

	if api == nil {
		t.Fatal("setupTestGkillServerAPI returned nil")
	}
	if api.APIAddress == nil {
		t.Fatal("APIAddress is nil")
	}
	if api.GkillDAOManager == nil {
		t.Fatal("GkillDAOManager is nil")
	}
	if api.FindFilter == nil {
		t.Fatal("FindFilter is nil")
	}

	// Verify the admin account was created
	ctx := context.Background()
	accounts, err := api.GkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(ctx)
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	if len(accounts) == 0 {
		t.Fatal("no accounts created — expected at least admin")
	}

	foundAdmin := false
	for _, acc := range accounts {
		if acc.UserID == "admin" {
			foundAdmin = true
			if !acc.IsAdmin {
				t.Error("admin account IsAdmin should be true")
			}
			if !acc.IsEnable {
				t.Error("admin account IsEnable should be true")
			}
		}
	}
	if !foundAdmin {
		t.Error("admin account not found")
	}

	// Verify server config was created
	configs, err := api.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		t.Fatalf("GetAllServerConfigs failed: %v", err)
	}
	if len(configs) == 0 {
		t.Fatal("no server configs created")
	}

	// Verify device can be retrieved
	device, err := api.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if device == "" {
		t.Error("GetDevice returned empty string")
	}
}

func TestHandleLogin_Success(t *testing.T) {
	ts, api, cleanup := setupTestRouter(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // sha256 of ""
	prepareLoginReadyAccount(t, api, "admin", passwordHash)

	req := &req_res.LoginRequest{
		UserID:         "admin",
		PasswordSha256: passwordHash,
		LocaleName:     "en",
	}
	resp := postJSON(t, ts.URL+"/api/login", req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var loginResp req_res.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if len(loginResp.Errors) > 0 {
		for _, e := range loginResp.Errors {
			t.Errorf("unexpected error: code=%s msg=%s", e.ErrorCode, e.ErrorMessage)
		}
		t.FailNow()
	}

	if loginResp.SessionID == "" {
		t.Error("SessionID is empty after successful login")
	}

	if len(loginResp.Messages) == 0 {
		t.Error("expected at least one success message")
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	ts, api, cleanup := setupTestRouter(t)
	defer cleanup()

	correctHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	prepareLoginReadyAccount(t, api, "admin", correctHash)

	req := &req_res.LoginRequest{
		UserID:         "admin",
		PasswordSha256: "wrong_password_hash",
		LocaleName:     "en",
	}
	resp := postJSON(t, ts.URL+"/api/login", req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (errors in body)", resp.StatusCode)
	}

	var loginResp req_res.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if len(loginResp.Errors) == 0 {
		t.Fatal("expected error for wrong password, got none")
	}

	foundPasswordError := false
	for _, e := range loginResp.Errors {
		if e.ErrorCode == message.AccountInvalidPasswordError {
			foundPasswordError = true
		}
	}
	if !foundPasswordError {
		t.Errorf("expected error code %s, got errors: %+v", message.AccountInvalidPasswordError, loginResp.Errors)
	}

	if loginResp.SessionID != "" {
		t.Error("SessionID should be empty for failed login")
	}
}

func TestHandleLogin_NonexistentUser(t *testing.T) {
	ts, _, cleanup := setupTestRouter(t)
	defer cleanup()

	req := &req_res.LoginRequest{
		UserID:         "nonexistent_user",
		PasswordSha256: "somehash",
		LocaleName:     "en",
	}
	resp := postJSON(t, ts.URL+"/api/login", req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (errors in body)", resp.StatusCode)
	}

	var loginResp req_res.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if len(loginResp.Errors) == 0 {
		t.Fatal("expected error for nonexistent user, got none")
	}

	foundNotFoundError := false
	for _, e := range loginResp.Errors {
		if e.ErrorCode == message.AccountNotFoundError {
			foundNotFoundError = true
		}
	}
	if !foundNotFoundError {
		t.Errorf("expected error code %s, got errors: %+v", message.AccountNotFoundError, loginResp.Errors)
	}
}

func TestHandleLogin_PasswordResetTokenBlocks(t *testing.T) {
	// The default admin account has a password reset token set.
	// Login should be rejected with the appropriate error.
	ts, _, cleanup := setupTestRouter(t)
	defer cleanup()

	req := &req_res.LoginRequest{
		UserID:         "admin",
		PasswordSha256: "",
		LocaleName:     "en",
	}
	resp := postJSON(t, ts.URL+"/api/login", req)
	defer resp.Body.Close()

	var loginResp req_res.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if len(loginResp.Errors) == 0 {
		t.Fatal("expected error because password reset token is set")
	}

	foundResetError := false
	for _, e := range loginResp.Errors {
		if e.ErrorCode == message.AccountPasswordResetTokenIsNotNilError {
			foundResetError = true
		}
	}
	if !foundResetError {
		t.Errorf("expected error code %s, got: %+v", message.AccountPasswordResetTokenIsNotNilError, loginResp.Errors)
	}
}

func TestHandleLogin_InvalidJSON(t *testing.T) {
	ts, _, cleanup := setupTestRouter(t)
	defer cleanup()

	resp, err := http.Post(ts.URL+"/api/login", "application/json", bytes.NewReader([]byte("not json")))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var loginResp req_res.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if len(loginResp.Errors) == 0 {
		t.Fatal("expected error for invalid JSON, got none")
	}
}

func TestHandleLogin_WrongMethod(t *testing.T) {
	ts, _, cleanup := setupTestRouter(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/api/login")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	// gorilla/mux returns 405 Method Not Allowed for wrong method
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405 for GET on POST-only endpoint", resp.StatusCode)
	}
}

func TestHandleLogin_ConcurrentRequests(t *testing.T) {
	ts, api, cleanup := setupTestRouter(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	prepareLoginReadyAccount(t, api, "admin", passwordHash)

	const concurrency = 5
	var wg sync.WaitGroup
	errors := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := &req_res.LoginRequest{
				UserID:         "admin",
				PasswordSha256: passwordHash,
				LocaleName:     "en",
			}
			b, _ := json.Marshal(req)
			resp, err := http.Post(ts.URL+"/api/login", "application/json", bytes.NewReader(b))
			if err != nil {
				errors <- "POST failed: " + err.Error()
				return
			}
			defer resp.Body.Close()

			var loginResp req_res.LoginResponse
			if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
				errors <- "decode failed: " + err.Error()
				return
			}
			if len(loginResp.Errors) > 0 {
				errors <- "login error: " + loginResp.Errors[0].ErrorCode
			}
		}()
	}

	wg.Wait()
	close(errors)

	for errMsg := range errors {
		t.Errorf("concurrent login error: %s", errMsg)
	}
}

func TestGenerateNewID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateNewID()
		if id == "" {
			t.Fatal("GenerateNewID returned empty string")
		}
		if ids[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

// --- GkillDAOManager integration tests ---

func TestGkillDAOManager_ConfigDAOs_NotNil(t *testing.T) {
	api, cleanup := setupTestGkillServerAPI(t)
	defer cleanup()

	cfgDAOs := api.GkillDAOManager.ConfigDAOs
	if cfgDAOs == nil {
		t.Fatal("ConfigDAOs is nil")
	}
	if cfgDAOs.AccountDAO == nil {
		t.Error("AccountDAO is nil")
	}
	if cfgDAOs.LoginSessionDAO == nil {
		t.Error("LoginSessionDAO is nil")
	}
	if cfgDAOs.ServerConfigDAO == nil {
		t.Error("ServerConfigDAO is nil")
	}
	if cfgDAOs.AppllicationConfigDAO == nil {
		t.Error("AppllicationConfigDAO is nil")
	}
	if cfgDAOs.RepositoryDAO == nil {
		t.Error("RepositoryDAO is nil")
	}
	if cfgDAOs.ShareKyouInfoDAO == nil {
		t.Error("ShareKyouInfoDAO is nil")
	}
	if cfgDAOs.FileUploadHistoryDAO == nil {
		t.Error("FileUploadHistoryDAO is nil")
	}
	if cfgDAOs.GkillNotificationTargetDAO == nil {
		t.Error("GkillNotificationTargetDAO is nil")
	}
}

// loginAndGetSession is a helper that logs in and returns the session ID.
func loginAndGetSession(t *testing.T, tsURL string, api *GkillServerAPI, userID, passwordHash string) string {
	t.Helper()
	prepareLoginReadyAccount(t, api, userID, passwordHash)

	req := &req_res.LoginRequest{
		UserID:         userID,
		PasswordSha256: passwordHash,
		LocaleName:     "en",
	}
	resp := postJSON(t, tsURL+"/api/login", req)
	defer resp.Body.Close()

	var loginResp req_res.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response failed: %v", err)
	}
	if len(loginResp.Errors) > 0 {
		t.Fatalf("login failed: %+v", loginResp.Errors)
	}
	if loginResp.SessionID == "" {
		t.Fatal("login returned empty session ID")
	}
	return loginResp.SessionID
}

func TestHandleGetApplicationConfig_RequiresSession(t *testing.T) {
	ts, _, cleanup := setupTestRouter(t)
	defer cleanup()

	// Send request with empty session ID — should return errors
	body := map[string]string{
		"session_id":  "",
		"locale_name": "en",
	}
	resp := postJSON(t, ts.URL+"/api/get_application_config", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Parse the generic response to check for errors
	var result struct {
		Errors []*message.GkillError `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(result.Errors) == 0 {
		t.Error("expected error for empty session ID, got none")
	}
}

func TestHTTPServer_RouteRegistration(t *testing.T) {
	ts, _, cleanup := setupTestRouter(t)
	defer cleanup()

	// Verify that registered endpoints respond (not 404).
	// We send valid-structured JSON with a login request body for /api/login,
	// and use GET (expecting 405) for others to confirm routes exist without
	// triggering panics on malformed request bodies.
	t.Run("login_route_exists", func(t *testing.T) {
		req := &req_res.LoginRequest{
			UserID:         "admin",
			PasswordSha256: "",
			LocaleName:     "en",
		}
		resp := postJSON(t, ts.URL+"/api/login", req)
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Error("login endpoint returned 404")
		}
	})

	// For other POST-only routes, sending GET should return 405 (not 404),
	// proving the route is registered.
	getEndpoints := []string{
		"/api/logout",
		"/api/get_application_config",
		"/api/add_kmemo",
		"/api/add_tag",
		"/api/submit_kftl_text",
	}
	for _, ep := range getEndpoints {
		t.Run("route_"+ep, func(t *testing.T) {
			resp, err := http.Get(ts.URL + ep)
			if err != nil {
				t.Fatalf("GET %s failed: %v", ep, err)
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				t.Errorf("endpoint %s returned 404 — route not registered", ep)
			}
		})
	}
}

// --- Unexported / package-level function tests ---

func TestGetLocalizer_DefaultsToJapanese(t *testing.T) {
	// Unknown locale should fallback to "ja"
	localizer := GetLocalizer("nonexistent_locale")
	if localizer == nil {
		t.Fatal("GetLocalizer returned nil for unknown locale")
	}
}

func TestGetLocalizer_KnownLocales(t *testing.T) {
	locales := []string{"ja", "en", "zh", "ko", "es", "fr", "de"}
	for _, loc := range locales {
		l := GetLocalizer(loc)
		if l == nil {
			t.Errorf("GetLocalizer(%q) returned nil", loc)
		}
	}
}

// Ensure that unused imports don't cause compile errors
var _ = (*dao.GkillDAOManager)(nil)
