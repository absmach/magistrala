package postgres

import (
	"context"

	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/re"
)

// SQL Queries as Strings
const (
	addRuleQuery = `
		INSERT INTO rules (id, domain_id, input_channel, input_topic, logic_type, logic_value,
			output_channel, output_topic, recurring_time, recurring_type, recurring_period, status)
		VALUES (:id, :domain_id, :input_channel, :input_topic, :logic_type, :logic_value,
			:output_channel, :output_topic, :recurring_time, :recurring_type, :recurring_period, :status)
		RETURNING id;
	`

	viewRuleQuery = `
		SELECT id, domain_id, input_channel, input_topic, logic_type, logic_value, output_channel, 
			output_topic, recurring_time, recurring_type, recurring_period, status
		FROM rules
		WHERE id = :id;
	`

	updateRuleQuery = `
		UPDATE rules
		SET input_channel = :input_channel, input_topic = :input_topic, logic_type = :logic_type, 
			logic_value = :logic_value, output_channel = :output_channel, output_topic = :output_topic, 
			recurring_time = :recurring_time, recurring_type = :recurring_type, 
			recurring_period = :recurring_period, status = :status
		WHERE id = :id;
	`

	removeRuleQuery = `
		DELETE FROM rules
		WHERE id = :id;
	`

	listRulesQuery = `
		SELECT id, domain_id, input_channel, input_topic, logic_type, logic_value, output_channel, 
			output_topic, recurring_time, recurring_type, recurring_period, status
		FROM rules;
	`
)

type PostgresRepository struct {
	DB postgres.Database
}

func NewRepository(db postgres.Database) re.Repository {
	return &PostgresRepository{DB: db}
}

func (repo *PostgresRepository) AddRule(ctx context.Context, r re.Rule) (re.Rule, error) {
	dbr := ruleToDb(r)
	_, err := repo.DB.NamedExecContext(ctx, addRuleQuery, dbr)
	if err != nil {
		return re.Rule{}, err
	}
	return r, nil
}

func (repo *PostgresRepository) ViewRule(ctx context.Context, id string) (re.Rule, error) {
	var r re.Rule
	row := repo.DB.QueryRowxContext(ctx, `
		SELECT id, domain_id, input_channel, input_topic, logic_type, logic_value, output_channel,
			output_topic, recurring_time, recurring_type, recurring_period, status
		FROM rules WHERE id = $1
	`, id)

	err := row.Scan(
		&r.ID,
		&r.DomainID,
		&r.InputChannel,
		&r.InputTopic,
		&r.Logic.Type,
		&r.Logic.Value,
		&r.OutputChannel,
		&r.OutputTopic,
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
		SET input_channel = $2, input_topic = $3, logic_type = $4, logic_value = $5, output_channel = $6,
			output_topic = $7, recurring_time = $8, recurring_type = $9, recurring_period = $10, status = $11
		WHERE id = $1
	`,
		r.ID,
		r.InputChannel,
		r.InputTopic,
		r.Logic.Type,
		r.Logic.Value,
		r.OutputChannel,
		r.OutputTopic,
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
	rows, err := repo.DB.NamedQueryContext(ctx, listRulesQuery, pm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []re.Rule
	var r dbRule
	for rows.Next() {
		if err := rows.StructScan(&r); err != nil {
			return nil, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		rules = append(rules, dbToRule(r))
	}

	return rules, nil
}
