package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/sirupsen/logrus"
)

/*******************************************************************************
	DATA STRUCTURES
		1. AlrRepository
		2. alrCopyFromSource
			- Used to implement the CopyFrom from the pgx package CopyFrom is
			Go's connection to PostgreSQL's COPY functionality.
*******************************************************************************/

type AlrRepository struct {
	*sql.DB
	*pgx.Conn
}

type alrCopyFromSource struct {
	ctx       context.Context
	metaKey   int
	timestamp time.Time
	alrCount  int
	alrRowLen int
	alrs      []models.Alr
}

/*******************************************************************************
	HELPER FUNCTIONS
		1. NewAlrRepo - Used to instantiate AlrRepository struct
		2. gogEncoder
			- Used for turning map[string]string to []bytes
		3. gobDecoder
			- Used for turning []bytes back to map
*******************************************************************************/

func NewAlrRepo(db *sql.DB) *AlrRepository {
	conn, err := stdlib.AcquireConn(db)
	if err != nil {
		logrus.Warn(err, "failed to acquire connection")
	}
	return &AlrRepository{db, conn}
}

func GobEncoder(mp map[string]string) ([]byte, error) {
	var bytea bytes.Buffer
	enc := gob.NewEncoder(&bytea)
	if err := enc.Encode(mp); err != nil {
		return nil, err
	}
	return bytea.Bytes(), nil
}

func GobDecoder(byteMap []byte) (map[string]string, error) {
	var mp map[string]string
	r := bytes.NewReader(byteMap)
	dec := gob.NewDecoder(r)
	if err := dec.Decode(&mp); err != nil {
		return nil, err
	}
	return mp, nil
}

/*******************************************************************************
	METHODS
		1. Next, Err, Values
			- These methods are need to implement the CopyFrom from pgx.
		2. AddAlr
			- Adds new rows to the ALR Tables. A single call of this function
			can ingest ALR data per ACO.
		3. GetAlr
			- Retreive data from database.
*******************************************************************************/

func (a *alrCopyFromSource) Next() bool {
	// If we went through all of the rows, this condition should be false
	return a.alrCount < a.alrRowLen
}

func (a *alrCopyFromSource) Err() error {
	// Err() returns nil if "Done" has not closed and context is still valid
	return a.ctx.Err()
}

func (a *alrCopyFromSource) Values() ([]interface{}, error) {
	// Get the row
	alr := a.alrs[a.alrCount]
	// Move counter up so we look at the following row next
	a.alrCount++

	// Convert the map[string]string to slices of bytes
	bytea, err := GobEncoder(alr.KeyValue)
	if err != nil {
		return nil, err
	}

	row := []interface{}{
		// Ordering is based off how they are in the database. See migration 10
		a.metaKey,   // bigint
		alr.BeneMBI, // varchar
		alr.BeneHIC,
		alr.BeneFirstName,
		alr.BeneLastName,
		alr.BeneSex,
		alr.BeneDOB,
		alr.BeneDOD,
		bytea, // bytea
	}
	return row, nil
}

func (r *AlrRepository) AddAlr(ctx context.Context, aco string, timestamp time.Time, alrs []models.Alr) error {
	// Do this in a single transaction using BeginEX from the pgx package
	tx, err := r.BeginEx(ctx, nil)
	if err != nil {
		return err
	}

	// Update the alr_meta table and get the foreign key
	updateAlrMeta := sqlFlavor.NewInsertBuilder().InsertInto("alr_meta")
	updateAlrMeta.Cols("aco", "timestp").Values(aco, timestamp)
	query, args := updateAlrMeta.Build()
	query = fmt.Sprintf("%s RETURNING id", query)

	var metaKey int
	if err := tx.QueryRowEx(ctx, query, nil, args...).Scan(&metaKey); err != nil {
		return err
	}

	// Update the alr table
	cfs := &alrCopyFromSource{
		ctx:       ctx,
		metaKey:   metaKey,
		timestamp: timestamp,
		alrCount:  0,
		alrRowLen: len(alrs),
		alrs:      alrs,
	}

	fields := []string{"metakey", "mbi", "hic", "firstname", "lastname", "sex", "dob", "dod", "keyvalue"}
	_, err = tx.CopyFrom(pgx.Identifier([]string{"alr"}), fields, cfs)
	// If the copyfrom fails, attempt to rollback changes
	if err != nil {
		if err := tx.Rollback(); err != nil {
			logrus.Warnf("Failed to rollback transaction %s", err.Error())
		}
		return err
	}

	// End the transaction
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *AlrRepository) GetAlr(ctx context.Context, ACO string, timestamp time.Time) ([]models.Alr, error) {

	/* 	Query the alr_meta table to get foreign key */

	tx, err := r.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Build the query
	// sqlFlavor is from the repository.go file
	meta := sqlFlavor.NewSelectBuilder()
	meta.Select("id").From("alr_meta")
	meta.Where(
		meta.Equal("aco", ACO),
		meta.GreaterEqualThan("timestp", timestamp),
	)
	query, args := meta.Build()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		// No rollback needed since nothing is begin added to DB
		return nil, err
	}

	// Get the foreign keys from DB
	var foreignKeys []interface{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		foreignKeys = append(foreignKeys, id)
	}

	if len(foreignKeys) == 0 {
		return nil, errors.New("No records were found")
	}

	/* 	End of alr_meta */

	/* Query the alr table for ALR data */

	// Build the query
	alr := sqlFlavor.NewSelectBuilder()
	alr.Select("id", "metakey", "mbi", "hic", "firstname", "lastname", "sex",
		"dob", "dod", "keyvalue").From("alr")
	alr.Where(
		alr.In("metakey", foreignKeys...),
	)
	// Re-using variables from before
	query, args = alr.Build()
	// Using the foreign key, query the alr table
	rows, err = tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	// Get ALR data
	var alrs []models.Alr
	for rows.Next() {
		var alr models.Alr
		var keyValueBytes []byte
		if err := rows.Scan(&alr.ID, &alr.MetaKey, &alr.BeneMBI, &alr.BeneHIC,
			&alr.BeneFirstName, &alr.BeneLastName, &alr.BeneSex,
			&alr.BeneDOB, &alr.BeneDOD, &keyValueBytes); err != nil {
			return nil, err
		}
		keyValue, err := GobDecoder(keyValueBytes)
		if err != nil {
			return nil, err
		}
		alr.KeyValue = keyValue
		alrs = append(alrs, alr)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	// Send back either a specific error or the results back to caller
	return alrs, nil
}
