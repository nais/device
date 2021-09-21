package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	auth2 "github.com/nais/device/pkg/apiserver/auth"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/device-agent/auth"
	"github.com/nais/device/pkg/pb"

	"github.com/stretchr/testify/assert"
)

func TestSessionInfo_Expired(t *testing.T) {
	expired := &pb.Session{Expiry: timestamppb.New(time.Now().Add(-10 * time.Second))}
	assert.True(t, expired.Expired())

	valid := pb.Session{Expiry: timestamppb.New(time.Now().Add(10 * time.Second))}
	assert.False(t, valid.Expired())
}

func TestRunFlow(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	urlOpener := func(url string, expectedCode int, expectedMsg string) auth.UrlOpener {
		return func() error {
			time.Sleep(1 * time.Second)
			client := &http.Client{
				Timeout: 5 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			r, err := client.Get(url)
			if err != nil {
				err = fmt.Errorf("performing get request: %v", err)
				t.Error(err)
				return err
			}
			defer r.Body.Close()

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				err = fmt.Errorf("reading response body: %v", err)
				t.Error(err)
				return err
			}

			assert.Equal(t, r.StatusCode, expectedCode)
			assert.Contains(t, string(b), expectedMsg)

			return nil
		}
	}

	sessionInfoGetter := func(ctx context.Context, params string) (*pb.Session, error) {
		fmt.Println("sessioninfogetter")
		return &pb.Session{
			Key:    "key",
			Expiry: timestamppb.New(time.Now().Add(1 * time.Minute)),
		}, nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port

	opener := urlOpener(failureUrl(port), http.StatusSeeOther, "https://naisdevice-approval.nais.io")
	si, err := auth.RunFlow(ctx, opener, sessionInfoGetter, listener)
	assert.Error(t, err)
	assert.Nil(t, si)

	listener, err = net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	port = listener.Addr().(*net.TCPAddr).Port
	opener = urlOpener(successUrl(port), http.StatusOK, "Successfully authenticated 👌 Close me pls")
	si, err = auth.RunFlow(ctx, opener, sessionInfoGetter, listener)
	assert.NoError(t, err)
	assert.NotNil(t, si.Key)
	assert.Equal(t, "key", si.Key)
}

func TestMakeSessionInfoGetter(t *testing.T) {
	expectedSessionInfo := &pb.Session{
		Key:    "key",
		Expiry: timestamppb.New(time.Now()),
	}
	expectedParams := "?code=123&state=asd"
	expectedPlatform := "linux"
	expectedSerial := "serial"
	expectedSessionInfo.Expiry.Nanos = 0

	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, request.Header.Get("x-naisdevice-platform"), expectedPlatform)
		assert.Equal(t, request.Header.Get("x-naisdevice-serial"), expectedSerial)
		assert.Equal(t, request.URL.RawQuery, expectedParams)

		err := json.NewEncoder(writer).Encode(auth2.LegacySessionFromProtobuf(expectedSessionInfo))
		assert.NoError(t, err)
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	sessionInfoGetter := auth.MakeSessionInfoGetter(s.URL, "linux", "serial", 51800)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	si, err := sessionInfoGetter(ctx, expectedParams)

	assert.NoError(t, err)
	assert.NotNil(t, si)
	assert.Equal(t, expectedSessionInfo.Key, si.Key)
	assert.Equal(t, expectedSessionInfo.Expiry, si.Expiry)
}

func successUrl(port int) string {
	localUrl := fmt.Sprintf("http://127.0.0.1:%d", port)
	return localUrl + "/?code=123&state=123"
}

func failureUrl(port int) string {
	localUrl := fmt.Sprintf("http://127.0.0.1:%d", port)
	return localUrl + "/?error=interaction_required&error_description=AADSTS50105%3a+The+signed+in+user+%27%7bEmailHidden%7d%27+is+not+assigned+to+a+role+for+the+application+%276e45010d-2637-4a40-b91d-d4cbb451fb57%27(nais-device-apiserver).%0d%0aTrace+ID%3a+320db82b-71a0-4520-a16f-f962e19a9000%0d%0aCorrelation+ID%3a+ba8b945c-1cd8-4344-b355-b40c63174248%0d%0aTimestamp%3a+2020-07-14+11%3a00%3a34Z&error_uri=https%3a%2f%2flogin.microsoftonline.com%2ferror%3fcode%3d50105&state=8MuwmgykQzr1FCT2"
}
