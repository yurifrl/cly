package models

import "time"

type CustomerSearchResponse struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Document string `json:"document"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Active   bool   `json:"active"`
	CPF      string `json:"cpf"`
	UserID   int    `json:"user_id"`
	UserName string `json:"user_name"`
}

type VerifyPhoneRequest struct {
	CustomerID  int    `json:"customer_id"`
	PhoneNumber string `json:"phone_number"`
}

type ProcessingSummary struct {
	TotalRecords int
	Successful   int
	Failed       int
}

type ErrorRecord struct {
	CPF   string
	Error string
}

type Customer struct {
	Active              bool       `json:"active"`
	BetForbidden        bool       `json:"bet_forbidden"`
	Birthday            string     `json:"birthday"`
	CountryID           int        `json:"country_id"`
	Cpf                 string     `json:"cpf"`
	DepositForbidden    bool       `json:"deposit_forbidden"`
	Document            string     `json:"document"`
	Email               string     `json:"email"`
	FirstName           string     `json:"first_name"`
	ID                  int        `json:"id"`
	LanguageID          int        `json:"language_id"`
	LastName            string     `json:"last_name"`
	Name                string     `json:"name"`
	Phone               string     `json:"phone"`
	ReferredByID        int        `json:"referred_by_id"`
	ReferredByName      string     `json:"referred_by_name"`
	RegisteredAt        string     `json:"registered_at"`
	Role                int        `json:"role"`
	SelfDeletedAt       *time.Time `json:"self_deleted_at"`
	SelfDeletedUntil    *time.Time `json:"self_deleted_until"`
	StatusUpdatedAt     *time.Time `json:"status_updated_at"`
	TimezoneID          int        `json:"timezone_id"`
	UID                 int        `json:"uid"`
	UserID              int        `json:"user_id"`
	UserName            string     `json:"user_name"`
	WithdrawalForbidden bool       `json:"withdrawal_forbidden"`
}
