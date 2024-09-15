package docs

import (
	_ "embed"
)

//go:embed apidocs.swagger.yaml
var APIV11JSON []byte
