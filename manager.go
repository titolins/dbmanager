package dbmanager

import (
	"database/sql"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/lann/builder"
)

// DBManager represents the dbManager
type DBManager interface {
	ResolveTestRecord(string, ...RelationValuesOption)
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

// New returns a DBManager
func New(db *sql.DB, t *testing.T, defaultValues map[string]RelationValues) DBManager {
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
	opts ...RelationValuesOption,
) {
	values := dbMan.relationValues(tableName, opts...)
	_, err := dbMan.insertBuilder(tableName).SetMap(sq.Eq(values)).Suffix("ON CONFLICT DO NOTHING").Query()
	if err != nil {
		dbMan.t.Fatalf("Test setup failed: could not create test record for '%s': %+v", tableName, err)
	}
}

func (dbMan *dbManager) insertBuilder(tableName string) sq.InsertBuilder {
	return dbMan.queryBuilder.
		RunWith(dbMan.db).
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
