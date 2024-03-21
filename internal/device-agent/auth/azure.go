package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/oauth2"

	"github.com/nais/device/internal/auth"
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

		// We used to use r.Context() here, but a Google Chrome update broke that.
		// It seems that Chrome closes the HTTP connection prematurely, because the context
		// is at this point already canceled.
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
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
		for _, group := range groups.([]any) {
			if group.(string) == auth.NaisDeviceApprovalGroup {
				approvalOK = true
			}
		}

		if !approvalOK {
			http.Redirect(w, r, "https://naisdevice-approval.external.prod-gcp.nav.cloud.nais.io/", http.StatusSeeOther)
			authFlowChan <- &authFlowResponse{Tokens: nil, err: fmt.Errorf("do's and don'ts not accepted, opening https://naisdevice-approval.external.prod-gcp.nav.cloud.nais.io/ in browser")}
			return
		}

		successfulResponse(w, "Successfully authenticated 👌 Close me pls", r.Header.Get("user-agent"))
		authFlowChan <- &authFlowResponse{Tokens: &Tokens{Token: t}, err: nil}
	}
}
