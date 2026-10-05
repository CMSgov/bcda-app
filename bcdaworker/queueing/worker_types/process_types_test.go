package worker_types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClaimsWindowUnmarshalJSON(t *testing.T) {
	t.Run("unmarshals new Earliest and Latest keys", func(t *testing.T) {
		jsonData := []byte(`{"Earliest":"2020-01-01T00:00:00Z","Latest":"2021-01-01T00:00:00Z"}`)
		var cw ClaimsWindow
		err := json.Unmarshal(jsonData, &cw)
		assert.NoError(t, err)
		assert.Equal(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), cw.Earliest)
		assert.Equal(t, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), cw.Latest)
	})

	t.Run("unmarshals legacy LowerBound and UpperBound keys", func(t *testing.T) {
		jsonData := []byte(`{"LowerBound":"2020-01-01T00:00:00Z","UpperBound":"2021-01-01T00:00:00Z"}`)
		var cw ClaimsWindow
		err := json.Unmarshal(jsonData, &cw)
		assert.NoError(t, err)
		assert.Equal(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), cw.Earliest)
		assert.Equal(t, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), cw.Latest)
	})

	t.Run("marshals dual-writing Earliest/Latest and LowerBound/UpperBound keys", func(t *testing.T) {
		cw := ClaimsWindow{
			Earliest: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			Latest:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		data, err := json.Marshal(cw)
		assert.NoError(t, err)
		var m map[string]string
		err = json.Unmarshal(data, &m)
		assert.NoError(t, err)
		assert.Equal(t, "2020-01-01T00:00:00Z", m["Earliest"])
		assert.Equal(t, "2021-01-01T00:00:00Z", m["Latest"])
		assert.Equal(t, "2020-01-01T00:00:00Z", m["LowerBound"])
		assert.Equal(t, "2021-01-01T00:00:00Z", m["UpperBound"])
	})

	t.Run("unmarshals legacy LowerBound and UpperBound within JobEnqueueArgs", func(t *testing.T) {
		jsonData := []byte(`{"ID":1,"ClaimsWindow":{"LowerBound":"2020-01-01T00:00:00Z","UpperBound":"2021-01-01T00:00:00Z"}}`)
		var args JobEnqueueArgs
		err := json.Unmarshal(jsonData, &args)
		assert.NoError(t, err)
		assert.Equal(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), args.ClaimsWindow.Earliest)
		assert.Equal(t, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), args.ClaimsWindow.Latest)
	})

	t.Run("unmarshals new Earliest and Latest within JobEnqueueArgs", func(t *testing.T) {
		jsonData := []byte(`{"ID":1,"ClaimsWindow":{"Earliest":"2020-01-01T00:00:00Z","Latest":"2021-01-01T00:00:00Z"}}`)
		var args JobEnqueueArgs
		err := json.Unmarshal(jsonData, &args)
		assert.NoError(t, err)
		assert.Equal(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), args.ClaimsWindow.Earliest)
		assert.Equal(t, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), args.ClaimsWindow.Latest)
	})
}
