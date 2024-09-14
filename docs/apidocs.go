package docs

import (
	_ "embed"
)

//go:embed apidocs.swagger.yaml
var ApiV1JSON []byte
