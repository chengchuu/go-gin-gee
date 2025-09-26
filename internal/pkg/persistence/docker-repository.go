package persistence

import (
	"fmt"
	"log"

	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/docker"
	"github.com/go-resty/resty/v2"
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

func (r *DockerRepository) GetTagName(namespace string, repository string, includedStr string) (string, error) {
	var tagName string
	var err error
	// Duplicated ↓
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
	res, ok := lo.Find(dockerV2Tags.Results, func(v models.DockerV2TagsResult) bool {
		return lo.Substring(v.Name, -3, 3) == includedStr
	})
	if !ok {
		return tagName, err
	}
	tagName = res.Name
	log.Println("findNames:", res)
	log.Println("findNames ok:", ok)
	log.Println("findNames name:", res.Name)
	return tagName, err
}
