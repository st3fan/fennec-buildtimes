// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type BuddyBuild struct {
	AccessToken string
}

type CommitInfo struct {
	Author    string   `json:"author"`
	Branch    string   `json:"branch"`
	CommitSHA string   `json:"commit_sha"`
	URL       string   `json:"html_url"`
	Message   string   `json:"message"`
	Tags      []string `json:"tags"`
}

type Build struct {
	BuildNumber int        `json:"build_number"`
	BuildStatus string     `json:"build_status"`
	CommitInfo  CommitInfo `json:"commit_info"`
	CreatedAt   time.Time  `json:"created_at"`
	Finished    bool       `json:"finished"`
	FinishedAt  time.Time  `json:"finished_at"`
	StartedAt   time.Time  `json:"started_at"`
}

func (b Build) QueueDuration() int {
	return (int(b.StartedAt.Sub(b.CreatedAt).Seconds()) + 15) / 15
}

func (b Build) BuildDuration() int {
	return (int(b.FinishedAt.Sub(b.StartedAt).Seconds()) + 15) / 15
}

func (b Build) TotalDuration() time.Duration {
	return b.FinishedAt.Sub(b.CreatedAt)
}

func (bb BuddyBuild) getBuilds(appID string, branch string) ([]Build, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.buddybuild.com/v1/apps/%s/builds?limit=100&branch=%s", appID, branch), nil)
	req.Header.Add("Authorization", "Bearer "+bb.AccessToken)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var builds []Build
	if err = json.Unmarshal(body, &builds); err != nil {
		return nil, err
	}

	return builds, nil
}

type templateVariables struct {
	Application string
	Builds      []Build
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	bb := BuddyBuild{AccessToken: os.Getenv("BUDDYBUILD_ACCESS_KEY")}

	builds, err := bb.getBuilds("57bf25c0f096bc01001e21e0", request.QueryStringParameters["branch"])
	if err != nil {
		log.Println("Failed to retrieve builds: ", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       http.StatusText(http.StatusInternalServerError),
		}, nil
	}

	t, err := template.ParseFiles("templates/main.html")
	if err != nil {
		log.Println("Failed to parse template: ", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       http.StatusText(http.StatusInternalServerError),
		}, nil
	}

	var builder strings.Builder
	if err := t.Execute(&builder, &templateVariables{Application: "Fennec", Builds: builds}); err != nil {
		log.Println("Failed to execute template: ", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       http.StatusText(http.StatusInternalServerError),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
		Body: builder.String(),
	}, nil
}

func main() {
	lambda.Start(handler)
}
