package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/pkg/auth"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/oauth2"
)

func handleRedirectAzure(state string, conf oauth2.Config, codeVerifier *codeverifier.CodeVerifier, authFlowChan chan *authFlowResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Catch if user has not approved terms
		responseState := r.URL.Query().Get("state")
		if state != responseState {
			failAuth(fmt.Errorf("invalid 'state' in auth response, try again"), w, authFlowChan)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			failAuth(fmt.Errorf("could not find 'code' URL query parameter"), w, authFlowChan)
			return
		}

		ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(30*time.Second))
		defer cancel()

		codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier.String())
		t, err := conf.Exchange(ctx, code, codeVerifierParam)
		if err != nil {
			failAuth(fmt.Errorf("exchanging code for tokens: %w", err), w, authFlowChan)
			return
		}

		parsedToken, err := jwt.Parse([]byte(t.AccessToken))
		if err != nil {
			failAuth(fmt.Errorf("parsing authFlowResponse: %v", err), w, authFlowChan)
			return
		}

		groups, ok := parsedToken.Get("groups")
		if !ok {
			failAuth(fmt.Errorf("no groups found in authFlowResponse"), w, authFlowChan)
			return
		}

		approvalOK := false
		for _, group := range groups.([]interface{}) {
			if group.(string) == auth.NaisDeviceApprovalGroup {
				approvalOK = true
			}
		}

		if !approvalOK {
			http.Redirect(w, r, "https://naisdevice-approval.nais.io/", http.StatusSeeOther)
			authFlowChan <- &authFlowResponse{Token: nil, err: fmt.Errorf("do's and don'ts not accepted, opening https://naisdevice-approval.nais.io/ in browser")}
			return
		}

		token := &Token{
			Token:  t.AccessToken,
			Expiry: t.Expiry,
		}

		successfulResponse(w, "Successfully authenticated 👌 Close me pls")
		authFlowChan <- &authFlowResponse{Token: token, err: nil}
	}
}
