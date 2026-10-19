package domain

// IncomeAmounts separates beneficial ownership from money held by the recipient.
// Base amounts use the saved conversion and allocation, including rounding.
func IncomeAmounts(income Income, userID string, base bool) (share, received int64) {
	amount := income.AmountMinor
	if base {
		amount = income.BaseAmountMinor
	}
	if income.EarnedBy == userID {
		received = amount
	}
	if income.GroupID == "" && income.OwnerID == userID {
		return amount, received
	}
	for _, split := range income.Splits {
		if split.UserID == userID {
			if base {
				share += split.BaseAmountMinor
			} else {
				share += split.AmountMinor
			}
		}
	}
	return share, received
}
