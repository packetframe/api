package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Target struct {
	DiscoveredLabels struct {
		Address        string `json:"__address__"`
		MetricsPath    string `json:"__metrics_path__"`
		Scheme         string `json:"__scheme__"`
		ScrapeInterval string `json:"__scrape_interval__"`
		ScrapeTimeout  string `json:"__scrape_timeout__"`
		Job            string `json:"job"`
		Node           string `json:"node,omitempty"`
	} `json:"discoveredLabels"`
	Labels struct {
		Instance string `json:"instance"`
		Job      string `json:"job"`
		Node     string `json:"node,omitempty"`
	} `json:"labels"`
	ScrapePool         string    `json:"scrapePool"`
	ScrapeUrl          string    `json:"scrapeUrl"`
	GlobalUrl          string    `json:"globalUrl"`
	LastError          string    `json:"lastError"`
	LastScrape         time.Time `json:"lastScrape"`
	LastScrapeDuration float64   `json:"lastScrapeDuration"`
	Health             string    `json:"health"`
	ScrapeInterval     string    `json:"scrapeInterval"`
	ScrapeTimeout      string    `json:"scrapeTimeout"`
}

type Resp struct {
	Status string `json:"status"`
	Data   struct {
		ActiveTargets  []Target `json:"activeTargets"`
		DroppedTargets []Target `json:"droppedTargets"`
	} `json:"data"`
}

// MonitorTargets handles a GET request to get node target status
func MonitorTargets(c *fiber.Ctx) error {
	ok, _, err := checkAdminUserAuth(c)
	if err != nil || !ok {
		return err
	}

	resp, err := http.Get("http://prometheus:9090/api/v1/targets")
	if err != nil {
		return internalServerError(c, err)
	}
	defer resp.Body.Close()
	var r Resp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return internalServerError(c, err)
	}

	nodes := 0
	var failedTargets []Target
	for _, target := range r.Data.ActiveTargets {
		if target.Labels.Job == "node" {
			nodes++
		}

		if target.Health != "up" {
			failedTargets = append(failedTargets, target)
		}
	}

	failedJobs := map[string]bool{}
	failedNodes := map[string]bool{}
	for _, target := range failedTargets {
		failedJobs[target.Labels.Job] = true
		failedNodes[target.Labels.Node] = true
	}

	plural := ""
	if len(failedJobs) != 1 {
		plural = "s"
	}
	out := fmt.Sprintf("%d nodes, %d targets, %d target errors across %d failed job%s\n", nodes, len(r.Data.ActiveTargets), len(failedTargets), len(failedJobs), plural)

	nodesStr := ""
	if len(failedNodes) == nodes {
		nodesStr = "all nodes"
	} else {
		for node := range failedNodes {
			nodesStr += node + ", "
		}
	}

	for job := range failedJobs {
		out += fmt.Sprintf("job %s failed on %s\n", job, nodesStr)
	}

	return c.Status(http.StatusOK).SendString(out)
}
