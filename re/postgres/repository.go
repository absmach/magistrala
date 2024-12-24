package postgres

import (
	"context"
	"errors"

	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/re"
)

type PostgresRepository struct {
	DB postgres.Database
}

func NewRepository(db postgres.Database) re.Repository {
	return &PostgresRepository{DB: db}
}

func (repo *PostgresRepository) AddRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	_, err := repo.DB.ExecContext(ctx, `
		INSERT INTO rules (id, domain_id, input_topic, logic_type, logic_value, output_topic, recurring_time, recurring_type, recurring_period, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		r.ID,
		r.DomainID,
		r.InputTopic,
		r.Logic.Type,
		r.Logic.Value,
		r.OutputTopics,
		r.Schedule.Time,
		r.Schedule.RecurringType,
		r.Schedule.RecurringPeriod,
		r.Status,
	)
	if err != nil {
		return re.Rule{}, err
	}
	return r, nil
}

func (repo *PostgresRepository) ViewRule(ctx context.Context, id string) (re.Rule, error) {
	var r re.Rule
	row := repo.DB.QueryRowxContext(ctx, `
		SELECT id, domain_id, input_topic, logic_type, logic_value, output_topic, recurring_time, recurring_type, recurring_period, status
		FROM rules WHERE id = $1
	`, id)

	err := row.Scan(
		&r.ID,
		&r.DomainID,
		&r.InputTopic,
		&r.Logic.Type,
		&r.Logic.Value,
		&r.OutputTopics,
		&r.Schedule.Time,
		&r.Schedule.RecurringType,
		&r.Schedule.RecurringPeriod,
		&r.Status,
	)
	if err != nil {
		return re.Rule{}, err
	}

	return r, nil
}

func (repo *PostgresRepository) UpdateRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	result, err := repo.DB.ExecContext(ctx, `
		UPDATE rules
		SET input_topic = $2, logic_type = $3, logic_value = $4, output_topic = $5, 
		    recurring_time = $6, recurring_type = $7, recurring_period = $8, status = $9
		WHERE id = $1
	`,
		r.ID,
		r.InputTopic,
		r.Logic.Type,
		r.Logic.Value,
		r.OutputTopics,
		r.Schedule.Time,
		r.Schedule.RecurringType,
		r.Schedule.RecurringPeriod,
		r.Status,
	)
	if err != nil {
		return re.Rule{}, err
	}

	if _, err := result.RowsAffected(); err != nil {
		return re.Rule{}, errors.New("no rows affected")
	}

	return r, nil
}

func (repo *PostgresRepository) RemoveRule(ctx context.Context, id string) error {
	result, err := repo.DB.ExecContext(ctx, `
		DELETE FROM rules WHERE id = $1
	`, id)
	if err != nil {
		return err
	}

	if _, err := result.RowsAffected(); err != nil {
		return errors.New("no rows affected")
	}

	return nil
}

func (repo *PostgresRepository) ListRules(ctx context.Context, pm re.PageMeta) ([]re.Rule, error) {
	rows, err := repo.DB.QueryContext(ctx, `
		SELECT id, domain_id, input_topic, logic_type, logic_value, output_topic, 
			recurring_time, recurring_type, recurring_period, status FROM rules
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []re.Rule
	for rows.Next() {
		var r re.Rule
		err := rows.Scan(
			&r.ID,
			&r.DomainID,
			&r.InputTopic,
			&r.Logic.Type,
			&r.Logic.Value,
			&r.OutputTopics,
			&r.Schedule.Time,
			&r.Schedule.RecurringType,
			&r.Schedule.RecurringPeriod,
			&r.Status,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	return rules, nil
}
