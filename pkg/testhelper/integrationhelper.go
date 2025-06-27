package testhelper

import (
	"os"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.uber.org/zap"
)

type RetryFunc func(res *dockertest.Resource) error

func IsIntegration() bool {
	return os.Getenv("TEST_INTEGRATION") == "true"
}

func StartDockerPool() *dockertest.Pool {
	pool, err := dockertest.NewPool("")
	if err != nil {
		zap.S().Fatalf("Could not construct pool", zap.Error(err))
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		zap.S().Fatal("Could not connect to Docker", zap.Error(err))
	}
	return pool
}

func StartDockerInstance(pool *dockertest.Pool, image, tag string, retryFunc RetryFunc, env ...string) *dockertest.Resource {
	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: image,
		Tag:        tag,
		Env:        env,
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		zap.L().Fatal("Could not start resource")
	}

	if err := resource.Expire(120); err != nil {
		zap.S().Fatalln("couldn't set the resource expiration")
	}

	if err := pool.Retry(func() error {
		return retryFunc(resource)
	}); err != nil {
		zap.S().Fatalln("Couldn't connect to the resource")
	}
	return resource
}
