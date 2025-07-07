package database

import (
	"app/pkg/git/model"
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"strings"
	"time"
)

// executeQueryNeo send the actual query, together with the content, to neo4j
func executeQueryNeo(query string, content map[string]any, driver neo4j.DriverWithContext, ctx context.Context) {
	if _, err := neo4j.ExecuteQuery(
		ctx, driver, query, content, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"),
	); err != nil {
		panic(err)
	}
}

func executeQueryWithRetNeo(query string, content map[string]any, driver neo4j.DriverWithContext, ctx context.Context) []*neo4j.Record {
	if result, err := neo4j.ExecuteQuery(
		ctx, driver, query, content, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"),
	); err != nil {
		panic(err)
	} else {
		return result.Records
	}
}

// addVersion sends the version nodes and relationships to neo4j
func addVersion(version model.Version, component string, commit string, date time.Time, driver neo4j.DriverWithContext, ctx context.Context) {
	versionName := version.GetVersionString()
	name := versionName

	if strings.HasPrefix(component, ".") {
		name = "local"
	}

	if strings.HasSuffix(component, ".yaml") || strings.HasSuffix(component, ".yml") {
		for _, value := range executeQueryWithRetNeo(
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

				executeQueryNeo(
					`MATCH (c:Commit {full_name: $commit})
					MATCH (v:Commit {full_name: $version})
					MERGE (c)-[:USES {times: $times}]->(v)`,
					map[string]any{
						"commit":  commit,
						"version": workflowName,
						"times":   version.GetUses(),
					},
					driver, ctx,
				)

				return
			}
		}
	} else {
		executeQueryNeo(
			`MATCH (c:Component {full_name: $component})
			MATCH (co:Commit {full_name: $commit})
			MERGE (v:Version {name: $name, full_name: $version, type: $type})
			MERGE (c)-[:HAS]->(v)
			MERGE (co)-[:USES {times: $times}]->(v)`,
			map[string]any{
				"component": component,
				"commit":    commit,
				"name":      name,
				"version":   fmt.Sprintf("%s/%s", component, name),
				"type":      version.GetVersionType(),
				"times":     version.GetUses(),
			},
			driver, ctx,
		)
	}
}

// addComponents sends the component nodes and relationships to neo4j
func addComponents(component model.Component, commit string, date time.Time, driver neo4j.DriverWithContext, ctx context.Context) {
	componentName := ""
	componentSplit := strings.Split(component.GetName(), "/")
	vendorName := componentSplit[0]

	if strings.HasSuffix(component.GetName(), ".yaml") || strings.HasSuffix(component.GetName(), ".yml") {
		vendorName = strings.Split(commit, "/")[0]
		fullName := strings.Join(strings.Split(commit, "/")[:2], "/") + "/" + componentSplit[len(componentSplit)-1]

		executeQueryNeo(
			`MATCH (w:Workflow {full_name: $workflow})
			MERGE (v:Vendor {name: $vendor})
			MERGE (v)-[:PUBLISHED]->(w)`,
			map[string]any{
				"workflow": fullName,
				"vendor":   vendorName,
			},
			driver, ctx,
		)

		for _, version := range component.GetHistory() {
			addVersion(*version, fullName, commit, date, driver, ctx)
		}

		return
	} else if strings.HasPrefix(component.GetName(), ".") {
		componentName = componentSplit[len(componentSplit)-1]
		vendorName = strings.Split(commit, "/")[0]
	} else if len(componentSplit) == 2 {
		componentName = componentSplit[1]
	} else {
		componentName = strings.Join(componentSplit[1:], "/")
	}

	executeQueryNeo(
		`MERGE (v:Vendor {name: $vendor})
		MERGE (c:Component {name: $component, full_name: $full, type: $type})
		MERGE (v)-[:PUBLISHED]->(c)`,
		map[string]any{
			"vendor":    vendorName,
			"component": componentName,
			"full":      component.GetName(),
			"type":      component.GetCategory(),
		},
		driver, ctx,
	)

	for _, version := range component.GetHistory() {
		addVersion(*version, component.GetName(), commit, date, driver, ctx)
	}
}

// addCommits sends the commit nodes and relationships to neo4j
func addCommits(commit model.Commit, workflow string, driver neo4j.DriverWithContext, ctx context.Context) {
	commitFull := fmt.Sprintf("%s/%s", workflow, commit.GetHash())
	content, _ := commit.GetContent(false)

	executeQueryNeo(
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

	executeQueryNeo(
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
}

// SendToNeo adds the given repository to neo4j
func SendToNeo(repository model.Repository, driver neo4j.DriverWithContext, ctx context.Context) {
	vendor := strings.Split(repository.GetName(), "/")[0]
	repo := strings.Split(repository.GetName(), "/")[1]

	executeQueryNeo(
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

	for _, workflow := range repository.GetFiles() {
		addWorkflows(workflow, repository.GetName(), driver, ctx)
	}
}
