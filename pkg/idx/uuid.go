package idx

import (
	"github.com/google/uuid"
)

// UUID4 generates a random UUID (version 4).
func UUID4() string {
	id, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return id.String()
}
