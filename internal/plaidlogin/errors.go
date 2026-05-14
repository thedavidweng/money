package plaidlogin

import "fmt"

const (
	ErrorBaseConfigMissing           = "BASE_CONFIG_MISSING"
	ErrorNotLoggedIn                 = "NOT_LOGGED_IN"
	ErrorTeamSelectionRequired       = "TEAM_SELECTION_REQUIRED"
	ErrorAPIKeysFetchRequired        = "API_KEYS_FETCH_REQUIRED"
	ErrorDashboardTokenRefreshFailed = "DASHBOARD_TOKEN_REFRESH_FAILED"
	ErrorDashboardContractChanged    = "DASHBOARD_CONTRACT_CHANGED"
	ErrorPlaidDashboardLoginRejected = "PLAID_DASHBOARD_LOGIN_REJECTED"
	ErrorPlaidEnvironmentNotProvided = "PLAID_ENVIRONMENT_NOT_PROVISIONED"
	ErrorReadOnlyViolation           = "READ_ONLY_VIOLATION"
)

type Error struct {
	Code    string
	Message string
	Err     error
}

func (e Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e Error) Unwrap() error {
	return e.Err
}
