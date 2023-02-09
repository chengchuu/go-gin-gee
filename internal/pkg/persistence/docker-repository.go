package persistence

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	models "github.com/mazeyqian/go-gin-gee/internal/pkg/models/docker"
	"github.com/samber/lo"
)

type DockerRepository struct{}

var dockerRepository *DockerRepository

func GetDockerRepository() *DockerRepository {
	if dockerRepository == nil {
		dockerRepository = &DockerRepository{}
	}
	return dockerRepository
}

func (d *DockerRepository) GetTagName(namespace string, repository string, includedStr string) (string, error) {
	var tagName string
	var err error
	// Duplicated ↓
	// var dockerV2Tags *models.DockerV2Tags
	dockerV2Tags := &models.DockerV2Tags{}
	// https://registry.hub.docker.com/v2/repositories/mazeyqian/go-gin-gee/tags?page_size=100
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/%s/tags?page_size=20", namespace, repository)
	client := resty.New()
	_, err = client.R().
		SetResult(dockerV2Tags).
		Get(url)
	if err != nil {
		return tagName, err
	}
	// log.Println("  Body       :\n", resp)
	// log.Println("dockerV2Tags:", dockerV2Tags)
	// log.Println("dockerV2Tags.Results:", dockerV2Tags.Results)
	findNames := lo.Find[*models.DockerV2TagsResult](dockerV2Tags.Results, func(v models.DockerV2TagsResult) bool {
		// log.Println("lo.Substring(v.Name, -3, 3)", lo.Substring(v.Name, -3, 3))
		// log.Println("includedStr", includedStr)
		// log.Println("lo.Substring(v.Name, -3, 3) == includedStr", lo.Substring(v.Name, -3, 3) == includedStr)
		return lo.Substring(v.Name, -3, 3) == includedStr
	})
	// log.Println("findNames:", findNames)
	// log.Println("findNames len:", len(findNames))
	// log.Println("findNames[0]:", findNames[0])
	// log.Println("findNames[0] Name:", findNames[0].Name)
	return tagName, err
}
