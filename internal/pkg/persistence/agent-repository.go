package persistence

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/agent"
)

type AgentRepository struct{}

var agentRepository *AgentRepository

func GetAgentRepository() *AgentRepository {
	if agentRepository == nil {
		agentRepository = &AgentRepository{}
	}
	return agentRepository
}

func (r *AgentRepository) Mock(res *models.Response) (*models.ResponseData, error) {
	return &res.Data, nil
}

func (r *AgentRepository) Record(res *models.RecordRequestOrResponse) (string, error) {
	// Format the log file name
	var formattedTime string
	agentRecordsPath := config.GetConfig().Data.AgentRecordsPath
	if agentRecordsPath == "" {
		return "", fmt.Errorf("agent records path is not set")
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		formattedTime = time.Now().Format("15-04-05.000")
	} else {
		formattedTime = time.Now().In(loc).Format("15-04-05.000")
	}
	fileName := fmt.Sprintf("[%s]%s(%s).json",
		res.MethodOrStatusCode,
		strings.ReplaceAll(strings.ReplaceAll(res.URL, "/", "-"), ":", "-"),
		formattedTime,
	)
	filePath := fmt.Sprintf("%s/%s", agentRecordsPath, fileName)

	// Create the file and handle errors
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return "", fmt.Errorf("failed to create or open the file: %w", err)
	}
	defer file.Close()

	// Decode the URL-encoded data
	decodedData, err := url.QueryUnescape(res.Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode URL-encoded data: %w", err)
	}

	// Format the data with proper JSON indentation
	var formattedData interface{}
	if err := json.Unmarshal([]byte(decodedData), &formattedData); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}

	// Write the formatted JSON data to the file without escaping HTML characters
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")  // Enable pretty-printing with indentation
	encoder.SetEscapeHTML(false) // Disable HTML escaping for characters like <, >, &
	if err := encoder.Encode(formattedData); err != nil {
		return "", fmt.Errorf("failed to write JSON data to file: %w", err)
	}

	return filePath, nil
}
