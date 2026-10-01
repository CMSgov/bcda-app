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
}
