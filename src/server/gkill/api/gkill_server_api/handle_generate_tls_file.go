package gkill_server_api

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleGenerateTLSFile(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GenerateTLSFileRequest{}
	response := &req_res.GenerateTLSFileResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse generate tls to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidGenerateTLSFileResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse generate tls request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidGenerateTLSFileRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	device := auth.Device

	// 管理者権限がなければ弾く
	if !auth.Account.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s", auth.Account.UserID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	certFileName, pemFileName, err := g.getTLSFileNames(device)
	if err != nil {
		err = fmt.Errorf("error at get tls file names: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTLSFileNamesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	certFileName, pemFileName = os.ExpandEnv(certFileName), os.ExpandEnv(pemFileName)
	certFileName, pemFileName = filepath.ToSlash(certFileName), filepath.ToSlash(pemFileName)

	// あったら消す
	if _, err := os.Stat(certFileName); err == nil {
		err := os.Remove(certFileName)
		if err != nil {
			err = fmt.Errorf("error at remove cert file %s: %w", certFileName, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.RemoveCertFileError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	if _, err := os.Stat(pemFileName); err == nil {
		err := os.Remove(pemFileName)
		if err != nil {
			err = fmt.Errorf("error at remove pem file %s: %w", pemFileName, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.RemovePemFileError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	hostStr := "localhost"
	ecdsaCurveStr := ""
	ed25519KeyBool := false
	rsaBitsInt := 2048
	validFromStr := ""
	validForDuration := 365 * 24 * time.Hour
	isCABool := true
	host := &hostStr
	ecdsaCurve := &ecdsaCurveStr
	ed25519Key := &ed25519KeyBool
	rsaBits := &rsaBitsInt
	validFrom := &validFromStr
	validFor := &validForDuration
	isCA := &isCABool
	if len(*host) == 0 {
		slog.Log(r.Context(), gkill_log.Error, "finish Missing required --host parameter")
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	var priv any
	switch *ecdsaCurve {
	case "":
		if *ed25519Key {
			_, priv, err = ed25519.GenerateKey(rand.Reader)
		} else {
			priv, err = rsa.GenerateKey(rand.Reader, *rsaBits)
		}
	case "P224":
		priv, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case "P256":
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "P384":
		priv, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "P521":
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		slog.Log(r.Context(), gkill_log.Error, "finish Unrecognized elliptic", "curve", *ecdsaCurve)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Failed to generate private key", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ECDSA, ED25519 and RSA subject keys should have the DigitalSignature
	// KeyUsage bits set in the x509.Certificate template
	keyUsage := x509.KeyUsageDigitalSignature
	// Only RSA subject keys should have the KeyEncipherment KeyUsage bits set. In
	// the context of TLS this KeyUsage is particular to RSA key exchange and
	// authentication.
	if _, isRSA := priv.(*rsa.PrivateKey); isRSA {
		keyUsage |= x509.KeyUsageKeyEncipherment
	}

	var notBefore time.Time
	if len(*validFrom) == 0 {
		notBefore = time.Now()
	} else {
		notBefore, err = time.Parse("Jan 2 15:04:05 2006", *validFrom)
		if err != nil {
			slog.Log(r.Context(), gkill_log.Error, "finish Failed to parse creation date", "error", err)
			err = fmt.Errorf("error at generate tls files")
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GenerateTLSFilesError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	notAfter := notBefore.Add(*validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Failed to generate serial number", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(*host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if *isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Failed to create certificate", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	parentDirCert, parentDirKey := filepath.Dir(certFileName), filepath.Dir(pemFileName)
	parentDirCert, parentDirKey = filepath.ToSlash(parentDirCert), filepath.ToSlash((parentDirKey))

	err = os.MkdirAll(parentDirCert, os.ModePerm)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Failed to open cert.pem for writing", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	err = os.MkdirAll(parentDirKey, os.ModePerm)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Failed to open cert.pem for writing", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	certOut, err := os.Create(certFileName)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Failed to open cert.pem for writing", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Failed to write data to cert.pem", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := certOut.Close(); err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Error closing cert.pem", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	keyOut, err := os.OpenFile(pemFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Failed to open key.pem for writing", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Unable to marshal private key", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Failed to write data to key.pem", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := keyOut.Close(); err != nil {
		slog.Log(r.Context(), gkill_log.Error, "finish Error closing key.pem", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	message := &message.GkillMessage{
		MessageCode: message.TLSFileCreateSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_CREATE_TLS_FILE_MESSAGE"}),
	}
	response.Messages = append(response.Messages, message)
}
