package currency

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testXML = `<?xml version="1.0" encoding="UTF-8"?>
  <StructureSpecificData>
      <DataSet>
          <Series>
              <Obs TIME_PERIOD="2024-05-01" OBS_VALUE="1.0823"/>
              <Obs TIME_PERIOD="2024-05-02" OBS_VALUE="1.0791"/>
          </Series>
      </DataSet>
  </StructureSpecificData>`

func TestExtractObs(t *testing.T) {
	rates, err := extractObs(strings.NewReader(testXML))

	require.NoError(t, err)

	require.Len(t, rates, 2)
	assert.Equal(t, "2024-05-01", rates[0].Date.Format("2006-01-02"))
	assert.InDelta(t, 1.0823, rates[0].Value, 0.0001)

	assert.Equal(t, "2024-05-02", rates[1].Date.Format("2006-01-02"))
	assert.InDelta(t, 1.0791, rates[1].Value, 0.0001)

}
