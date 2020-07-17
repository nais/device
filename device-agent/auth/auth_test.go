package auth_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/nais/device/device-agent/auth"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/stretchr/testify/assert"
)

func TestSessionInfo_Expired(t *testing.T) {
	expired := auth.SessionInfo{Expiry: 1}
	assert.True(t, expired.Expired())

	rc := runtimeconfig.RuntimeConfig{SessionInfo: nil}
	assert.True(t, rc.SessionInfo.Expired())

	valid := auth.SessionInfo{Expiry: time.Now().Unix() + 10}
	assert.False(t, valid.Expired())
	assert.False(t, valid.Expired())
}

func TestRunFlow(t *testing.T) {
	failureURL := "http://127.0.0.1:51800/?error=interaction_required&error_description=AADSTS50105%3a+The+signed+in+user+%27%7bEmailHidden%7d%27+is+not+assigned+to+a+role+for+the+application+%276e45010d-2637-4a40-b91d-d4cbb451fb57%27(nais-device-apiserver).%0d%0aTrace+ID%3a+320db82b-71a0-4520-a16f-f962e19a9000%0d%0aCorrelation+ID%3a+ba8b945c-1cd8-4344-b355-b40c63174248%0d%0aTimestamp%3a+2020-07-14+11%3a00%3a34Z&error_uri=https%3a%2f%2flogin.microsoftonline.com%2ferror%3fcode%3d50105&state=8MuwmgykQzr1FCT2"
	successURL := "http://127.0.0.1:51800/?code=123&state=123"

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	urlOpener := func(url string) func() error {
		return func() error {
			fmt.Println("urlOpener")
			client := &http.Client{
				Timeout: 5 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			r, err := client.Get(url)
			if err != nil {
				err = fmt.Errorf("performing get request: %v", err)
				fmt.Println(err)
				return err
			}
			defer r.Body.Close()

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				err = fmt.Errorf("reading response body: %v", err)
				fmt.Println(err)
				return err
			}

			fmt.Printf("url response: %v\n", string(b))

			return nil
		}
	}

	sessionInfoGetter := func(ctx context.Context, params string) (*auth.SessionInfo, error) {
		fmt.Println("sessioninfogetter")
		return &auth.SessionInfo{
			Key:    "key",
			Expiry: time.Now().Add(1 * time.Minute).Unix(),
		}, nil
	}

	si, err := auth.RunFlow(ctx, urlOpener(failureURL), sessionInfoGetter)
	assert.Error(t, err)
	assert.Nil(t, si)

	si, err = auth.RunFlow(ctx, urlOpener(successURL), sessionInfoGetter)
	assert.NoError(t, err)
	assert.NotNil(t, si.Key)
	assert.Equal(t, "key", si.Key)
}
