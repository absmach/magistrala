package pgadapter

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/go-pg/pg/v9/types"
	"github.com/mmcloughlin/meow"
)

// CasbinRule represents a rule in Casbin.
type CasbinRule struct {
	ID    string
	PType string
	V0    string
	V1    string
	V2    string
	V3    string
	V4    string
	V5    string
}

type Filter struct {
	P []string
	G []string
}

// Adapter represents the github.com/go-pg/pg adapter for policy storage.
type Adapter struct {
	db        *pg.DB
	tableName string
	filtered  bool
}

type Option func(a *Adapter)

// NewAdapter is the constructor for Adapter.
// arg should be a PostgreS URL string or of type *pg.Options
// The adapter will create a DB named "casbin" if it doesn't exist
func NewAdapter(arg interface{}) (*Adapter, error) {
	db, err := createCasbinDatabase(arg)
	if err != nil {
		return nil, fmt.Errorf("pgadapter.NewAdapter: %v", err)
	}

	a := &Adapter{db: db}

	if err := a.createTableifNotExists(); err != nil {
		return nil, fmt.Errorf("pgadapter.NewAdapter: %v", err)
	}

	return a, nil
}

// NewAdapterByDB creates new Adapter by using existing DB connection
// creates table from CasbinRule struct if it doesn't exist
func NewAdapterByDB(db *pg.DB, opts ...Option) (*Adapter, error) {
	a := &Adapter{db: db}
	for _, opt := range opts {
		opt(a)
	}

	if len(a.tableName) > 0 {
		a.db.Model((*CasbinRule)(nil)).TableModel().Table().Name = a.tableName
		a.db.Model((*CasbinRule)(nil)).TableModel().Table().FullName = (types.Safe)(a.tableName)
		a.db.Model((*CasbinRule)(nil)).TableModel().Table().FullNameForSelects = (types.Safe)(a.tableName)
	}

	if err := a.createTableifNotExists(); err != nil {
		return nil, fmt.Errorf("pgadapter.NewAdapter: %v", err)
	}
	return a, nil
}

// WithTableName can be used to pass custom table name for Casbin rules
func WithTableName(tableName string) Option {
	return func(a *Adapter) {
		a.tableName = tableName
	}
}

func createCasbinDatabase(arg interface{}) (*pg.DB, error) {
	var opts *pg.Options
	var err error
	if connURL, ok := arg.(string); ok {
		opts, err = pg.ParseURL(connURL)
		if err != nil {
			return nil, err
		}
	} else {
		opts, ok = arg.(*pg.Options)
		if !ok {
			return nil, fmt.Errorf("must pass in a PostgreS URL string or an instance of *pg.Options, received %T instead", arg)
		}
	}

	db := pg.Connect(opts)
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE casbin")
	db.Close()

	opts.Database = "casbin"
	db = pg.Connect(opts)

	return db, nil
}

// Close close database connection
func (a *Adapter) Close() error {
	if a != nil && a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *Adapter) createTableifNotExists() error {
	err := a.db.CreateTable(&CasbinRule{}, &orm.CreateTableOptions{
		Temp:        false,
		IfNotExists: true,
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *CasbinRule) String() string {
	const prefixLine = ", "
	var sb strings.Builder

	sb.Grow(
		len(r.PType) +
			len(r.V0) + len(r.V1) + len(r.V2) +
			len(r.V3) + len(r.V4) + len(r.V5),
	)

	sb.WriteString(r.PType)
	if len(r.V0) > 0 {
		sb.WriteString(prefixLine)
		sb.WriteString(r.V0)
	}
	if len(r.V1) > 0 {
		sb.WriteString(prefixLine)
		sb.WriteString(r.V1)
	}
	if len(r.V2) > 0 {
		sb.WriteString(prefixLine)
		sb.WriteString(r.V2)
	}
	if len(r.V3) > 0 {
		sb.WriteString(prefixLine)
		sb.WriteString(r.V3)
	}
	if len(r.V4) > 0 {
		sb.WriteString(prefixLine)
		sb.WriteString(r.V4)
	}
	if len(r.V5) > 0 {
		sb.WriteString(prefixLine)
		sb.WriteString(r.V5)
	}

	return sb.String()
}

// LoadPolicy loads policy from database.
func (a *Adapter) LoadPolicy(model model.Model) error {
	var lines []*CasbinRule

	if err := a.db.Model(&lines).Select(); err != nil {
		return err
	}

	for _, line := range lines {
		persist.LoadPolicyLine(line.String(), model)
	}

	a.filtered = false

	return nil
}

func policyID(ptype string, rule []string) string {
	data := strings.Join(append([]string{ptype}, rule...), ",")
	sum := meow.Checksum(0, []byte(data))
	return fmt.Sprintf("%x", sum)
}

func savePolicyLine(ptype string, rule []string) *CasbinRule {
	line := &CasbinRule{PType: ptype}

	l := len(rule)
	if l > 0 {
		line.V0 = rule[0]
	}
	if l > 1 {
		line.V1 = rule[1]
	}
	if l > 2 {
		line.V2 = rule[2]
	}
	if l > 3 {
		line.V3 = rule[3]
	}
	if l > 4 {
		line.V4 = rule[4]
	}
	if l > 5 {
		line.V5 = rule[5]
	}

	line.ID = policyID(ptype, rule)

	return line
}

// SavePolicy saves policy to database.
func (a *Adapter) SavePolicy(model model.Model) error {
	tx, err := a.db.Begin()
	if err != nil {
		return fmt.Errorf("start DB transaction: %v", err)
	}
	defer tx.Close()

	_, err = tx.Model((*CasbinRule)(nil)).Where("id IS NOT NULL").Delete()
	if err != nil {
		return err
	}

	var lines []*CasbinRule

	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			lines = append(lines, line)
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			lines = append(lines, line)
		}
	}

	if len(lines) > 0 {
		_, err = tx.Model(&lines).
			OnConflict("DO NOTHING").
			Insert()
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit DB transaction: %v", err)
	}

	return nil
}

