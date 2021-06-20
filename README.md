# DBManager

## Description
`dbmanager` is a simple helper for creating postgresql records for tests.

It uses [squirrel](https://github.com/Masterminds/squirrel) for creating test records fast and easily.

The idea is that, given a set of default values for your relations, creating a record is as easy as
calling `Create` with the table name and a set of modifiers which will then override the default value
for each specified field.

Despite requiring the db connection handler, this was built for postgresql and won't work with any
databases since there are a couple of implementation details that are specific to postgresql:
    - setting the `PlaceholderFormat` to the dollar sign;
    - all queries being built with `ON CONFLICT DO NOTHING` so unique constraints are ignored.

## Usage
- The recommended usage would be to add the initialization code in a package accessible to all other
specific db logic packages (e.g. `models/dbmanager.go`) and then:
    + define a set of default values for each of the applications' relations;
    + create a constructor for initializing a dbmanager with such default values.

```go
var defaultRelationValuesMap = map[string]dbmanager.RelationValues{
	"users": {
		"id":         "c09d0a2d-3dab-4a60-bbfa-54c8ed397062",
		"username":   "tlins",
		"email":      "titolins@outlook.com",
		"password":   "super_secret",
		"created_at": sq.Expr("CURRENT_TIMESTAMP"),
		"updated_at": sq.Expr("CURRENT_TIMESTAMP"),
	},
}

func NewTestDBManager(db *sql.DB, t *testing.T) dbmanager.DBManager {
	return dbmanager.New(
		db,
		t,
		defaultRelationValuesMap,
	)
}
```

- When writing database tests, define a `runBefore` function that can set the db state to whatever 
it's required for the test. At that point, it's just a matter of initializing the db manager and 
creating the test records. This would look somewhat like this:

```go
	cases := []struct {
		title     string
		runBefore func(*sql.DB, *testing.T)
		exp       expType
		expErr    error
	}{
		{
			name: "fails to create existing user",
			runBefore: func(db *sql.DB, t *testing.T) { // injecting the db handler can be done at the execution time
				dbMan := models.NewTestDBManager(db, t)
				dbMan.ResolveTestRecord(
					"users",
					dbmanager.SetFieldValue("id", existingUserID), // presuming there's a unique constraint on the id field, this would cause the creation to fail
				)
			},
		},
	}
```

## Other thoughts
- The initial idea and the reason why it was decided to ignore unique constraints was that we could
simply seed the database once and run all tests against that given state. Doing it like that we could
simply call the same seed function for every test and not have to worry about execution order, etc..
It turned out to be really annoying to design tests having to consider a static db state (specially 
once your models start growing and the relations between them become more complex).

- For that reason, it was decided that having a clean state for each test case was a lot better instead.
The tool choosen for this job was the awesome [dbcleaner](https://github.com/khaiql/dbcleaner/blob/master/README.md).
It provides a nice and easy way to lock databases and clean them after each test case.

## Credits
- [squirrel](https://github.com/Masterminds/squirrel)
- [dbcleaner](https://github.com/khaiql/dbcleaner/blob/master/README.md)

## License
MIT
