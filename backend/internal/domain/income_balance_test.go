package domain

import "testing"

func TestIncomeRecipientOwesBeneficiaries(t *testing.T) {
	incomes := []Income{{EarnedBy: "a", AmountMinor: 100000, Splits: []*IncomeSplit{
		{BaseSplit: BaseSplit{UserID: "a", AmountMinor: 50000}},
		{BaseSplit: BaseSplit{UserID: "b", AmountMinor: 50000}},
	}}}
	assert := func(expenses []Expense, settlements []Settlement, wantA, wantB int64) {
		t.Helper()
		got := MemberBalancesWithIncomes(expenses, incomes, settlements)
		if len(got) != 2 || got[0].UserID != "a" || got[0].AmountMinor != wantA || got[1].UserID != "b" || got[1].AmountMinor != wantB {
			t.Fatalf("balances = %#v, want a=%d b=%d", got, wantA, wantB)
		}
	}
	assert(nil, nil, -50000, 50000)
	assert(nil, []Settlement{{FromUserID: "a", ToUserID: "b", AmountMinor: 50000}}, 0, 0)
	assert([]Expense{{PaidBy: "a", AmountMinor: 100000, Splits: []*ExpenseSplit{
		{BaseSplit: BaseSplit{UserID: "a", AmountMinor: 50000}},
		{BaseSplit: BaseSplit{UserID: "b", AmountMinor: 50000}},
	}}}, nil, 0, 0)
}
