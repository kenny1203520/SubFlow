package domain

import "testing"

func TestIncomeAmountsUsesSavedRoundedAllocation(t *testing.T) {
	income := Income{GroupID: "group", EarnedBy: "a", Currency: CurrencyUSD, AmountMinor: 100, BaseCurrency: CurrencyTWD, BaseAmountMinor: 33, Splits: []*IncomeSplit{{BaseSplit: BaseSplit{UserID: "a", AmountMinor: 50, BaseAmountMinor: 17}}, {BaseSplit: BaseSplit{UserID: "b", AmountMinor: 50, BaseAmountMinor: 16}}}}
	for _, tc := range []struct {
		user        string
		base        bool
		share, cash int64
	}{{"a", false, 50, 100}, {"b", false, 50, 0}, {"a", true, 17, 33}, {"b", true, 16, 0}, {"outsider", true, 0, 0}} {
		share, cash := IncomeAmounts(income, tc.user, tc.base)
		if share != tc.share || cash != tc.cash {
			t.Fatalf("%+v: got %d/%d", tc, share, cash)
		}
	}
	if share, cash := IncomeAmounts(Income{OwnerID: "a", EarnedBy: "a"}, "a", false); share != 0 || cash != 0 {
		t.Fatal("zero income must remain zero")
	}
}
