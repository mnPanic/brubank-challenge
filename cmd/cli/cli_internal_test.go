package cli

import (
	"invoice-generator/pkg/invoice"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadCalls(t *testing.T) {
	reader := func(_ string) ([]byte, error) {
		return []byte(`numero origen,numero destino,duracion,fecha
+5491167980950,+191167980952,462,2020-11-10T04:02:45Z
+5491167910920,+191167980952,392,2020-08-09T04:45:25Z
`), nil
	}

	calls, err := readCalls(reader, "test-file.csv")
	require.NoError(t, err)

	expectedCalls := []invoice.Call{
		{
			SourcePhone:      "+5491167980950",
			DestinationPhone: "+191167980952",
			Duration:         462,
			Date:             time.Date(2020, time.November, 10, 04, 02, 45, 0, time.UTC),
		},
		{
			SourcePhone:      "+5491167910920",
			DestinationPhone: "+191167980952",
			Duration:         392,
			Date:             time.Date(2020, time.August, 9, 04, 45, 25, 0, time.UTC),
		},
	}
	assert.Equal(t, expectedCalls, calls)
}
