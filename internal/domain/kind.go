package domain

// Kind é o tipo de um lançamento. Só receita soma saldo; os demais subtraem.
type Kind string

const (
	KindReceita       Kind = "receita"
	KindDespesa       Kind = "despesa"
	KindCartaoCredito Kind = "cartao_credito"
	KindInvestimento  Kind = "investimento"
)

// kindLabels é também a descrição padrão de um lançamento sem descrição.
var kindLabels = map[Kind]string{
	KindReceita:       "Receita",
	KindDespesa:       "Despesa",
	KindCartaoCredito: "Gasto no cartão de crédito",
	KindInvestimento:  "Investimento",
}

// AllKinds na ordem em que a interface os apresenta.
var AllKinds = []Kind{KindReceita, KindDespesa, KindCartaoCredito, KindInvestimento}

func (k Kind) Valid() bool {
	_, ok := kindLabels[k]
	return ok
}

// Label é o nome exibido e a descrição padrão do tipo.
func (k Kind) Label() string {
	return kindLabels[k]
}

// IsIncome diz se o tipo soma saldo.
func (k Kind) IsIncome() bool {
	return k == KindReceita
}

// Signed converte um valor sempre positivo no seu efeito sobre o saldo.
func (k Kind) Signed(amountCents int64) int64 {
	if k.IsIncome() {
		return amountCents
	}
	return -amountCents
}
