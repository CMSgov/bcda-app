package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
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
}

type alrCopyFromSource struct {
	ctx       context.Context
	metaKey   int64
	timestamp time.Time
	alrs      []models.Alr
}

/*******************************************************************************
	HELPER FUNCTIONS
		1. NewAlrRepo - Used to instantiate AlrRepository struct
		2. getPgxConn - Grabs pgx.Conn from sql.DB
		2. encoder
			- Used for turning map[string]string to []bytes
		3. decoder
			- Used for turning []bytes back to map
		4. rollBack
			- Used to do a rollback when a transaction fails
*******************************************************************************/

func NewAlrRepo(db *sql.DB) *AlrRepository {
	return &AlrRepository{db}
}

func getPgxConn(db *sql.DB) *pgx.Conn {
	conn, err := stdlib.AcquireConn(db)
	if err != nil {
		logrus.Warn(err, "failed to acquire connection")
	}
	return conn
}

func encoder(mp map[string]string) ([]byte, error) {
	var bytea bytes.Buffer
	enc := gob.NewEncoder(&bytea)
	if err := enc.Encode(mp); err != nil {
		return nil, err
	}
	return bytea.Bytes(), nil
}

func decoder(byteMap []byte) (map[string]string, error) {
	var mp map[string]string
	r := bytes.NewReader(byteMap)
	dec := gob.NewDecoder(r)
	if err := dec.Decode(&mp); err != nil {
		return nil, err
	}
	return mp, nil
}

func rollBack(tx *pgx.Tx) {
	if err := tx.Rollback(); err != nil {
		logrus.Warnf("Failed to rollback transaction %s", err.Error())
	}
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
	return len(a.alrs) > 0
}

func (a *alrCopyFromSource) Err() error {
	// Err() returns nil if "Done" has not closed and context is still valid
	return a.ctx.Err()
}

func (a *alrCopyFromSource) Values() ([]interface{}, error) {
	// Get the row
	alr := a.alrs[0]
	// Move counter up so we look at the following row next
	a.alrs = a.alrs[1:]

	// Convert the map[string]string to slices of bytes
	bytea, err := encoder(alr.KeyValue)
	if err != nil {
		return nil, err
	}

	row := []interface{}{
		&pgtype.Int8{Int: a.metaKey, Status: pgtype.Present},      // bigint
		&pgtype.Text{String: alr.BeneMBI, Status: pgtype.Present}, // varchar
		&pgtype.Text{String: alr.BeneHIC, Status: pgtype.Present},
		&pgtype.Text{String: alr.BeneFirstName, Status: pgtype.Present},
		&pgtype.Text{String: alr.BeneLastName, Status: pgtype.Present},
		&pgtype.Text{String: alr.BeneSex, Status: pgtype.Present},
		&pgtype.Timestamp{Time: alr.BeneDOB, Status: pgtype.Present},
		&pgtype.Timestamp{Time: alr.BeneDOD, Status: pgtype.Present},
		&pgtype.Bytea{Bytes: bytea, Status: pgtype.Present}, // bytea
	}

	return row, nil
}

func (r *AlrRepository) AddAlr(ctx context.Context, aco string, timestamp time.Time, alrs []models.Alr) error {
	// Grab pgx.Conn
	conn := getPgxConn(r.DB)

	// Do this in a single transaction using BeginEX from the pgx package
	tx, err := conn.BeginEx(ctx, nil)
	if err != nil {
		rollBack(tx)
		return err
	}

	// Update the alr_meta table and get the foreign key
	updateAlrMeta := sqlFlavor.NewInsertBuilder().InsertInto("alr_meta")
	updateAlrMeta.Cols("aco", "timestp").Values(aco, timestamp)
	query, args := updateAlrMeta.Build()
	query = fmt.Sprintf("%s RETURNING id", query)

	var metaKey int64
	if err := tx.QueryRowEx(ctx, query, nil, args...).Scan(&metaKey); err != nil {
		rollBack(tx)
		return err
	}

	// Update the alr table
	cfs := &alrCopyFromSource{
		ctx:       ctx,
		metaKey:   metaKey,
		timestamp: timestamp,
		alrs:      alrs,
	}

	fields := []string{"metakey", "mbi", "hic", "firstname", "lastname", "sex", "dob", "dod", "keyvalue"}
	_, err = tx.CopyFrom(pgx.Identifier([]string{"alr"}), fields, cfs)
	// If the copyfrom fails, attempt to rollback changes
	if err != nil {
		rollBack(tx)
		return err
	}

	// End the transaction
	if err := tx.Commit(); err != nil {
		rollBack(tx)
		return err
	}
	return nil
}

func (r *AlrRepository) GetAlr(ctx context.Context, ACO string, lowerBound time.Time, upperBound time.Time) ([]models.Alr, error) {

	/* 	Query the alr_meta table to get foreign key */

	// Build the query
	// sqlFlavor is from the repository.go file
	meta := sqlFlavor.NewSelectBuilder()
	meta.Select("alr_meta.timestp", "alr.id", "alr.metakey", "alr.mbi", "alr.hic",
		"alr.firstname", "alr.lastname", "alr.sex",
		"alr. dob", "alr.dod", "alr.keyvalue").
		From("alr_meta")

	var whereCond string
	if upperBound.IsZero() && lowerBound.IsZero() {
		whereCond = meta.Equal("aco", ACO)
	} else if upperBound.IsZero() && !lowerBound.IsZero() {
		whereCond = meta.And(meta.Equal("aco", ACO),
			meta.GreaterEqualThan("alr_meta.timestp", lowerBound))
	} else if !upperBound.IsZero() && lowerBound.IsZero() {
		whereCond = meta.And(meta.Equal("aco", ACO),
			meta.LessEqualThan("alr_meta.timestp", upperBound))
	} else {
		whereCond = meta.And(meta.Equal("aco", ACO),
			meta.LessEqualThan("alr_meta.timestp", upperBound),
			meta.GreaterEqualThan("alr_meta.timestp", lowerBound))
	}

	meta.Where(whereCond)
	meta.JoinWithOption("LEFT", "alr", "alr_meta.id = alr.metakey")
	query, args := meta.Build()

	// Using the foreign key, query the alr table
	rows, err := r.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	// Get ALR data
	var alrs []models.Alr
	for rows.Next() {
		var alr models.Alr
		var keyValueBytes []byte
		if err := rows.Scan(&alr.Timestamp, &alr.ID, &alr.MetaKey, &alr.BeneMBI, &alr.BeneHIC,
			&alr.BeneFirstName, &alr.BeneLastName, &alr.BeneSex,
			&alr.BeneDOB, &alr.BeneDOD, &keyValueBytes); err != nil {
			return nil, err
		}
		keyValue, err := decoder(keyValueBytes)
		if err != nil {
			return nil, err
		}
		alr.KeyValue = keyValue
		alrs = append(alrs, alr)
	}

	// Send back either a specific error or the results back to caller
	return alrs, nil
}
