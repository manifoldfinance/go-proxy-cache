// Package main contains the code that redirects go modules requests to the correct Repository
package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// prefix is the base URL
const prefix = "/"

// Repository contains the property of a repository, namely the URL + Git (VCS)
type repository struct {
	URL, VCS string
}

// repoMap contains the list of repositories distributed by Manifold Finance
var repoMap = map[string]repository{
	"mock-go-cacheproxy": {"https://github.com/manifoldfinance/mock-go-cacheproxy", "git"},
}

var xTemplate = template.Must(template.New("x").Parse(`<!DOCTYPE html>
<html>
	<head>
	<meta charset="utf-8">
    <meta name="viewport" content="width=device-width">
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
		<meta name="go-import" content="go.bintray.xyz/{{.Head}} {{.Repo.VCS}} {{.Repo.URL}}">
		<meta name="go-source" content="go.bintray.xyz/{{.Head}} https://github.com/manifoldfinance/{{.Head}}/ https://github.com/manifoldfinance/{{.Head}}/tree/master{/dir} https://github.com/manifoldfinance/{{.Head}}/blob/master{/dir}/{file}#L{line}">
		<meta http-equiv="refresh" content="0; url=https://godoc.org/go.bintray.xyz/{{.Head}}{{.Tail}}">
	</head>
	<body>
		Nothing to see here; <a href="https://godoc.org/go.bintray.xyz/{{.Head}}{{.Tail}}">move along</a>.
	</body>
</html>
`))

var templateRedirection = `<!DOCTYPE html>
<html>
	<head>
	<meta charset="utf-8">
    <meta name="viewport" content="width=device-width">
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
		<meta http-equiv="refresh" content="0; url=https://git.bintray.xyz">
	</head>
	<body>
		Redirecting...
	</body>
</html>`

func makeRedirection() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: 301,
		Body:       templateRedirection,
	}
}

func makeError(code int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: code,
		Body: fmt.Sprintf(`<!DOCTYPE html>
		<html>
		<head>
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
		</head>
		<body>%s</body>
		</html>
		`, body),
	}
}

func handleRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	head, tail := strings.TrimPrefix(req.Path, prefix), ""
	if i := strings.Index(head, "/"); i != -1 {
		head, tail = head[:i], head[i:]
		// remove the versioning as it is not supported by GoDoc
		re := regexp.MustCompile("/v[0-9]{1,}")
		tail = re.ReplaceAllString(tail, "")
	}

	// The base route redirects to the FQDN
	if head == "" {
		return makeRedirection(), nil
	}

	repo, ok := repoMap[head]
	if !ok {
		return makeError(404, "<h1>404 Page Not Found</h1>"), nil
	}
	data := struct {
		Head, Tail string
		Repo       repository
	}{head, tail, repo}

	buf := bytes.NewBufferString("")
	if err := xTemplate.Execute(buf, data); err != nil {
		return makeError(500, "<h1>500 Bad Request</h1>"), err
	}

	return events.APIGatewayProxyResponse{
		Body:       buf.String(),
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}
