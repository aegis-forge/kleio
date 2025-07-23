package database

import (
	"app/pkg/git/model"
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gosuri/uilive"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// addVersion sends the version nodes and relationships to neo4j
func addVersion(version model.Version, component string, commit string, date time.Time, driver neo4j.DriverWithContext, ctx context.Context) {
	versionName := version.GetVersionString()
	name := versionName

	if strings.HasSuffix(component, ".yaml") || strings.HasSuffix(component, ".yml") {
		for _, value := range ExecuteQueryWithRetNeo(
			`MATCH (:Workflow {full_name: $workflow})-[:PUSHED]->(c:Commit)
			ORDER BY c.date DESC
			RETURN c.full_name, c.date`,
			map[string]any{
				"workflow": component,
			},
			driver, ctx,
		) {
			workflowDateRaw, _ := value.Get("c.date")
			workflowDate := workflowDateRaw.(neo4j.LocalDateTime).Time()

			if date.After(workflowDate) {
				workflowNameRaw, _ := value.Get("c.full_name")
				workflowName := workflowNameRaw.(string)

				ExecuteQueryNeo(
					`MATCH (c:Commit {full_name: $commit})
					MATCH (v:Commit {full_name: $version})
					MERGE (c)-[:USES {times: $times, version: $semver, type: $type}]->(v)`,
					map[string]any{
						"commit":  commit,
						"version": workflowName,
						"times":   version.GetUses(),
						"semver":  version.GetVersionString(),
						"type":    version.GetVersionType(),
					},
					driver, ctx,
				)

				return
			}
		}
	} else {
		if res := ExecuteQueryWithRetNeo(
			`MATCH (v:Commit {full_name: $version})
			WITH COUNT(v) > 0 as node_v
			RETURN node_v`,
			map[string]any{
				"version": fmt.Sprintf("%s/%s", component, name),
			},
			driver, ctx,
		); res[0].Values[0] == true {
			ExecuteQueryNeo(
				`MATCH (c:Commit {full_name: $commit})
				MATCH (v:Commit {full_name: $version})
				MERGE (c)-[:USES {times: $times, version: $semver, type: $type}]->(v)`,
				map[string]any{
					"commit":  commit,
					"version": fmt.Sprintf("%s/%s", component, name),
					"times":   version.GetUses(),
					"semver":  version.GetVersionString(),
					"type":    version.GetVersionType(),
				},
				driver, ctx,
			)
		} else {
			for _, value := range ExecuteQueryWithRetNeo(
				`MATCH (:Component {full_name: $component})-[:DEPLOYS]->(:Version {full_name: $version})-[:PUSHES]->(c:Commit)
				ORDER BY c.date DESC
				RETURN c.full_name, c.date`,
				map[string]any{
					"component": component,
					"version":   fmt.Sprintf("%s/%s", component, name),
				},
				driver, ctx,
			) {
				workflowDateRaw, _ := value.Get("c.date")
				workflowDate := workflowDateRaw.(neo4j.LocalDateTime).Time()

				if date.After(workflowDate) {
					workflowNameRaw, _ := value.Get("c.full_name")
					workflowName := workflowNameRaw.(string)

					ExecuteQueryNeo(`
						MATCH (v:Commit {full_name: $version})
						MATCH (c:Commit {full_name: $commit})
						MERGE (c)-[:USES {times: $times, version: $semver, type: $type}]->(v)`,
						map[string]any{
							"version": workflowName,
							"commit":  commit,
							"times":   version.GetUses(),
							"semver":  version.GetVersionString(),
							"type":    version.GetVersionType(),
						},
						driver, ctx,
					)

					return
				}
			}
		}
	}
}

// addComponents sends the component nodes and relationships to neo4j
func addComponents(component model.Component, commit string, date time.Time, driver neo4j.DriverWithContext, ctx context.Context) {
	fullName := component.GetName()
	componentSplit := strings.Split(component.GetName(), "/")

	if strings.HasSuffix(component.GetName(), ".yaml") || strings.HasSuffix(component.GetName(), ".yml") {
		fullName = strings.Join(strings.Split(commit, "/")[:2], "/") + "/" + componentSplit[len(componentSplit)-1]
	}

	for _, version := range component.GetHistory() {
		addVersion(*version, fullName, commit, date, driver, ctx)
	}
}

// addCommits sends the commit nodes and relationships to neo4j
func addCommits(commit model.Commit, workflow string, driver neo4j.DriverWithContext, ctx context.Context) {
	commitFull := fmt.Sprintf("%s/%s", workflow, commit.GetHash())
	content, _ := commit.GetContent(false)

	ExecuteQueryNeo(
		`MATCH (w:Workflow {full_name: $full})
		MERGE (c:Commit {name: $hash, date: $date, full_name: $full_c, content: $content})
		MERGE (w)-[:PUSHED]->(c)`,
		map[string]any{
			"full":    workflow,
			"hash":    commit.GetHash(),
			"date":    neo4j.LocalDateTimeOf(commit.GetDate()),
			"full_c":  commitFull,
			"content": content,
		},
		driver, ctx)

	for _, component := range commit.GetComponents() {
		addComponents(*component, commitFull, commit.GetDate(), driver, ctx)
	}
}

// addWorkflows sends the workflow nodes and relationships to neo4j
func addWorkflows(workflow model.File, repo string, driver neo4j.DriverWithContext, ctx context.Context) {
	workflowFull := fmt.Sprintf("%s/%s", repo, workflow.GetFilename())

	ExecuteQueryNeo(
		`MATCH (r:Repository {full_name: $full})
		MERGE (w:Workflow {name: $workflow, full_name: $full_w, path: $path})
		MERGE (r)-[:CONTAINS]->(w)`,
		map[string]any{
			"full":     repo,
			"workflow": workflow.GetFilename(),
			"full_w":   workflowFull,
			"path":     workflow.GetFilepath(),
		},
		driver, ctx,
	)

	for _, commit := range workflow.GetHistory() {
		addCommits(commit, workflowFull, driver, ctx)
	}

	// Retrieve all the commits of a workflow and compute the syntactical diff between them
	commits := ExecuteQueryWithRetNeo(
		`MATCH (:Workflow {full_name: $workflow})-[PUSHES]->(c:Commit)
		RETURN c.full_name, c.content, c.date`,
		map[string]any{
			"workflow": workflowFull,
		},
		driver, ctx,
	)

	for index, value := range commits {
		if index == len(commits)-1 {
			continue
		}
		
		precFile, _ := value.Get("c.full_name")
		succFile, _ := commits[index+1].Get("c.full_name")
		
		precDateRaw, _ := value.Get("c.date")
		succDateRaw, _ := commits[index+1].Get("c.date")
		
		precDate := precDateRaw.(neo4j.LocalDateTime).Time()
		succDate := succDateRaw.(neo4j.LocalDateTime).Time()
		
		precCommit, _ := value.Get("c.content")
		succCommit, _ := commits[index+1].Get("c.content")

		precContent, err := base64.StdEncoding.DecodeString(precCommit.(string))
		succContent, err := base64.StdEncoding.DecodeString(succCommit.(string))

		if err != nil {
			panic(err)
		}
		
		cmd := exec.Command(
			"/bin/bash", "-c", 
			fmt.Sprintf("gawd --json <(echo \"%s\") <(echo \"%s\")",
				strings.ReplaceAll(strings.ReplaceAll(string(precContent), "$", "\\$"), "\"", "\\\""),
				strings.ReplaceAll(strings.ReplaceAll(string(succContent), "$", "\\$"), "\"", "\\\""),
			),
		)
		out, err := cmd.Output()
		
		if err != nil {
			panic(err)
		}
		
		ExecuteQueryNeo(
			`MATCH (c1:Commit {full_name: $commit1})
			MATCH (c2:Commit {full_name: $commit2})
			MERGE (c1)-[:CHANGED_TO {diff: $diff, delta: $delta}]->(c2)`,
			map[string]any{
				"commit1": precFile,
				"commit2": succFile,
				"diff": strings.TrimSpace(string(out)),
				"delta": succDate.Sub(precDate).Seconds(),
			},
			driver, ctx)
	}
}

// SendToNeo adds the given repository to neo4j
func SendToNeo(repository model.Repository, driver neo4j.DriverWithContext, ctx context.Context) {
	vendor := strings.Split(repository.GetName(), "/")[0]
	repo := strings.Split(repository.GetName(), "/")[1]

	fmt.Println("\u001B[37m[NEO4J]\u001B[0m Saving repo \033[31m" + repo + "\033[0m")

	// Add the vendor and its repository to Neo4j
	ExecuteQueryNeo(
		`MERGE (v:Vendor {name: $vendor})
		MERGE (r:Repository {name: $repository, full_name: $full, url: $url})
		MERGE (v)-[:OWNS]->(r)`,
		map[string]any{
			"vendor":     vendor,
			"repository": repo,
			"full":       repository.GetName(),
			"url":        repository.GetUrl(),
		},
		driver, ctx,
	)

	writer := uilive.New()
	writer.Start()

	for i, workflow := range repository.GetFiles() {
		addWorkflows(workflow, repository.GetName(), driver, ctx)

		_, _ = fmt.Fprintf(
			writer,
			"\u001B[37m[NEO4J]\u001B[0m Saving workflows [%d/%d]\n",
			i+1, repository.GetFilesNumber(),
		)

		time.Sleep(time.Millisecond * 25)
	}

	_, _ = fmt.Fprintf(writer.Bypass(), "\u001B[37m[NEO4J]\u001B[0m Saving workflows \u001B[32m✓\u001B[0m\n")

	writer.Stop()
	fmt.Println()
}
