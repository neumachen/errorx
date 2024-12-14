package errorx

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorSetter(t *testing.T) {
	t.Run("SetMetadata with valid JSON", func(t *testing.T) {
		err := New("test error")
		metadata := json.RawMessage(`{"key": "value"}`)
		
		setErr := err.SetMetadata(&metadata)
		require.NoError(t, setErr)
		
		require.Equal(t, &metadata, err.Metadata())
	})

	t.Run("SetMetadata with nil", func(t *testing.T) {
		err := New("test error")
		
		setErr := err.SetMetadata(nil)
		require.NoError(t, setErr)
		
		require.Nil(t, err.Metadata())
	})

	t.Run("SetMetadata can be updated", func(t *testing.T) {
		err := New("test error")
		metadata1 := json.RawMessage(`{"first": true}`)
		metadata2 := json.RawMessage(`{"second": true}`)
		
		setErr := err.SetMetadata(&metadata1)
		require.NoError(t, setErr)
		require.Equal(t, &metadata1, err.Metadata())
		
		setErr = err.SetMetadata(&metadata2)
		require.NoError(t, setErr)
		require.Equal(t, &metadata2, err.Metadata())
	})
}
