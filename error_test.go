package errorx

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError_MarshalJSON(t *testing.T) {
	errx := New(errors.New("test"))
	b, err := json.Marshal(errx)
	require.NoError(t, err)
	fmt.Println(string(b))
}
