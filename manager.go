package main

import (
	"database/sql"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/lann/builder"
)

// DBManager represents the dbManager
type DBManager interface {
	ResolveTestRecord(string, string, ...RelationValuesOption)
}

// RelationValues represents the models values used for querying the db in tests
type RelationValues map[string]interface{}

// RelationValuesOption represents the option function to be passed into the db tests manager
type RelationValuesOption func(RelationValues)

type dbManager struct {
	db                    *sql.DB
	t                     *testing.T
	queryBuilder          sq.StatementBuilderType
	defaultRelationValues map[string]RelationValues
}

// NewTestDBManager returns a dbManager
func NewTestDBManager(db *sql.DB, t *testing.T, defaultValues map[string]RelationValues) DBManager {
	return &dbManager{
		db:                    db,
		t:                     t,
		queryBuilder:          sq.StatementBuilderType(builder.EmptyBuilder).PlaceholderFormat(sq.Dollar),
		defaultRelationValues: defaultValues,
	}
}

// ResolveTestRecord checks if there is a record in the database with the
// give values and creates it if it's not there
func (dbMan *dbManager) ResolveTestRecord(
	tableName string,
	stringID string,
	opts ...RelationValuesOption,
) {
	values := dbMan.relationValues(tableName, opts...)

	tx, err := dbMan.db.Begin()
	if err != nil {
		dbMan.t.Fatalf("Test setup failed: could not start transaction: %s", err)
	}

	err = dbMan.checkIfExisting(tx, tableName, values)
	switch {
	case err == sql.ErrNoRows:
		builder := dbMan.insertBuilder(tableName).SetMap(sq.Eq(values))
		q, vs, err := builder.ToSql()
		if err != nil {
			dbMan.rollback(tx)
			dbMan.t.Fatalf("Test setup failed: could not generate sql string for '%s': %+v", tableName, err)
		}

		_, err = tx.Query(q, vs...)
		if err != nil {
			dbMan.rollback(tx)
			dbMan.t.Fatalf("Test setup failed: could not create test record for '%s': %s", tableName, err)
		}

		dbMan.commit(tx)
		return
	case err != nil:
		dbMan.rollback(tx)
		dbMan.t.Fatalf("failed during checkIfExisting %s", err)
	default:
		// no errors means we found a result and don't need to do anything else
		dbMan.commit(tx)
		return
	}
}

func (dbMan *dbManager) rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil {
		dbMan.t.Fatalf(
			"Test setup failed: failed to rollback: %s",
			err,
		)
	}
}
func (dbMan *dbManager) commit(tx *sql.Tx) {
	if commitErr := tx.Commit(); commitErr != nil {
		dbMan.t.Fatalf(
			"Test setup failed: could not commit transaction: %s", commitErr)
	}
}

func (dbMan *dbManager) checkIfExisting(
	tx *sql.Tx,
	tableName string,
	values RelationValues,
) error {
	selectValues := sq.Eq(parseSelectValues(values))

	builder := dbMan.selectBuilder(tableName).Where(selectValues).Limit(1)
	q, vs, err := builder.ToSql()
	if err != nil {
		return err
	}

	row := tx.QueryRow(q, vs...)
	err = row.Scan()
	if err != nil {
		return err
	}

	return nil
}

func (dbMan *dbManager) selectBuilder(tableName string) sq.SelectBuilder {
	return dbMan.queryBuilder.
		Select("").
		From(tableName)
}

func (dbMan *dbManager) insertBuilder(tableName string) sq.InsertBuilder {
	return dbMan.queryBuilder.
		Insert(tableName)
}

// relationValues returns a copy of the default values for a given
// relation with the applied option functions
func (dbMan *dbManager) relationValues(relationName string, opts ...RelationValuesOption) RelationValues {
	defaultVal := dbMan.getDefaultRelationValues(relationName)
	for _, opt := range opts {
		opt(defaultVal)
	}

	return defaultVal
}

// getDefaultRelationValues creates a copy of the default value
func (dbMan *dbManager) getDefaultRelationValues(relationName string) RelationValues {
	defaultValue, ok := dbMan.defaultRelationValues[relationName]
	if !ok {
		dbMan.t.Fatalf("no default values for relation '%s'", relationName)
	}

	values := make(RelationValues)
	for k, v := range defaultValue {
		values[k] = v
	}
	return values
}

func parseSelectValues(
	originalValues RelationValues,
) map[string]interface{} {
	values := make(map[string]interface{})

	for k, v := range originalValues {
		if _, ok := v.(sq.Sqlizer); !ok {
			values[k] = v
		}

	}
	return values
}
