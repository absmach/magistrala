package postgres

import (
	"context"
	"errors"

	"github.com/absmach/magistrala/re"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	DB *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{DB: db}
}

func (repo *PostgresRepository) AddRule(r re.Rule) (re.Rule, error) {
	_, err := repo.DB.Exec(context.Background(), `
		INSERT INTO rules (id, input_topics, logic_kind, logic_value, output_topics, schedule_dates, schedule_recurring, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		r.ID,
		r.InputTopics,
		r.Logic.Kind,
		r.Logic.Value,
		r.OutputTopics,
		r.Schedule.Dates,
		r.Schedule.Recurring,
		r.Active,
	)
	if err != nil {
		return re.Rule{}, err
	}
	return r, nil
}

func (repo *PostgresRepository) ViewRule(id string) (re.Rule, error) {
	var r re.Rule
	row := repo.DB.QueryRow(context.Background(), `
		SELECT id, input_topics, logic_kind, logic_value, output_topics, schedule_dates, schedule_recurring, active
		FROM rules WHERE id = $1
	`, id)

	err := row.Scan(
		&r.ID,
		&r.InputTopics,
		&r.Logic.Kind,
		&r.Logic.Value,
		&r.OutputTopics,
		&r.Schedule.Dates,
		&r.Schedule.Recurring,
		&r.Active,
	)
	if err != nil {
		return re.Rule{}, err
	}

	return r, nil
}

func (repo *PostgresRepository) UpdateRule(r re.Rule) (re.Rule, error) {
	result, err := repo.DB.Exec(context.Background(), `
		UPDATE rules
		SET input_topics = $2, logic_kind = $3, logic_value = $4, output_topics = $5, 
		    schedule_dates = $6, schedule_recurring = $7, active = $8
		WHERE id = $1
	`,
		r.ID,
		r.InputTopics,
		r.Logic.Kind,
		r.Logic.Value,
		r.OutputTopics,
		r.Schedule.Dates,
		r.Schedule.Recurring,
		r.Active,
	)
	if err != nil {
		return re.Rule{}, err
	}

	if result.RowsAffected() == 0 {
		return re.Rule{}, errors.New("no rows affected")
	}

	return r, nil
}

func (repo *PostgresRepository) RemoveRule(id string) error {
	result, err := repo.DB.Exec(context.Background(), `
		DELETE FROM rules WHERE id = $1
	`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("no rows affected")
	}

	return nil
}

func (repo *PostgresRepository) ListRules() ([]re.Rule, error) {
	rows, err := repo.DB.Query(context.Background(), `
		SELECT id, input_topics, logic_kind, logic_value, output_topics, schedule_dates, schedule_recurring, active
		FROM rules
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
			&r.InputTopics,
			&r.Logic.Kind,
			&r.Logic.Value,
			&r.OutputTopics,
			&r.Schedule.Dates,
			&r.Schedule.Recurring,
			&r.Active,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	return rules, nil
}