// AddPolicy adds a policy rule to the storage.
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	line := savePolicyLine(ptype, rule)
	err := a.db.RunInTransaction(func(tx *pg.Tx) error {
		_, err := a.db.Model(line).
			OnConflict("DO NOTHING").
			Insert()

		return err
	})

	return err
}

// AddPolicies adds policy rules to the storage.
func (a *Adapter) AddPolicies(sec string, ptype string, rules [][]string) error {
	var lines []*CasbinRule
	for _, rule := range rules {
		line := savePolicyLine(ptype, rule)
		lines = append(lines, line)
	}

	err := a.db.RunInTransaction(func(tx *pg.Tx) error {
		_, err := tx.Model(&lines).
			OnConflict("DO NOTHING").
			Insert()
		return err
	})

	return err
}

// RemovePolicy removes a policy rule from the storage.
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	line := savePolicyLine(ptype, rule)
	err := a.db.RunInTransaction(func(tx *pg.Tx) error {
		return a.db.Delete(line)
	})

	return err
}

// RemovePolicies removes policy rules from the storage.
func (a *Adapter) RemovePolicies(sec string, ptype string, rules [][]string) error {
	var lines []*CasbinRule
	for _, rule := range rules {
		line := savePolicyLine(ptype, rule)
		lines = append(lines, line)
	}

	err := a.db.RunInTransaction(func(tx *pg.Tx) error {
		_, err := tx.Model(&lines).
			Delete()
		return err
	})

	return err
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	query := a.db.Model((*CasbinRule)(nil)).Where("p_type = ?", ptype)

	idx := fieldIndex + len(fieldValues)
	if fieldIndex <= 0 && idx > 0 && fieldValues[0-fieldIndex] != "" {
		query = query.Where("v0 = ?", fieldValues[0-fieldIndex])
	}
	if fieldIndex <= 1 && idx > 1 && fieldValues[1-fieldIndex] != "" {
		query = query.Where("v1 = ?", fieldValues[1-fieldIndex])
	}
	if fieldIndex <= 2 && idx > 2 && fieldValues[2-fieldIndex] != "" {
		query = query.Where("v2 = ?", fieldValues[2-fieldIndex])
	}
	if fieldIndex <= 3 && idx > 3 && fieldValues[3-fieldIndex] != "" {
		query = query.Where("v3 = ?", fieldValues[3-fieldIndex])
	}
	if fieldIndex <= 4 && idx > 4 && fieldValues[4-fieldIndex] != "" {
		query = query.Where("v4 = ?", fieldValues[4-fieldIndex])
	}
	if fieldIndex <= 5 && idx > 5 && fieldValues[5-fieldIndex] != "" {
		query = query.Where("v5 = ?", fieldValues[5-fieldIndex])
	}

	err := a.db.RunInTransaction(func(tx *pg.Tx) error {
		_, err := query.Delete()
		return err
	})

	return err
}

func (a *Adapter) LoadFilteredPolicy(model model.Model, filter interface{}) error {
	if filter == nil {
		return a.LoadPolicy(model)
	}

	filterValue, ok := filter.(*Filter)
	if !ok {
		return fmt.Errorf("invalid filter type")
	}
	err := a.loadFilteredPolicy(model, filterValue, persist.LoadPolicyLine)
	if err != nil {
		return err
	}
	a.filtered = true
	return nil
}

func buildQuery(query *orm.Query, values []string) (*orm.Query, error) {
	for ind, v := range values {
		if v == "" {
			continue
		}
		switch ind {
		case 0:
			query = query.Where("v0 = ?", v)
		case 1:
			query = query.Where("v1 = ?", v)
		case 2:
			query = query.Where("v2 = ?", v)
		case 3:
			query = query.Where("v3 = ?", v)
		case 4:
			query = query.Where("v4 = ?", v)
		case 5:
			query = query.Where("v5 = ?", v)
		default:
			return nil, fmt.Errorf("filter has more values than expected, should not exceed 6 values")
		}
	}
	return query, nil
}

func (a *Adapter) loadFilteredPolicy(model model.Model, filter *Filter, handler func(string, model.Model)) error {
	if filter.P != nil {
		lines := []*CasbinRule{}

		query := a.db.Model(&lines).Where("p_type = 'p'")
		query, err := buildQuery(query, filter.P)
		if err != nil {
			return err
		}
		err = query.Select()
		if err != nil {
			return err
		}

		for _, line := range lines {
			handler(line.String(), model)
		}
	}
	if filter.G != nil {
		lines := []*CasbinRule{}

		query := a.db.Model(&lines).Where("p_type = 'g'")
		query, err := buildQuery(query, filter.G)
		if err != nil {
			return err
		}
		err = query.Select()
		if err != nil {
			return err
		}

		for _, line := range lines {
			handler(line.String(), model)
		}
	}
	return nil
}

func (a *Adapter) IsFiltered() bool {
	return a.filtered
}
