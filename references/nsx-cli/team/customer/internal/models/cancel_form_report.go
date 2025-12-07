package models

import (
	"sync"
	"time"
)

type CancelFormReportSummary struct {
	TotalCustomers       int
	CustomersWithBalance int
	TotalBalance         float64
	OnboardingFinished   int
	DifferentCPF         int
	FailedCustomers      int
	StartTime            time.Time
	EndTime              time.Time

	sync.RWMutex
}

func NewReportSummary() *CancelFormReportSummary {
	return &CancelFormReportSummary{}
}

func (s *CancelFormReportSummary) AddCustomerWithBalance() {
	s.Lock()
	s.CustomersWithBalance++
	s.Unlock()
}

func (s *CancelFormReportSummary) AddTotalBalance(balance float64) {
	s.Lock()
	s.TotalBalance += balance
	s.Unlock()
}

func (s *CancelFormReportSummary) AddDifferentCPF() {
	s.Lock()
	s.DifferentCPF++
	s.Unlock()
}

func (s *CancelFormReportSummary) AddOnboardingFinished() {
	s.Lock()
	s.OnboardingFinished++
	s.Unlock()
}
