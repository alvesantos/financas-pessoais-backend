package service

import (
	"context"
	"errors"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// ensureCategory confere a categoria informada antes de gravá-la em um
// lançamento ou em um fixo. Sem categoria não há nada a checar.
//
// A busca é no escopo do usuário, então a categoria de outra pessoa
// simplesmente não existe. O tipo precisa bater: uma categoria de despesa
// em uma receita produziria um agrupamento sem sentido no painel.
func ensureCategory(
	ctx context.Context,
	repository domain.CategoryRepository,
	userID int64,
	categoryID *int64,
	kind domain.Kind,
) error {
	if categoryID == nil {
		return nil
	}

	category, err := repository.FindByID(ctx, userID, *categoryID)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			return domain.ErrValidation.WithFields(map[string]string{
				"category_id": "categoria não encontrada",
			})
		}
		return err
	}

	if category.Kind != kind {
		return domain.ErrValidation.WithFields(map[string]string{
			"category_id": "a categoria é de " + category.Kind.Label() + ", não combina com este tipo",
		})
	}

	return nil
}
