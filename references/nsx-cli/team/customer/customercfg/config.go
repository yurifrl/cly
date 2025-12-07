package customercfg

import (
	"fmt"

	"github.com/NSXBet/nsx-cli/shared/awssdk"
	"github.com/NSXBet/nsx-cli/shared/skin"
)

const (
	tokenSecretNameMask = "%s/customer-service/customer-service-%s"
	tokenSecretRegion   = "us-east-1"
	HostKey             = "HOST"
	TokenKey            = "CUSTOMER_SERVICE_INTERNAL_API_TOKEN"
)

func GetCustomerServiceConfig() (map[string]string, error) {
	skinEnum, env := skin.GetSkinAndEnv()
	skinName := skinEnum.String()
	if skinEnum == skin.Betdev {
		skinName = "webdevelopers"
	}
	tokenSecretName := fmt.Sprintf(tokenSecretNameMask, env, skinName)

	secretMap, err := awssdk.GetSecretMapWithRegion(tokenSecretName, tokenSecretRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer service config: %w", err)
	}

	switch skinEnum {
	case skin.Betdev:
		secretMap[HostKey] = "https://prod-webdevelopers-customer.nsx.services"
	case skin.Betnacional:
		secretMap[HostKey] = "https://prod-betnacional-customer.nsx.services"
	case skin.Mrjackbets:
		secretMap[HostKey] = "https://prod-mrjack-customer.nsx.services"
	case skin.Mundialbet:
		secretMap[HostKey] = "https://homol-mundialbet-customer.nsx.services"
	default:
		return nil, fmt.Errorf("invalid skin: %s", skinEnum)
	}

	return secretMap, nil
}
