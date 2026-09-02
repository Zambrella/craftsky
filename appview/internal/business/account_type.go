package business

import "errors"

var ErrInvalidAccountType = errors.New("business: invalid account type")

type AccountType string

const (
	AccountTypeRegular  AccountType = "regular"
	AccountTypeBusiness AccountType = "business"
)

func ParseAccountType(raw string) (AccountType, error) {
	accountType := AccountType(raw)
	switch accountType {
	case AccountTypeRegular, AccountTypeBusiness:
		return accountType, nil
	default:
		return "", ErrInvalidAccountType
	}
}

func ResolveAccountType(stored *AccountType) AccountType {
	if stored == nil {
		return AccountTypeRegular
	}
	return *stored
}
