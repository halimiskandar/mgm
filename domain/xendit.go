package domain

import "time"

type XenditResponse struct {
	ID                        string                  `json:"id"`
	ExternalID                string                  `json:"external_id"`
	UserID                    string                  `json:"user_id"`
	Status                    string                  `json:"status"`
	MerchantName              string                  `json:"merchant_name"`
	MerchantProfilePictureURL string                  `json:"merchant_profile_picture_url"`
	Amount                    int64                   `json:"amount"`
	Description               string                  `json:"description"`
	ExpiryDate                time.Time               `json:"expiry_date"`
	InvoiceURL                string                  `json:"invoice_url"`
	AvailableBanks            []AvailableBank         `json:"available_banks"`
	AvailableRetailOutlets    []AvailableRetailOutlet `json:"available_retail_outlets"`
	AvailableEwallets         []AvailableEwallet      `json:"available_ewallets"`
	AvailableQrCodes          []AvailableQrCode       `json:"available_qr_codes"`
	AvailableDirectDebits     []AvailableDirectDebit  `json:"available_direct_debits"`
	AvailablePaylaters        []AvailablePaylater     `json:"available_paylaters"`
	ShouldExcludeCreditCard   bool                    `json:"should_exclude_credit_card"`
	ShouldSendEmail           bool                    `json:"should_send_email"`
	SuccessRedirectURL        string                  `json:"success_redirect_url"`
	FailureRedirectURL        string                  `json:"failure_redirect_url"`
	Created                   time.Time               `json:"created"`
	Updated                   time.Time               `json:"updated"`
	Currency                  string                  `json:"currency"`
	Items                     []Item                  `json:"items"`
	Customer                  Customer                `json:"customer"`
	Metadata                  Metadata                `json:"metadata"`
}

type AvailableBank struct {
	BankCode          string `json:"bank_code"`
	CollectionType    string `json:"collection_type"`
	TransferAmount    int64  `json:"transfer_amount"`
	BankBranch        string `json:"bank_branch"`
	AccountHolderName string `json:"account_holder_name"`
	IdentityAmount    int64  `json:"identity_amount"`
}

type AvailableDirectDebit struct {
	DirectDebitType string `json:"direct_debit_type"`
}

type AvailableEwallet struct {
	EwalletType string `json:"ewallet_type"`
}

type AvailablePaylater struct {
	PaylaterType string `json:"paylater_type"`
}

type AvailableQrCode struct {
	QrCodeType string `json:"qr_code_type"`
}

type AvailableRetailOutlet struct {
	RetailOutletName string `json:"retail_outlet_name"`
}

type Customer struct {
	Email string `json:"email"`
}

type Item struct {
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
	Price    int64  `json:"price"`
	Category string `json:"category"`
}

type Metadata struct {
	StoreBranch string `json:"store_branch"`
}
