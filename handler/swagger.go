package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/amjadjibon/dbank/docs"
)

func SwaggerUI(w http.ResponseWriter, _ *http.Request) {
	swaggerTemplate := template.Must(template.New("swagger").Parse(`
<html>
	<head>
	<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3/swagger-ui.css">

	<script src="https://unpkg.com/swagger-ui-dist@3/swagger-ui-standalone-preset.js"></script>
	<script src="https://unpkg.com/swagger-ui-dist@3/swagger-ui-bundle.js" charset="UTF-8"></script>
	</head>
	<body>
	<div id="swagger-ui"></div>
	<script>
		window.addEventListener('load', (event) => {
			const ui = SwaggerUIBundle({
			    url: "/swagger/v1/openapiv2.json",
			    dom_id: '#swagger-ui',
			    presets: [
			      SwaggerUIBundle.presets.apis,
			      SwaggerUIBundle.SwaggerUIStandalonePreset
			    ],
				plugins: [
                	SwaggerUIBundle.plugins.DownloadUrl
            	],
				deepLinking: true,
			  })
			window.ui = ui
		});
	</script>
	</body>
</html>`))

	var payload bytes.Buffer
	if err := swaggerTemplate.Execute(&payload, struct{}{}); err != nil {
		fmt.Println(err)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Could not render Swagger"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(payload.Bytes())
	if err != nil {
		fmt.Println(err)
	}
}

func SwaggerAPIv1(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(docs.APIV11JSON); err != nil {
		fmt.Println(err)
	}
}
