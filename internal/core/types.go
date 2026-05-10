package core

type Account struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	InstitutionID    string  `json:"institution_id"`
	Type             string  `json:"type"`
	Subtype          string  `json:"subtype,omitempty"`
	CurrentBalance   float64 `json:"current_balance"`
	AvailableBalance float64 `json:"available_balance,omitempty"`
	Currency         string  `json:"currency"`
	Source           string  `json:"source"`
	UpdatedAt        string  `json:"updated_at,omitempty"`
}

type Transaction struct {
	ID            string  `json:"id"`
	AccountID     string  `json:"account_id"`
	Date          string  `json:"date"`
	Amount        float64 `json:"amount"`
	Name          string  `json:"name"`
	MerchantName  string  `json:"merchant_name,omitempty"`
	Category      string  `json:"category,omitempty"`
	Subcategory   string  `json:"subcategory,omitempty"`
	Pending       bool    `json:"pending"`
	Currency      string  `json:"currency"`
	Source        string  `json:"source"`
	NeedsReview   bool    `json:"needs_review,omitempty"`
	Notes         string  `json:"notes,omitempty"`
	LastChangedAt string  `json:"last_changed_at,omitempty"`
}
