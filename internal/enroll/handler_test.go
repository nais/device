package enroll

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nais/device/internal/token"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type capturingWorker struct {
	request *DeviceRequest
	err     error
}

func (w *capturingWorker) Run(ctx context.Context) error {
	return nil
}

func (w *capturingWorker) Send(ctx context.Context, req *DeviceRequest) (*Response, error) {
	w.request = req
	if w.err != nil {
		return nil, w.err
	}

	return &Response{APIServerGRPCAddress: "test"}, nil
}

func TestHandlerPublicKeyValidationAndNormalization(t *testing.T) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)

	canonical := privateKey.PublicKey().String()
	legacyEncoded := base64.StdEncoding.EncodeToString([]byte(canonical))

	tests := []struct {
		name                string
		publicKey           string
		wantStatus          int
		wantWorkerCalled    bool
		wantStoredPublicKey string
		wantErrBody         string
	}{
		{
			name:                "canonical key",
			publicKey:           canonical,
			wantStatus:          http.StatusOK,
			wantWorkerCalled:    true,
			wantStoredPublicKey: canonical,
		},
		{
			name:                "legacy base64 encoded key",
			publicKey:           legacyEncoded,
			wantStatus:          http.StatusOK,
			wantWorkerCalled:    true,
			wantStoredPublicKey: canonical,
		},
		{
			name:             "invalid key",
			publicKey:        "invalid",
			wantStatus:       http.StatusBadRequest,
			wantWorkerCalled: false,
			wantErrBody:      "invalid wireguard public key: expected canonical key or base64-encoded canonical key\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := &capturingWorker{}
			handler := NewHandler(worker, logrus.NewEntry(logrus.New()))

			payload, err := json.Marshal(&DeviceRequest{
				Platform:           "darwin",
				Serial:             "serial-1",
				WireGuardPublicKey: tt.publicKey,
			})
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
			req = req.WithContext(token.WithEmail(req.Context(), "user@example.com"))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantErrBody != "" {
				assert.Equal(t, tt.wantErrBody, rr.Body.String())
			}

			if tt.wantWorkerCalled {
				require.NotNil(t, worker.request)
				assert.Equal(t, tt.wantStoredPublicKey, worker.request.WireGuardPublicKey)
				assert.Equal(t, "user@example.com", worker.request.Owner)
			} else {
				assert.Nil(t, worker.request)
			}
		})
	}
}
