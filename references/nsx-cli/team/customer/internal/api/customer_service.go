package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/NSXBet/nsx-cli/shared/interact"
	"github.com/NSXBet/nsx-cli/team/customer/customercfg"
	"github.com/NSXBet/nsx-cli/team/customer/internal/models"
)

type CustomerServiceClient struct {
	config map[string]string
	client *http.Client
}

func NewCustomerServiceClient(cfg map[string]string) *CustomerServiceClient {
	return &CustomerServiceClient{
		config: cfg,
		client: &http.Client{},
	}
}

func (c *CustomerServiceClient) SearchCustomerByCPF(cpf string) (models.CustomerSearchResponse, error) {
	payload := map[string]string{"cpf": cpf}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return models.CustomerSearchResponse{}, err
	}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/admin/customers/search-by-identifier", c.config[customercfg.HostKey]),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return models.CustomerSearchResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Token", c.config[customercfg.TokenKey])

	resp, err := c.client.Do(req)
	if err != nil {
		return models.CustomerSearchResponse{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			interact.Error("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return models.CustomerSearchResponse{}, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	var response models.CustomerSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return models.CustomerSearchResponse{}, err
	}

	return response, nil
}

func (c *CustomerServiceClient) VerifyPhone(customerID int, phoneNumber string) error {
	payload := models.VerifyPhoneRequest{
		CustomerID:  customerID,
		PhoneNumber: phoneNumber,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/admin/verify-phone", c.config[customercfg.HostKey]),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Token", c.config[customercfg.TokenKey])

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			interact.Error("Failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"API returned status code: %d | body: %s | customer_id: %d | phone_number: %s",
			resp.StatusCode,
			string(body),
			customerID,
			phoneNumber,
		)
	}

	return nil
}

func (c *CustomerServiceClient) GetCustomerByID(customerID string) (models.Customer, error) {
	interact.Debug("Getting customer by ID: %s", customerID)

	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/admin/customers/id/%s", c.config[customercfg.HostKey], customerID),
		nil)
	if err != nil {
		return models.Customer{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Token", c.config[customercfg.TokenKey])

	resp, err := c.client.Do(req)
	if err != nil {
		return models.Customer{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			interact.Error("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return models.Customer{}, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	var response models.Customer
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return models.Customer{}, err
	}

	return response, nil
}
