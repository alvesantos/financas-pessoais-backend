package domain

// Frequency é a periodicidade de um lançamento fixo.
type Frequency string

const (
	FrequencyDiario    Frequency = "diario"
	FrequencySemanal   Frequency = "semanal"
	FrequencyQuinzenal Frequency = "quinzenal"
	FrequencyMensal    Frequency = "mensal"
	FrequencySemestral Frequency = "semestral"
	FrequencyAnual     Frequency = "anual"
)

var frequencyLabels = map[Frequency]string{
	FrequencyDiario:    "Diário",
	FrequencySemanal:   "Semanal",
	FrequencyQuinzenal: "Quinzenal",
	FrequencyMensal:    "Mensal",
	FrequencySemestral: "Semestral",
	FrequencyAnual:     "Anual",
}

// AllFrequencies na ordem do mais curto para o mais longo.
var AllFrequencies = []Frequency{
	FrequencyDiario,
	FrequencySemanal,
	FrequencyQuinzenal,
	FrequencyMensal,
	FrequencySemestral,
	FrequencyAnual,
}

func (f Frequency) Valid() bool {
	_, ok := frequencyLabels[f]
	return ok
}

func (f Frequency) Label() string {
	return frequencyLabels[f]
}

// stepInDays é o intervalo das frequências contadas em dias. As demais são
// contadas em meses, porque "todo dia 20" não é um número fixo de dias.
func (f Frequency) stepInDays() (int, bool) {
	switch f {
	case FrequencyDiario:
		return 1, true
	case FrequencySemanal:
		return 7, true
	case FrequencyQuinzenal:
		return 14, true
	default:
		return 0, false
	}
}

// stepInMonths é o intervalo das frequências contadas em meses.
func (f Frequency) stepInMonths() (int, bool) {
	switch f {
	case FrequencyMensal:
		return 1, true
	case FrequencySemestral:
		return 6, true
	case FrequencyAnual:
		return 12, true
	default:
		return 0, false
	}
}
