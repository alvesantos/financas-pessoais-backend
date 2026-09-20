package domain_test

import (
	"testing"
	"time"

	"github.com/alvesantos/financas-backend/internal/domain"
)

func data(ano int, mes time.Month, dia int) time.Time {
	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

func dias(datas []time.Time) []int {
	out := make([]int, len(datas))
	for i, d := range datas {
		out[i] = d.Day()
	}
	return out
}

func iguais(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func fixo(frequencia domain.Frequency, inicio time.Time) domain.RecurringEntry {
	return domain.RecurringEntry{
		ID:          1,
		UserID:      1,
		Description: "Academia",
		AmountCents: 15990,
		Kind:        domain.KindDespesa,
		Frequency:   frequencia,
		StartDate:   inicio,
		Active:      true,
	}
}

func TestOcorrenciasMensais(t *testing.T) {
	// "Academia, todo dia 20": cai uma vez por mês, no dia 20.
	entrada := fixo(domain.FrequencyMensal, data(2026, time.January, 20))

	ocorrencias := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.September))

	if got := dias(ocorrencias); !iguais(got, []int{20}) {
		t.Errorf("dias = %v, esperava [20]", got)
	}
}

func TestOcorrenciaMensalNaoAparecemAntesDoInicio(t *testing.T) {
	entrada := fixo(domain.FrequencyMensal, data(2026, time.October, 20))

	if got := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.September)); len(got) != 0 {
		t.Errorf("esperava nenhuma ocorrência antes do início, veio %v", dias(got))
	}
}

func TestOcorrenciaMensalPrendeDia31AoUltimoDiaDoMes(t *testing.T) {
	// Um fixo do dia 31 não pode vazar para o dia 1º do mês seguinte.
	entrada := fixo(domain.FrequencyMensal, data(2026, time.January, 31))

	casos := map[time.Month]int{
		time.February: 28,
		time.April:    30,
		time.March:    31,
	}

	for mes, esperado := range casos {
		ocorrencias := entrada.OccurrencesIn(domain.MonthPeriod(2026, mes))

		if got := dias(ocorrencias); !iguais(got, []int{esperado}) {
			t.Errorf("%v: dias = %v, esperava [%d]", mes, got, esperado)
		}
	}
}

func TestOcorrenciasDiarias(t *testing.T) {
	entrada := fixo(domain.FrequencyDiario, data(2026, time.September, 28))

	ocorrencias := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.September))

	if got := dias(ocorrencias); !iguais(got, []int{28, 29, 30}) {
		t.Errorf("dias = %v, esperava [28 29 30]", got)
	}
}

func TestOcorrenciasSemanais(t *testing.T) {
	entrada := fixo(domain.FrequencySemanal, data(2026, time.September, 3))

	ocorrencias := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.September))

	if got := dias(ocorrencias); !iguais(got, []int{3, 10, 17, 24}) {
		t.Errorf("dias = %v, esperava [3 10 17 24]", got)
	}
}

func TestOcorrenciasQuinzenaisAtravessamOMes(t *testing.T) {
	// O ciclo continua de onde parou: o mês seguinte não recomeça no dia 1.
	entrada := fixo(domain.FrequencyQuinzenal, data(2026, time.September, 25))

	setembro := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.September))
	outubro := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.October))

	if got := dias(setembro); !iguais(got, []int{25}) {
		t.Errorf("setembro = %v, esperava [25]", got)
	}
	if got := dias(outubro); !iguais(got, []int{9, 23}) {
		t.Errorf("outubro = %v, esperava [9 23]", got)
	}
}

func TestOcorrenciasSemestrais(t *testing.T) {
	entrada := fixo(domain.FrequencySemestral, data(2026, time.March, 10))

	casos := map[time.Month][]int{
		time.March:     {10},
		time.September: {10},
		time.June:      nil,
	}

	for mes, esperado := range casos {
		got := dias(entrada.OccurrencesIn(domain.MonthPeriod(2026, mes)))

		if !iguais(got, esperado) {
			t.Errorf("%v: dias = %v, esperava %v", mes, got, esperado)
		}
	}
}

func TestOcorrenciasAnuais(t *testing.T) {
	entrada := fixo(domain.FrequencyAnual, data(2024, time.September, 15))

	emSetembro := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.September))
	emOutubro := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.October))

	if got := dias(emSetembro); !iguais(got, []int{15}) {
		t.Errorf("setembro = %v, esperava [15]", got)
	}
	if len(emOutubro) != 0 {
		t.Errorf("outubro deveria ficar vazio, veio %v", dias(emOutubro))
	}
}

func TestFixoEncerradoNaoProjetaDepoisDoFim(t *testing.T) {
	fim := data(2026, time.September, 15)
	entrada := fixo(domain.FrequencyMensal, data(2026, time.January, 20))
	entrada.EndDate = &fim

	if got := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.September)); len(got) != 0 {
		t.Errorf("esperava nenhuma ocorrência depois do fim, veio %v", dias(got))
	}

	// Antes do fim continua valendo.
	if got := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.August)); len(got) != 1 {
		t.Errorf("agosto deveria ter 1 ocorrência, veio %v", dias(got))
	}
}

func TestFixoInativoNaoProjeta(t *testing.T) {
	entrada := fixo(domain.FrequencyMensal, data(2026, time.January, 20))
	entrada.Active = false

	if got := entrada.OccurrencesIn(domain.MonthPeriod(2026, time.September)); len(got) != 0 {
		t.Errorf("fixo inativo não deveria projetar, veio %v", dias(got))
	}
}

func TestProjecaoCarregaOrigemEFrequencia(t *testing.T) {
	entrada := fixo(domain.FrequencyMensal, data(2026, time.January, 20))

	projetados := entrada.ProjectInto(domain.MonthPeriod(2026, time.September))

	if len(projetados) != 1 {
		t.Fatalf("esperava 1 projeção, veio %d", len(projetados))
	}

	p := projetados[0]
	if !p.IsProjected() {
		t.Error("a projeção precisa se identificar como projetada")
	}
	if *p.RecurringID != entrada.ID {
		t.Errorf("RecurringID = %d, esperava %d", *p.RecurringID, entrada.ID)
	}
	if *p.Frequency != domain.FrequencyMensal {
		t.Errorf("Frequency = %q, esperava %q", *p.Frequency, domain.FrequencyMensal)
	}
	if p.ID != 0 {
		t.Error("projeção não tem linha no banco, logo não tem ID")
	}
}

func TestSinalDoSaldoPorTipo(t *testing.T) {
	casos := map[domain.Kind]int64{
		domain.KindReceita:       10000,
		domain.KindDespesa:       -10000,
		domain.KindCartaoCredito: -10000,
		domain.KindInvestimento:  -10000,
	}

	for tipo, esperado := range casos {
		if got := tipo.Signed(10000); got != esperado {
			t.Errorf("%s: %d, esperava %d", tipo, got, esperado)
		}
	}
}

func TestDescricaoPadraoDeCadaTipo(t *testing.T) {
	casos := map[domain.Kind]string{
		domain.KindReceita:       "Receita",
		domain.KindDespesa:       "Despesa",
		domain.KindCartaoCredito: "Gasto no cartão de crédito",
		domain.KindInvestimento:  "Investimento",
	}

	for tipo, esperado := range casos {
		if got := tipo.Label(); got != esperado {
			t.Errorf("%s: label = %q, esperava %q", tipo, got, esperado)
		}
	}
}
